/*
# Module: tools/validator.go
Validates LinkedDoc headers and links.

## Linked Modules
- [parser](parser.go): Uses LinkedDocHeader type

## Tags
tooling, validation, linkedoc

## Exports
Validator, NewValidator, Validate

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "tools/validator.go" ;
    code:description "Validates LinkedDoc headers and links" ;
    code:dependsOn <parser.go> ;
    code:exports :Validator, :NewValidator ;
    code:tags "tooling", "validation" .
<!-- End LinkedDoc RDF -->
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validator validates LinkedDoc headers
type Validator struct {
	verbose bool
	errors  []error
}

// NewValidator creates a new validator
func NewValidator(verbose bool) *Validator {
	return &Validator{
		verbose: verbose,
		errors:  []error{},
	}
}

// Validate validates all LinkedDoc headers and returns errors
func (v *Validator) Validate(headers []*LinkedDocHeader) []error {
	v.errors = []error{}

	for _, header := range headers {
		v.validateHeader(header)
		v.validateLinks(header)
		v.validateTags(header)
		v.validateRDF(header)
	}

	return v.errors
}

// validateHeader validates basic header requirements
func (v *Validator) validateHeader(header *LinkedDocHeader) {
	// Module name is required
	if header.Module == "" {
		v.addError(header.FilePath, "missing module name")
		return
	}

	// Description is required
	if header.Description == "" {
		v.addError(header.FilePath, "missing description")
	}

	// Description should be concise (one line, under 100 chars)
	if len(header.Description) > 150 {
		v.addWarning(header.FilePath, fmt.Sprintf("description is too long (%d chars, recommend <100)", len(header.Description)))
	}

	// Should have at least one export (unless it's a main.go)
	if len(header.Exports) == 0 && !strings.HasSuffix(header.FilePath, "main.go") {
		v.addWarning(header.FilePath, "no exports listed (is this intentional?)")
	}

	// Should have at least one tag
	if len(header.Tags) == 0 {
		v.addWarning(header.FilePath, "no tags specified")
	}

	if v.verbose {
		fmt.Printf("  Validated header structure for: %s\n", header.Module)
	}
}

// validateLinks validates that all linked modules exist
func (v *Validator) validateLinks(header *LinkedDocHeader) {
	baseDir := filepath.Dir(header.FilePath)

	for _, link := range header.LinkedModules {
		// Resolve relative path
		linkedPath := filepath.Join(baseDir, link.Path)
		linkedPath = filepath.Clean(linkedPath)

		// Check if file exists
		if _, err := os.Stat(linkedPath); os.IsNotExist(err) {
			v.addError(header.FilePath, fmt.Sprintf("broken link to '%s' (resolved to: %s)", link.Name, linkedPath))
		} else if v.verbose {
			fmt.Printf("  Link OK: %s -> %s\n", link.Name, link.Path)
		}

		// Validate relationship description is not empty
		if link.Relationship == "" {
			v.addWarning(header.FilePath, fmt.Sprintf("link to '%s' has no relationship description", link.Name))
		}
	}
}

// validateTags validates that tags are from the known taxonomy
func (v *Validator) validateTags(header *LinkedDocHeader) {
	// Define known tags from the taxonomy
	knownTags := map[string]bool{
		// Functional
		"http":           true,
		"storage":        true,
		"api-client":     true,
		"business-logic": true,
		"data-types":     true,
		"middleware":     true,
		"auth":           true,
		"validation":     true,
		// Domain
		"location":   true,
		"errors":     true,
		"tips":       true,
		"commercial": true,
		"social":     true,
		"cryptogram": true,
		// Technical
		"dynamodb":      true,
		"cache":         true,
		"rate-limiting": true,
		"websocket":     true,
		"rdf":           true,
		"solid":         true,
		// Additional common tags
		"api":         true,
		"tracking":    true,
		"search":      true,
		"geolocation": true,
		"parsing":     true,
		"tooling":     true,
		"linkedoc":    true,
	}

	for _, tag := range header.Tags {
		if !knownTags[tag] {
			v.addWarning(header.FilePath, fmt.Sprintf("unknown tag '%s' (not in taxonomy)", tag))
		}
	}

	if v.verbose && len(header.Tags) > 0 {
		fmt.Printf("  Tags OK: %v\n", header.Tags)
	}
}

// validateRDF validates basic RDF structure
func (v *Validator) validateRDF(header *LinkedDocHeader) {
	if header.RDFMetadata == "" {
		v.addWarning(header.FilePath, "no RDF metadata found")
		return
	}

	// Check for required RDF elements
	rdf := header.RDFMetadata

	if !strings.Contains(rdf, "@prefix code:") {
		v.addError(header.FilePath, "RDF missing @prefix code: declaration")
	}

	if !strings.Contains(rdf, "code:name") {
		v.addError(header.FilePath, "RDF missing code:name property")
	}

	if !strings.Contains(rdf, "code:description") {
		v.addError(header.FilePath, "RDF missing code:description property")
	}

	// Check that module name in RDF matches header
	if strings.Contains(rdf, "code:name") && strings.Contains(rdf, fmt.Sprintf("\"%s\"", header.Module)) {
		if v.verbose {
			fmt.Printf("  RDF module name matches header\n")
		}
	} else {
		v.addWarning(header.FilePath, "RDF module name may not match header")
	}

	if v.verbose {
		fmt.Printf("  RDF structure OK\n")
	}
}

// addError adds a validation error
func (v *Validator) addError(filePath, message string) {
	v.errors = append(v.errors, fmt.Errorf("%s: %s", filePath, message))
}

// addWarning adds a validation warning
func (v *Validator) addWarning(filePath, message string) {
	if v.verbose {
		fmt.Printf("  ⚠️  Warning: %s: %s\n", filePath, message)
	}
	// For now, don't add warnings to error list (can be configurable later)
}

// CheckCircularDependencies checks for circular dependencies (future enhancement)
func (v *Validator) CheckCircularDependencies(headers []*LinkedDocHeader) []error {
	// Build dependency graph
	graph := make(map[string][]string)
	for _, header := range headers {
		for _, link := range header.LinkedModules {
			graph[header.Module] = append(graph[header.Module], link.Name)
		}
	}

	// TODO: Implement cycle detection algorithm
	// For now, just return empty (future enhancement)

	return []error{}
}
