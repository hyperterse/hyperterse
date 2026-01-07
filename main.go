package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/parser"
	"github.com/hyperterse/hyperterse/core/pb"
	"github.com/hyperterse/hyperterse/core/runtime"
)

var (
	// log is the logger instance for the main package
	log = logger.New("main")
)

func main() {
	filePath := flag.String("file", "", "Path to the configuration file (.hyperterse, .yaml, or .yml)")
	flag.Parse()

	if *filePath == "" {
		log.Println("Please provide a file path using -file")
		os.Exit(1)
	}

	content, err := os.ReadFile(*filePath)
	if err != nil {
		log.PrintError("Error reading file", err)
		os.Exit(1)
	}

	var model *pb.Model

	// Determine parser based on file extension
	if strings.HasSuffix(*filePath, ".yaml") || strings.HasSuffix(*filePath, ".yml") {
		model, err = parser.ParseYAML(content)
		if err != nil {
			log.PrintError("Config Error", err)
			os.Exit(1)
		}
	} else {
		// Default to DSL parser for .hyperterse files
		p := parser.NewParser(string(content))
		model, err = p.Parse()
		if err != nil {
			log.PrintError("Parsing Error", err)
			os.Exit(1)
		}
	}

	// Log parsed adapters
	log.Println("Parsed Configuration:")
	if len(model.Adapters) > 0 {
		log.Println("\tAdapters:")
		for _, adapter := range model.Adapters {
			log.Printf("\t  - Name: %s, Connector: %s", adapter.Name, adapter.Connector.String())
		}
		log.Println("")
	} else {
		log.Println("\tAdapters: (none)")
		log.Println("")
	}

	// Log parsed queries
	if len(model.Queries) > 0 {
		log.Println("\tQueries:")
		for _, query := range model.Queries {
			log.Printf("\t  - Name: %s", query.Name)
			if query.Description != "" {
				log.Printf("\t    Description: %s", query.Description)
			}
			if len(query.Use) > 0 {
				log.Printf("\t    Uses: %s", strings.Join(query.Use, ", "))
			}
			if len(query.Inputs) > 0 {
				inputNames := make([]string, 0, len(query.Inputs))
				for _, input := range query.Inputs {
					optional := ""
					if input.Optional {
						optional = "\t (optional)"
					}
					inputNames = append(inputNames, fmt.Sprintf("%s:%s%s", input.Name, input.Type, optional))
				}
				log.Printf("\t    Inputs: %s", strings.Join(inputNames, ", "))
			} else {
				log.Printf("\t    Inputs: (none)")
			}
			if len(query.Data) > 0 {
				dataNames := make([]string, 0, len(query.Data))
				for _, data := range query.Data {
					dataNames = append(dataNames, fmt.Sprintf("%s:%s", data.Name, data.Type))
				}
				log.Printf("\t    Outputs: %s", strings.Join(dataNames, ", "))
			} else {
				log.Printf("\t    Outputs: (none)")
			}
		}
		log.Println("")
	} else {
		log.Println("\tQueries: (none)")
		log.Println("")
	}

	if err := parser.Validate(model); err != nil {
		// Check if it's a ValidationErrors type to format it prettily
		if validationErr, ok := err.(*parser.ValidationErrors); ok {
			// Iterate over all validation errors and format them
			log.PrintValidationErrors(validationErr.Errors)
		} else {
			log.PrintError("Validation Error", err)
		}
		os.Exit(1)
	}

	log.PrintSuccess("Validation successful! ")

	log.Println("Starting runtime")
	// Start the runtime server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Runtime initialization")
	rt, err := runtime.NewRuntime(model, port)
	if err != nil {
		log.PrintError("Failed to create runtime", err)
		os.Exit(1)
	}
	log.PrintSuccess("Runtime initialized")

	log.Println("Starting runtime server")
	if err := rt.Start(); err != nil {
		log.PrintError("Runtime error", err)
		os.Exit(1)
	}
	log.PrintSuccess("Runtime server started")
}
