package embeddings

import (
	"fmt"
	"math"
	"sort"
)

// SimilarityResult represents a similarity search result
type SimilarityResult struct {
	CodeEmbedding
	Similarity float32 `json:"similarity"` // Cosine similarity score (0-1)
}

// SimilaritySearch performs similarity search on embeddings
type SimilaritySearch struct {
	index *EmbeddingIndex
}

// NewSimilaritySearch creates a new similarity search
func NewSimilaritySearch(index *EmbeddingIndex) *SimilaritySearch {
	return &SimilaritySearch{
		index: index,
	}
}

// Search finds the top-k most similar code blocks
func (s *SimilaritySearch) Search(queryEmbedding []float32, topK int) []SimilarityResult {
	if len(s.index.Embeddings) == 0 {
		return []SimilarityResult{}
	}

	// Calculate similarity for all embeddings
	results := make([]SimilarityResult, 0, len(s.index.Embeddings))
	
	for _, codeEmb := range s.index.Embeddings {
		similarity := cosineSimilarity(queryEmbedding, codeEmb.Embedding)
		
		results = append(results, SimilarityResult{
			CodeEmbedding: codeEmb,
			Similarity:    similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top-k
	if topK > len(results) {
		topK = len(results)
	}

	return results[:topK]
}

// FindDuplicates finds code blocks that are very similar (>threshold)
func (s *SimilaritySearch) FindDuplicates(queryEmbedding []float32, threshold float32, excludeID string) []SimilarityResult {
	results := s.Search(queryEmbedding, len(s.index.Embeddings))
	
	duplicates := make([]SimilarityResult, 0)
	for _, result := range results {
		// Skip the query itself
		if result.ID == excludeID {
			continue
		}
		
		// Only include results above threshold
		if result.Similarity >= threshold {
			duplicates = append(duplicates, result)
		}
	}

	return duplicates
}

// SearchByCode searches for similar code using a code snippet
func (s *SimilaritySearch) SearchByCode(provider EmbeddingProvider, code string, topK int) ([]SimilarityResult, error) {
	// Generate embedding for the query code
	queryEmbedding, err := provider.GenerateEmbedding(code)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	return s.Search(queryEmbedding, topK), nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// DuplicateDetector detects duplicate code
type DuplicateDetector struct {
	search    *SimilaritySearch
	provider  EmbeddingProvider
	threshold float32
}

// NewDuplicateDetector creates a new duplicate detector
func NewDuplicateDetector(index *EmbeddingIndex, provider EmbeddingProvider, threshold float32) *DuplicateDetector {
	if threshold == 0 {
		threshold = 0.85 // Default threshold
	}

	return &DuplicateDetector{
		search:    NewSimilaritySearch(index),
		provider:  provider,
		threshold: threshold,
	}
}

// DetectDuplicates detects if new code is duplicate
func (d *DuplicateDetector) DetectDuplicates(code, filePath, funcName string) ([]SimilarityResult, error) {
	// Generate embedding for new code
	embedding, err := d.provider.GenerateEmbedding(code)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Find duplicates
	duplicates := d.search.FindDuplicates(embedding, d.threshold, "")

	// Filter out the same file/function
	filtered := make([]SimilarityResult, 0)
	for _, dup := range duplicates {
		if dup.FilePath != filePath || dup.FuncName != funcName {
			filtered = append(filtered, dup)
		}
	}

	return filtered, nil
}

// GetSimilarityLevel returns a human-readable similarity level
func GetSimilarityLevel(similarity float32) string {
	if similarity >= 0.95 {
		return "Nearly Identical"
	} else if similarity >= 0.85 {
		return "Very Similar"
	} else if similarity >= 0.75 {
		return "Similar"
	} else if similarity >= 0.60 {
		return "Somewhat Similar"
	}
	return "Different"
}
