/*
# Module: tools/parser.go
Parses LinkedDoc headers from Go source files.

## Linked Modules
(None - leaf module)

## Tags
tooling, parsing, linkedoc

## Exports
Parser, LinkedDocHeader, LinkedModule, NewParser

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "tools/parser.go" ;
    code:description "Parses LinkedDoc headers from Go source files" ;
    code:exports :Parser, :LinkedDocHeader, :NewParser ;
    code:tags "tooling", "parsing" .
<!-- End LinkedDoc RDF -->
*/
package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LinkedDocHeader represents a parsed LinkedDoc header
type LinkedDocHeader struct {
	Module        string
	Description   string
	LinkedModules []LinkedModule
	Tags          []string
	Exports       []string
	RDFMetadata   string
	FilePath      string
	LinesOfCode   int
	LastModified  time.Time
	FileHash      string
}

// LinkedModule represents a linked module reference
type LinkedModule struct {
	Name         string
	Path         string
	Relationship string
}

// Parser parses LinkedDoc headers from Go files
type Parser struct {
	verbose   bool
	cacheFile string
	cache     map[string]string // filepath -> file hash
}

var (
	// Regex patterns for parsing LinkedDoc headers
	modulePattern      = regexp.MustCompile(`^#\s+Module:\s+(.+)$`)
	descriptionPattern = regexp.MustCompile(`^([^#\[].*?)$`)
	linkPattern        = regexp.MustCompile(`^-\s+\[([^\]]+)\]\(([^\)]+)\):\s+(.+)$`)
	tagsPattern        = regexp.MustCompile(`^##\s+Tags\s*$`)
	exportsPattern     = regexp.MustCompile(`^##\s+Exports\s*$`)
	rdfStartPattern    = regexp.MustCompile(`<!--\s*LinkedDoc\s+RDF\s*-->`)
	rdfEndPattern      = regexp.MustCompile(`<!--\s*End\s+LinkedDoc\s+RDF\s*-->`)
)

// NewParser creates a new LinkedDoc parser
func NewParser(verbose bool) *Parser {
	return &Parser{
		verbose:   verbose,
		cacheFile: ".linkedoc_cache/file_hashes.txt",
		cache:     make(map[string]string),
	}
}

// ParseDirectory parses all Go files in a directory recursively
func (p *Parser) ParseDirectory(path string, incremental bool) ([]*LinkedDocHeader, error) {
	// Load cache for incremental builds
	if incremental {
		p.loadCache()
	}

	var headers []*LinkedDocHeader

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if info.IsDir() || !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		// Skip vendor and test files
		if strings.Contains(filePath, "/vendor/") || strings.HasSuffix(filePath, "_test.go") {
			return nil
		}

		// Check cache for incremental builds
		if incremental {
			fileHash, err := p.calculateFileHash(filePath)
			if err == nil && p.cache[filePath] == fileHash {
				if p.verbose {
					fmt.Printf("Skipping unchanged file: %s\n", filePath)
				}
				return nil
			}
		}

		// Parse the file
		header, err := p.ParseFile(filePath)
		if err != nil {
			// Not all files will have LinkedDoc headers - that's okay
			if p.verbose {
				fmt.Printf("No LinkedDoc header in: %s\n", filePath)
			}
			return nil
		}

		if p.verbose {
			fmt.Printf("Parsed: %s\n", filePath)
		}

		headers = append(headers, header)
		return nil
	})

	// Save cache for next incremental build
	if incremental {
		p.saveCache()
	}

	return headers, err
}

// ParseFile parses a single Go file for its LinkedDoc header
func (p *Parser) ParseFile(filePath string) (*LinkedDocHeader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	header := &LinkedDocHeader{
		FilePath:     filePath,
		LastModified: info.ModTime(),
	}

	// Count lines of code
	lineCount := 0
	tempFile, _ := os.Open(filePath)
	scanner := bufio.NewScanner(tempFile)
	for scanner.Scan() {
		lineCount++
	}
	tempFile.Close()
	header.LinesOfCode = lineCount

	// Calculate file hash
	hash, _ := p.calculateFileHash(filePath)
	header.FileHash = hash

	// Parse the header comment
	scanner = bufio.NewScanner(file)
	inComment := false
	inRDF := false
	section := ""
	var rdfLines []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for comment start
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			continue
		}

		// Check for comment end
		if strings.HasSuffix(trimmed, "*/") {
			break
		}

		if !inComment {
			continue
		}

		// Remove leading comment markers
		trimmed = strings.TrimPrefix(trimmed, "*")
		trimmed = strings.TrimSpace(trimmed)

		// Check for RDF section
		if rdfStartPattern.MatchString(trimmed) {
			inRDF = true
			continue
		}
		if rdfEndPattern.MatchString(trimmed) {
			inRDF = false
			continue
		}
		if inRDF {
			rdfLines = append(rdfLines, line)
			continue
		}

		// Parse module name
		if matches := modulePattern.FindStringSubmatch(trimmed); matches != nil {
			header.Module = strings.TrimSpace(matches[1])
			continue
		}

		// Parse section headers
		if strings.HasPrefix(trimmed, "## Linked Modules") {
			section = "links"
			continue
		}
		if tagsPattern.MatchString(trimmed) {
			section = "tags"
			continue
		}
		if exportsPattern.MatchString(trimmed) {
			section = "exports"
			continue
		}

		// Parse section content
		switch section {
		case "":
			// This is the description (first non-header line)
			if header.Description == "" && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				header.Description = trimmed
			}

		case "links":
			if matches := linkPattern.FindStringSubmatch(trimmed); matches != nil {
				header.LinkedModules = append(header.LinkedModules, LinkedModule{
					Name:         strings.TrimSpace(matches[1]),
					Path:         strings.TrimSpace(matches[2]),
					Relationship: strings.TrimSpace(matches[3]),
				})
			}

		case "tags":
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				// Parse comma-separated tags
				tags := strings.Split(trimmed, ",")
				for _, tag := range tags {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						header.Tags = append(header.Tags, tag)
					}
				}
				section = "" // Tags are on one line
			}

		case "exports":
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				// Parse comma-separated exports
				exports := strings.Split(trimmed, ",")
				for _, exp := range exports {
					exp = strings.TrimSpace(exp)
					if exp != "" {
						header.Exports = append(header.Exports, exp)
					}
				}
				section = "" // Exports are on one line
			}
		}
	}

	// Store RDF metadata
	if len(rdfLines) > 0 {
		header.RDFMetadata = strings.Join(rdfLines, "\n")
	}

	// Validate we found a LinkedDoc header
	if header.Module == "" {
		return nil, fmt.Errorf("no LinkedDoc header found")
	}

	return header, nil
}

// calculateFileHash calculates MD5 hash of file for caching
func (p *Parser) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// loadCache loads the file hash cache
func (p *Parser) loadCache() error {
	file, err := os.Open(p.cacheFile)
	if err != nil {
		return err // Cache doesn't exist yet
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) == 2 {
			p.cache[parts[0]] = parts[1]
		}
	}

	return scanner.Err()
}

// saveCache saves the file hash cache
func (p *Parser) saveCache() error {
	// Create cache directory
	dir := filepath.Dir(p.cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(p.cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for path, hash := range p.cache {
		fmt.Fprintf(file, "%s\t%s\n", path, hash)
	}

	return nil
}
