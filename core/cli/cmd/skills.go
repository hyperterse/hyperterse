package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hyperterse/hyperterse/core/cli/internal"
	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/spf13/cobra"
)

var (
	skillOutput string
	skillName   string
)

// skillsCmd represents the skills command
var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Generate an Agent Skills compatible archive",
	Long: `Generate a downloadable Agent Skills compatible archive (.zip) from your configuration.
The archive will contain a SKILL.md file with YAML frontmatter and documentation.`,
	RunE: generateSkills,
}

func init() {
	generateCmd.AddCommand(skillsCmd)

	skillsCmd.Flags().StringVarP(&skillOutput, "output", "o", "skill.zip", "Output path for the skills archive")
	skillsCmd.Flags().StringVar(&skillName, "name", "", "Skill name (default: derived from config or 'hyperterse-skill')")
}

func generateSkills(cmd *cobra.Command, args []string) error {
	log := logger.New("generate")

	// Load config
	model, err := internal.LoadConfig(configFile)
	if err != nil {
		log.PrintError("Error loading config", err)
		os.Exit(1)
	}

	// Validate model
	if err := parser.Validate(model); err != nil {
		if validationErr, ok := err.(*parser.ValidationErrors); ok {
			log.PrintValidationErrors(validationErr.Errors)
		} else {
			log.PrintError("Validation Error", err)
		}
		os.Exit(1)
	}

	// Determine skill name
	if skillName == "" {
		// Try to derive from config file name
		baseName := strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile))
		skillName = strings.ToLower(strings.ReplaceAll(baseName, "_", "-"))
		if skillName == "" {
			skillName = "hyperterse-skill"
		}
	}

	// Validate skill name (must be lowercase, alphanumeric and hyphens only, max 64 chars)
	skillName = strings.ToLower(skillName)
	skillName = strings.ReplaceAll(skillName, "_", "-")
	if len(skillName) > 64 {
		skillName = skillName[:64]
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "hyperterse-skill-*")
	if err != nil {
		log.PrintError("Failed to create temporary directory", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	skillDir := filepath.Join(tmpDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		log.PrintError("Failed to create skill directory", err)
		os.Exit(1)
	}

	// Generate SKILL.md
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if err := generateSkillMarkdown(skillMdPath, model, skillName); err != nil {
		log.PrintError("Failed to generate SKILL.md", err)
		os.Exit(1)
	}

	// Create zip archive
	if err := createZipArchive(skillDir, skillOutput); err != nil {
		log.PrintError("Failed to create archive", err)
		os.Exit(1)
	}

	log.PrintSuccess(fmt.Sprintf("Skills archive generated: %s", skillOutput))
	return nil
}

func generateSkillMarkdown(path string, model *hyperterse.Model, name string) error {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", name))

	// Generate description
	description := fmt.Sprintf("Execute database queries via Hyperterse runtime. Provides access to %d query endpoint(s)", len(model.Queries))
	if len(model.Queries) > 0 {
		queryNames := make([]string, len(model.Queries))
		for i, q := range model.Queries {
			queryNames[i] = q.Name
		}
		description += fmt.Sprintf(": %s", strings.Join(queryNames, ", "))
	}
	sb.WriteString(fmt.Sprintf("description: %s\n", description))
	sb.WriteString("---\n\n")

	// Markdown body
	// Title case conversion
	titleParts := strings.Split(strings.ReplaceAll(name, "-", " "), " ")
	for i, part := range titleParts {
		if len(part) > 0 {
			titleParts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	title := strings.Join(titleParts, " ")
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString("This Skill provides access to Hyperterse database query endpoints.\n\n")

	// Available Queries
	if len(model.Queries) > 0 {
		sb.WriteString("## Available Queries\n\n")
		for _, query := range model.Queries {
			sb.WriteString(fmt.Sprintf("### %s\n\n", query.Name))
			if query.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", query.Description))
			}

			// Inputs
			if len(query.Inputs) > 0 {
				sb.WriteString("**Inputs:**\n\n")
				for _, input := range query.Inputs {
					required := "required"
					if input.Optional {
						required = "optional"
					}
					sb.WriteString(fmt.Sprintf("- `%s` (%s): %s - %s\n", input.Name, input.Type.String(), required, input.Description))
				}
				sb.WriteString("\n")
			}

			// Outputs
			if len(query.Data) > 0 {
				sb.WriteString("**Outputs:**\n\n")
				for _, data := range query.Data {
					sb.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", data.Name, data.Type.String(), data.Description))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Usage
	sb.WriteString("## Usage\n\n")
	sb.WriteString("Use the Hyperterse runtime API to execute queries. Each query is available as a POST endpoint.\n\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func createZipArchive(sourceDir, outputPath string) error {
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipEntry, file)
		return err
	})
}
