/*
# Module: tools/linkedoc_build.go
Main entry point for LinkedDoc build tool - parses, validates, and generates indices.

## Linked Modules
- [parser](parser.go): LinkedDoc header parser
- [validator](validator.go): Link and schema validator
- [index_generator](index_generator.go): JSON index generator

## Tags
tooling, validation, linkedoc

## Exports
main

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "tools/linkedoc_build.go" ;
    code:description "Main entry point for LinkedDoc build tool" ;
    code:dependsOn <parser.go>, <validator.go>, <index_generator.go> ;
    code:exports :main ;
    code:tags "tooling", "validation" .
<!-- End LinkedDoc RDF -->
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const version = "1.0.0"

var (
	validateFlag     = flag.Bool("validate", false, "Validate all LinkedDoc headers and links")
	generateIndex    = flag.Bool("generate-index", false, "Generate JSON index for AI navigation")
	pathFlag         = flag.String("path", ".", "Path to scan (default: current directory)")
	incrementalFlag  = flag.Bool("incremental", false, "Only process changed files")
	verboseFlag      = flag.Bool("verbose", false, "Verbose output")
	versionFlag      = flag.Bool("version", false, "Show version")
	outputFlag       = flag.String("output", "docs/linkedoc_index.json", "Output path for JSON index")
)

func main() {
	flag.Parse()

	// Show version
	if *versionFlag {
		fmt.Printf("linkedoc_build version %s\n", version)
		os.Exit(0)
	}

	// Show usage if no flags
	if !*validateFlag && !*generateIndex {
		flag.Usage()
		fmt.Println("\nExamples:")
		fmt.Println("  linkedoc_build --validate")
		fmt.Println("  linkedoc_build --generate-index")
		fmt.Println("  linkedoc_build --validate --path location-tracker/handlers")
		fmt.Println("  linkedoc_build --incremental --generate-index")
		os.Exit(1)
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(*pathFlag)
	if err != nil {
		log.Fatalf("Failed to resolve path: %v", err)
	}

	if *verboseFlag {
		log.Printf("Scanning path: %s", absPath)
	}

	// Parse all Go files
	parser := NewParser(*verboseFlag)
	headers, err := parser.ParseDirectory(absPath, *incrementalFlag)
	if err != nil {
		log.Fatalf("Failed to parse files: %v", err)
	}

	if *verboseFlag {
		log.Printf("Parsed %d LinkedDoc headers", len(headers))
	}

	if len(headers) == 0 {
		log.Println("⚠️  No LinkedDoc headers found")
		if !*validateFlag && *generateIndex {
			log.Println("Creating empty index...")
			if err := generateEmptyIndex(*outputFlag); err != nil {
				log.Fatalf("Failed to create empty index: %v", err)
			}
			log.Println("✓ Empty index created")
		}
		os.Exit(0)
	}

	// Validate if requested
	if *validateFlag {
		if *verboseFlag {
			log.Println("Starting validation...")
		}

		validator := NewValidator(*verboseFlag)
		errors := validator.Validate(headers)

		if len(errors) == 0 {
			fmt.Println("✓ All LinkedDoc headers are valid")
			fmt.Printf("  - %d modules validated\n", len(headers))
			fmt.Printf("  - 0 broken links\n")
			fmt.Printf("  - 0 schema violations\n")
		} else {
			fmt.Println("✗ Validation failed:")
			for _, err := range errors {
				fmt.Printf("  - %v\n", err)
			}
			os.Exit(1)
		}
	}

	// Generate index if requested
	if *generateIndex {
		if *verboseFlag {
			log.Println("Generating JSON index...")
		}

		generator := NewIndexGenerator(*verboseFlag)
		if err := generator.Generate(headers, *outputFlag); err != nil {
			log.Fatalf("Failed to generate index: %v", err)
		}

		fmt.Printf("✓ JSON index generated: %s\n", *outputFlag)
		fmt.Printf("  - %d modules indexed\n", len(headers))
	}
}

func generateEmptyIndex(outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write empty JSON object
	return os.WriteFile(outputPath, []byte("{}"), 0644)
}
