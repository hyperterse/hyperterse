package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	initOutputFile string
)

const (
	scaffoldRootDir  = "app"
	scaffoldToolsDir = "tools"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:          "init [folder-name]",
	Short:        "Initialize a new Hyperterse configuration file",
	RunE:         runInit,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initOutputFile, "output", "o", "", "Output file path for the configuration (default: <folder>/.hyperterse)")
}

func runInit(cmd *cobra.Command, args []string) error {
	var folderName string
	if len(args) > 0 {
		folderName = strings.TrimSpace(args[0])
	}
	if folderName == "" {
		fmt.Fprint(os.Stderr, "Project folder name: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		folderName = strings.TrimSpace(line)
		if folderName == "" {
			return fmt.Errorf("folder name is required. Use: hyperterse init <folder-name>")
		}
	}

	baseDir := folderName
	if baseDir == "." {
		baseDir = ""
	}
	if baseDir != "" {
		baseDir = filepath.Clean(baseDir)
	}

	outputPath := initOutputFile
	if outputPath == "" {
		if baseDir != "" {
			outputPath = filepath.Join(baseDir, ".hyperterse")
		} else {
			outputPath = ".hyperterse"
		}
	} else if baseDir != "" && !filepath.IsAbs(outputPath) && filepath.Dir(outputPath) == "." {
		outputPath = filepath.Join(baseDir, outputPath)
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file '%s' already exists. Use a different filename or remove the existing file", outputPath)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Generate the config content
	configContent := generateConfigTemplate()

	// Write the file
	if err := os.WriteFile(outputPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	scaffoldBase := filepath.Dir(outputPath)
	appToolDir := filepath.Join(scaffoldBase, scaffoldRootDir, scaffoldToolsDir, "hello-world")
	if err := os.MkdirAll(appToolDir, 0755); err != nil {
		return fmt.Errorf("failed to create app tool directory: %w", err)
	}

	toolConfig := `description: "Hello world tool"
handler: "./handler.ts"
inputs:
  name:
    type: string
    description: "Name to greet."
auth:
  plugin: allow_all
`
	if err := os.WriteFile(filepath.Join(appToolDir, "config.terse"), []byte(toolConfig), 0644); err != nil {
		return fmt.Errorf("failed to write tool config.terse: %w", err)
	}

	handlerTS := `export default function handler(payload: {
  inputs: Record<string, unknown>
  tool: string
}) {
  const name = String(payload.inputs?.name ?? "world");
  return { message: ` + "`Hello, ${name}!`" + ` };
}
`
	if err := os.WriteFile(filepath.Join(appToolDir, "handler.ts"), []byte(handlerTS), 0644); err != nil {
		return fmt.Errorf("failed to write tool handler.ts: %w", err)
	}

	skillDir := filepath.Join(scaffoldBase, ".agents", "skills", "hyperterse-docs")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create .agents/skills/hyperterse-docs directory: %w", err)
	}
	skillContent := `---
name: hyperterse-docs
description: Hyperterse LLM integration docs. Use when building tools, adapters, or MCP integrations with Hyperterse.
---

This entire project is built with Hyperterse - the declarative and performant MCP framework.

You are an expert at building tools, adapters, and MCP integrations with Hyperterse. You are well versed with

# Hyperterse

When working with Hyperterse tools, adapters, or MCP integrations, read the latest documentation from:

**https://docs.hyperterse.com/llms.txt**

Fetch and use this content for accurate schema, configuration, and API reference.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to write .agents/skills/hyperterse-docs/SKILL.md: %w", err)
	}

	agentSkillDir := filepath.Join(scaffoldBase, ".agents", "skills", "hyperterse-agents")
	if err := os.MkdirAll(agentSkillDir, 0755); err != nil {
		return fmt.Errorf("failed to create .agents/skills/hyperterse-agents directory: %w", err)
	}
	agentSkillContent := `---
name: hyperterse-agents
description: Build declarative agents with Hyperterse. Use when creating app/agents/*/config.terse, configuring agents in .hyperterse, setting tool permissions, and testing A2A agent endpoints mounted at /agent/{name}.
---

This project uses Hyperterse for MCP tools and declarative agent runtimes.

When asked to create or update Hyperterse agents, follow this workflow:

## 1) Read source-of-truth docs first

Fetch and use:

- https://docs.hyperterse.com/llms.txt

Prioritize these docs:

- /agents/quickstart
- /agents/tool-access
- /agents/model-providers
- /reference/agent-config
- /reference/root-config

Do not guess schema fields when docs are available.

## 2) Configure root agent defaults in .hyperterse

Use conservative defaults:

    agents:
      directory: agents
      tool_access:
        mode: allow_none

allow_none at root keeps permissions explicit and safe.

## 3) Create an agent config

Add app/agents/<agent-name>/config.terse:

    name: <agent-name>
    description: "<short purpose>"
    instruction: "<what the agent should do>"
    model:
      provider: <provider>
      model: <model-name>
    tool_access:
      mode: allow_list
      tools:
        - <tool-name>

Prefer allow_list for least-privilege access.

## 4) Validate and run

- hyperterse validate
- hyperterse start

## 5) Verify runtime endpoints

- Public agent card: GET /agent/<agent-name>/.well-known/agent-card.json
- JSON-RPC endpoint: POST /agent/<agent-name>
- Streaming endpoint: POST /agent/<agent-name> with method SendStreamingMessage
- Task endpoints: POST /agent/<agent-name> with methods GetTask, CancelTask, and SubscribeToTask

Rules:

- Keep instructions specific and task-oriented.
- Avoid allow_all unless explicitly requested.
- If a model/provider fails, check required environment variables.
`
	if err := os.WriteFile(filepath.Join(agentSkillDir, "SKILL.md"), []byte(agentSkillContent), 0644); err != nil {
		return fmt.Errorf("failed to write .agents/skills/hyperterse-agents/SKILL.md: %w", err)
	}

	displayBase := scaffoldBase
	if displayBase == "." {
		displayBase = ""
	}
	relToolConfig := filepath.Join(scaffoldRootDir, scaffoldToolsDir, "hello-world", "config.terse")
	relDocsSkill := filepath.Join(".agents", "skills", "hyperterse-docs", "SKILL.md")
	relAgentsSkill := filepath.Join(".agents", "skills", "hyperterse-agents", "SKILL.md")
	if displayBase != "" {
		relToolConfig = filepath.Join(displayBase, relToolConfig)
		relDocsSkill = filepath.Join(displayBase, relDocsSkill)
		relAgentsSkill = filepath.Join(displayBase, relAgentsSkill)
	}

	fmt.Printf("✓ Created configuration file: %s\n", outputPath)
	fmt.Printf("✓ Created tool config: %s\n", relToolConfig)
	fmt.Printf("✓ Created agent skill: %s\n", relDocsSkill)
	fmt.Printf("✓ Created agent skill: %s\n", relAgentsSkill)
	editPath := filepath.Join(scaffoldBase, scaffoldRootDir)
	if editPath == filepath.Join(".", scaffoldRootDir) {
		editPath = scaffoldRootDir
	}
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s and add tools under %s/%s\n", outputPath, editPath, scaffoldToolsDir)
	fmt.Printf("  2. Add adapters in %s/adapters/ when you need database connections\n", editPath)
	fmt.Printf("  3. Run: hyperterse start")

	return nil
}

func generateConfigTemplate() string {
	return `name: my-service

server:
  port: 8080
  log_level: 3

root: app

tools:
  directory: tools

`
}
