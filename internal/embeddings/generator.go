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
	LOC        int       `json:"loc"`         // Lines of code
	Complexity int       `json:"complexity"`  // Cyclomatic complexity
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

	reused := 0
	
	// Collect all code snippets that need new embeddings
	type snippetMetadata struct {
		FilePath  string
		Function  analysis.FunctionInfo
		Language  string
		Snippet   string
	}
	
	var snippetsToGenerate []string
	var snippetMeta []snippetMetadata

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
				continue
			}
		}

		// Collect snippets for batch generation
		for _, fn := range fileAnalysis.Functions {
			// Create code snippet for embedding
			codeSnippet := g.createCodeSnippet(fn, fileAnalysis.Language)
			
			snippetsToGenerate = append(snippetsToGenerate, codeSnippet)
			snippetMeta = append(snippetMeta, snippetMetadata{
				FilePath: filePath,
				Function: fn,
				Language: fileAnalysis.Language,
				Snippet:  codeSnippet,
			})
		}
	}

	generated := len(snippetsToGenerate)
	
	if generated > 0 {
		fmt.Printf("  Generating %d embeddings in batches (Reusing: %d)...\n", generated, reused)
		
		// Generate all embeddings in batches
		embeddings, err := g.provider.GenerateBatchEmbeddings(snippetsToGenerate)
		if err != nil {
			return nil, fmt.Errorf("batch embedding generation failed: %w", err)
		}
		
		// Verify we got all embeddings
		if len(embeddings) != len(snippetsToGenerate) {
			return nil, fmt.Errorf("embedding count mismatch: expected %d, got %d", len(snippetsToGenerate), len(embeddings))
		}
		
		// Map embeddings back to functions
		for i, emb := range embeddings {
			meta := snippetMeta[i]
			
			codeEmb := CodeEmbedding{
				ID:         g.generateID(meta.FilePath, meta.Function.Name, meta.Function.StartLine),
				FilePath:   meta.FilePath,
				FuncName:   meta.Function.Name,
				StartLine:  meta.Function.StartLine,
				EndLine:    meta.Function.EndLine,
				Code:       meta.Snippet,
				Embedding:  emb,
				Language:   meta.Language,
				LOC:        meta.Function.LOC,
				Complexity: meta.Function.Complexity,
			}
			
			index.Embeddings = append(index.Embeddings, codeEmb)
		}
		
		fmt.Printf("  ✅ Generated %d embeddings successfully\n", generated)
	}
	
	fmt.Printf("  Finished: Generated %d, Reused %d embeddings (Total: %d)\n", generated, reused, generated+reused)
	return index, nil
}

// createCodeSnippet creates a code snippet for embedding
func (g *Generator) createCodeSnippet(fn analysis.FunctionInfo, language string) string {
	// Use actual function body for better semantic matching
	snippet := fmt.Sprintf("// Language: %s\n", language)
	snippet += fmt.Sprintf("// Function: %s\n", fn.Name)
	snippet += fmt.Sprintf("// Complexity: %d, LOC: %d\n", fn.Complexity, fn.LOC)
	
	// Include actual body for semantic similarity
	if fn.Body != "" {
		snippet += fn.Body
	} else {
		// Fallback to metadata if body not available
		if len(fn.Parameters) > 0 {
			snippet += fmt.Sprintf("// Parameters: %v\n", fn.Parameters)
		}
		if fn.ReturnType != "" {
			snippet += fmt.Sprintf("// Returns: %s\n", fn.ReturnType)
		}
	}
	
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
