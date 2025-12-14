package analysis

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

// DuplicateLocation represents where a duplicate block is found
type DuplicateLocation struct {
	FilePath  string
	FuncName  string
	StartLine int
	EndLine   int
}

// ExactDuplicationDetector specific for hash-based matching
type ExactDuplicationDetector struct {
	// hash -> locations
	hashes map[string][]DuplicateLocation
}

// NewExactDuplicationDetector creates a new detector
func NewExactDuplicationDetector() *ExactDuplicationDetector {
	return &ExactDuplicationDetector{
		hashes: make(map[string][]DuplicateLocation),
	}
}

// AddFunction adds a function to the index for duplicate checking
func (d *ExactDuplicationDetector) AddFunction(file string, fn FunctionInfo) {
	if fn.Body == "" {
		return
	}
	
	// Normalize: remove whitespace to catch formatting differences
	hash := computeCodeHash(fn.Body)
	
	loc := DuplicateLocation{
		FilePath:  file,
		FuncName:  fn.Name,
		StartLine: fn.StartLine,
		EndLine:   fn.EndLine,
	}
	
	d.hashes[hash] = append(d.hashes[hash], loc)
}

// DetectDuplicates checks if the given code string is a duplicate
func (d *ExactDuplicationDetector) DetectDuplicates(code string) []DuplicateLocation {
	hash := computeCodeHash(code)
	return d.hashes[hash]
}

// computeCodeHash calculates SHA256 of normalized code
func computeCodeHash(code string) string {
	var sb strings.Builder
	for _, r := range code {
		if !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	normalized := sb.String()
	
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum)
}

// GetDuplicates returns all found duplicates
func (d *ExactDuplicationDetector) GetDuplicates() map[string][]DuplicateLocation {
	return d.hashes
}
