package embeddings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/katichai/katich/internal/analysis"
)

// CodeEmbedding represents an embedding for a code block
type CodeEmbedding struct {
	ID         string    `json:"id"`          // Unique identifier (hash of code)
	FilePath   string    `json:"file_path"`   // File containing the code
	FuncName   string    `json:"func_name"`   // Function/class name
	StartLine  int       `json:"start_line"`  // Start line number
	EndLine    int       `json:"end_line"`    // End line number
	Code       string    `json:"code"`        // The actual code
	Embedding  []float32 `json:"embedding"`   // The embedding vector
	Language   string    `json:"language"`    // Programming language
}

// EmbeddingIndex stores all embeddings
type EmbeddingIndex struct {
	Embeddings []CodeEmbedding   `json:"embeddings"`
	FileHashes map[string]string `json:"file_hashes"`
	Dimension  int               `json:"dimension"`
	Provider   string            `json:"provider"`
	Version    string            `json:"version"`
}

// Generator generates embeddings for code
type Generator struct {
	provider EmbeddingProvider
	rootPath string
}

// NewGenerator creates a new embedding generator
func NewGenerator(provider EmbeddingProvider, rootPath string) *Generator {
	return &Generator{
		provider: provider,
		rootPath: rootPath,
	}
}

// GenerateForAnalysis generates embeddings for analyzed code
// GenerateForAnalysis generates embeddings for analyzed code
func (g *Generator) GenerateForAnalysis(analysisResult *analysis.AnalysisResult, incremental bool) (*EmbeddingIndex, error) {
	// Try to load existing index for incremental update
	var existingIndex *EmbeddingIndex
	if incremental {
		indexPath := filepath.Join(g.rootPath, ".katich", "embeddings.json")
		var err error
		existingIndex, err = LoadIndex(indexPath)
		if err != nil {
			// If loading fails, just treat as full rebuild
			fmt.Printf("  ⚠️  Could not load existing index for incremental update: %v\n", err)
			existingIndex = nil
		}
	}

	index := &EmbeddingIndex{
		Embeddings: make([]CodeEmbedding, 0),
		FileHashes: make(map[string]string),
		Dimension:  g.provider.GetDimension(),
		Provider:   g.provider.GetName(),
		Version:    "1.0",
	}

	totalFunctions := 0
	for _, fileAnalysis := range analysisResult.Files {
		totalFunctions += len(fileAnalysis.Functions)
	}

	processed := 0
	reused := 0
	generated := 0

	for filePath, fileAnalysis := range analysisResult.Files {
		// Update hash map
		index.FileHashes[filePath] = fileAnalysis.Hash

		// Check if file is unchanged
		isUnchanged := false
		if existingIndex != nil {
			if oldHash, ok := existingIndex.FileHashes[filePath]; ok {
				if oldHash == fileAnalysis.Hash {
					isUnchanged = true
				}
			}
		}

		if isUnchanged {
			// Reuse embeddings for this file
			foundEmbeddings := false
			for _, emb := range existingIndex.Embeddings {
				if emb.FilePath == filePath {
					index.Embeddings = append(index.Embeddings, emb)
					foundEmbeddings = true
				}
			}
			
			// If we successfully reused embeddings, skip generation
			if foundEmbeddings {
				reused += len(fileAnalysis.Functions)
				processed += len(fileAnalysis.Functions)
				continue
			}
		}

		// Generate embeddings for each function in this file
		for _, fn := range fileAnalysis.Functions {
			// Create code snippet for embedding
			codeSnippet := g.createCodeSnippet(fn, fileAnalysis.Language)
			
			// Generate embedding
			embedding, err := g.provider.GenerateEmbedding(codeSnippet)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: Failed to generate embedding for %s:%s: %v\n", filePath, fn.Name, err)
				continue
			}

			// Create code embedding
			codeEmb := CodeEmbedding{
				ID:        g.generateID(filePath, fn.Name, fn.StartLine),
				FilePath:  filePath,
				FuncName:  fn.Name,
				StartLine: fn.StartLine,
				EndLine:   fn.EndLine,
				Code:      codeSnippet,
				Embedding: embedding,
				Language:  fileAnalysis.Language,
			}

			index.Embeddings = append(index.Embeddings, codeEmb)
			processed++
			generated++

			// Progress indicator
			if processed%10 == 0 {
				fmt.Printf("  Processed %d/%d (Generated: %d, Reused: %d)...\n", processed, totalFunctions, generated, reused)
			}
		}
	}

	fmt.Printf("  Finished: Generated %d, Reused %d embeddings\n", generated, reused)
	return index, nil
}

// createCodeSnippet creates a code snippet for embedding
func (g *Generator) createCodeSnippet(fn analysis.FunctionInfo, language string) string {
	// For now, create a simple representation
	// In the future, we could read the actual code from the file
	snippet := fmt.Sprintf("// Language: %s\n", language)
	snippet += fmt.Sprintf("// Function: %s\n", fn.Name)
	
	if len(fn.Parameters) > 0 {
		snippet += fmt.Sprintf("// Parameters: %v\n", fn.Parameters)
	}
	
	if fn.ReturnType != "" {
		snippet += fmt.Sprintf("// Returns: %s\n", fn.ReturnType)
	}
	
	if fn.Comments != "" {
		snippet += fmt.Sprintf("// Comments: %s\n", fn.Comments)
	}
	
	snippet += fmt.Sprintf("// Complexity: %d\n", fn.Complexity)
	snippet += fmt.Sprintf("// Lines: %d\n", fn.LOC)

	return snippet
}

// generateID generates a unique ID for a code block
func (g *Generator) generateID(filePath, funcName string, startLine int) string {
	data := fmt.Sprintf("%s:%s:%d", filePath, funcName, startLine)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8])
}

// SaveIndex saves the embedding index to disk
func (g *Generator) SaveIndex(index *EmbeddingIndex, outputPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// LoadIndex loads an embedding index from disk
func LoadIndex(indexPath string) (*EmbeddingIndex, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var index EmbeddingIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}
