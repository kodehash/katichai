package review

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileReader reads file contents for HTML rendering
type FileReader struct {
	repoRoot string
	cache    map[string]string
}

// NewFileReader creates a new file reader
func NewFileReader(repoRoot string) *FileReader {
	return &FileReader{
		repoRoot: repoRoot,
		cache:    make(map[string]string),
	}
}

// ReadFile reads the content of a file, resolving paths relative to repo root
func (fr *FileReader) ReadFile(filePath string) (string, error) {
	// Check cache first
	if content, exists := fr.cache[filePath]; exists {
		return content, nil
	}

	// Resolve path relative to repo root
	var fullPath string
	if filepath.IsAbs(filePath) {
		fullPath = filePath
	} else {
		fullPath = filepath.Join(fr.repoRoot, filePath)
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// Cache and return
	fr.cache[filePath] = string(content)
	return string(content), nil
}

// ReadFiles reads multiple files and returns a map of file path to content
func (fr *FileReader) ReadFiles(filePaths []string) (map[string]string, error) {
	result := make(map[string]string)
	
	for _, filePath := range filePaths {
		content, err := fr.ReadFile(filePath)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		result[filePath] = content
	}
	
	return result, nil
}

// ClearCache clears the file content cache
func (fr *FileReader) ClearCache() {
	fr.cache = make(map[string]string)
}

