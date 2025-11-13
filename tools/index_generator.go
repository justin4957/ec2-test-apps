/*
# Module: tools/index_generator.go
Generates JSON index for AI-optimized code navigation.

## Linked Modules
- [parser](parser.go): Uses LinkedDocHeader type

## Tags
tooling, indexing, linkedoc, json

## Exports
IndexGenerator, NewIndexGenerator, Generate

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "tools/index_generator.go" ;
    code:description "Generates JSON index for AI-optimized navigation" ;
    code:dependsOn <parser.go> ;
    code:exports :IndexGenerator, :NewIndexGenerator ;
    code:tags "tooling", "indexing", "json" .
<!-- End LinkedDoc RDF -->
*/
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModuleIndex represents the JSON structure for a module
type ModuleIndex struct {
	Description   string         `json:"description"`
	FilePath      string         `json:"file_path"`
	LinksTo       []LinkRef      `json:"links_to"`
	Exports       []string       `json:"exports"`
	Tags          []string       `json:"tags"`
	LinesOfCode   int            `json:"lines_of_code"`
	LastModified  string         `json:"last_modified"`
	HasRDF        bool           `json:"has_rdf"`
}

// LinkRef represents a linked module reference in the index
type LinkRef struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Relationship string `json:"relationship"`
}

// IndexGenerator generates JSON index from LinkedDoc headers
type IndexGenerator struct {
	verbose bool
}

// NewIndexGenerator creates a new index generator
func NewIndexGenerator(verbose bool) *IndexGenerator {
	return &IndexGenerator{
		verbose: verbose,
	}
}

// Generate generates a JSON index from parsed headers
func (g *IndexGenerator) Generate(headers []*LinkedDocHeader, outputPath string) error {
	index := make(map[string]ModuleIndex)

	// Convert headers to index format
	for _, header := range headers {
		// Convert linked modules to link refs
		var linksTo []LinkRef
		for _, link := range header.LinkedModules {
			linksTo = append(linksTo, LinkRef{
				Name:         link.Name,
				Path:         link.Path,
				Relationship: link.Relationship,
			})
		}

		// Create module index entry
		moduleIndex := ModuleIndex{
			Description:  header.Description,
			FilePath:     g.relativizePath(header.FilePath),
			LinksTo:      linksTo,
			Exports:      header.Exports,
			Tags:         header.Tags,
			LinesOfCode:  header.LinesOfCode,
			LastModified: header.LastModified.Format("2006-01-02T15:04:05Z"),
			HasRDF:       header.RDFMetadata != "",
		}

		// Use module name as key
		index[header.Module] = moduleIndex

		if g.verbose {
			fmt.Printf("  Indexed: %s\n", header.Module)
		}
	}

	// Generate statistics
	stats := g.generateStatistics(headers)

	// Create full index with metadata
	fullIndex := map[string]interface{}{
		"_metadata": map[string]interface{}{
			"generated_at":   fmt.Sprintf("%s", headers[0].LastModified.Format("2006-01-02T15:04:05Z")),
			"module_count":   len(headers),
			"total_loc":      stats.TotalLOC,
			"avg_loc":        stats.AvgLOC,
			"modules_with_rdf": stats.ModulesWithRDF,
		},
		"modules": index,
	}

	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write JSON to file
	data, err := json.MarshalIndent(fullIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	if g.verbose {
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("  Total modules: %d\n", len(headers))
		fmt.Printf("  Total LOC: %d\n", stats.TotalLOC)
		fmt.Printf("  Average LOC: %d\n", stats.AvgLOC)
		fmt.Printf("  Modules with RDF: %d\n", stats.ModulesWithRDF)
	}

	return nil
}

// Statistics holds index statistics
type Statistics struct {
	TotalLOC       int
	AvgLOC         int
	ModulesWithRDF int
}

// generateStatistics calculates statistics from headers
func (g *IndexGenerator) generateStatistics(headers []*LinkedDocHeader) Statistics {
	stats := Statistics{}

	for _, header := range headers {
		stats.TotalLOC += header.LinesOfCode
		if header.RDFMetadata != "" {
			stats.ModulesWithRDF++
		}
	}

	if len(headers) > 0 {
		stats.AvgLOC = stats.TotalLOC / len(headers)
	}

	return stats
}

// relativizePath makes a path relative to the project root
func (g *IndexGenerator) relativizePath(absPath string) string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return absPath
	}

	// Make path relative to cwd
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return absPath
	}

	// Convert backslashes to forward slashes for consistency
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	return relPath
}

// GenerateDependencyGraph generates a visual dependency graph (future enhancement)
func (g *IndexGenerator) GenerateDependencyGraph(headers []*LinkedDocHeader, format string) error {
	// TODO: Generate Mermaid, GraphViz, or D3.js dependency graph
	// For now, this is a placeholder for future enhancement
	return fmt.Errorf("dependency graph generation not yet implemented")
}

// GenerateTagIndex generates a tag-based index (future enhancement)
func (g *IndexGenerator) GenerateTagIndex(headers []*LinkedDocHeader) map[string][]string {
	tagIndex := make(map[string][]string)

	for _, header := range headers {
		for _, tag := range header.Tags {
			tagIndex[tag] = append(tagIndex[tag], header.Module)
		}
	}

	return tagIndex
}
