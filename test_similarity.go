package main

import (
	"fmt"
	"log"

	"github.com/katichai/katich/internal/embeddings"
)

func main() {
	// Load the embedding index
	index, err := embeddings.LoadIndex(".katich/embeddings.json")
	if err != nil {
		log.Fatalf("Failed to load index: %v", err)
	}

	fmt.Printf("Loaded %d embeddings\n", len(index.Embeddings))
	fmt.Printf("Provider: %s\n", index.Provider)
	fmt.Printf("Dimension: %d\n\n", index.Dimension)

	// Create similarity search
	search := embeddings.NewSimilaritySearch(index)

	// Find the CalculateSum function
	var calculateSumEmbedding []float32
	var calculateSumID string
	for _, emb := range index.Embeddings {
		if emb.FuncName == "CalculateSum" {
			calculateSumEmbedding = emb.Embedding
			calculateSumID = emb.ID
			fmt.Printf("Found CalculateSum in %s (lines %d-%d)\n", emb.FilePath, emb.StartLine, emb.EndLine)
			break
		}
	}

	if calculateSumEmbedding == nil {
		log.Fatal("CalculateSum not found")
	}

	// Search for similar functions
	fmt.Println("\nSearching for similar functions...")
	results := search.Search(calculateSumEmbedding, 5)

	fmt.Println("\nTop 5 similar functions:")
	for i, result := range results {
		if result.ID == calculateSumID {
			fmt.Printf("%d. %s (SELF)\n", i+1, result.FuncName)
		} else {
			similarity := result.Similarity * 100
			level := embeddings.GetSimilarityLevel(result.Similarity)
			fmt.Printf("%d. %s - %.1f%% similar (%s)\n", i+1, result.FuncName, similarity, level)
			fmt.Printf("   Location: %s (lines %d-%d)\n", result.FilePath, result.StartLine, result.EndLine)
		}
	}

	// Find duplicates (>85% similar)
	fmt.Println("\nSearching for duplicates (>85% similar)...")
	duplicates := search.FindDuplicates(calculateSumEmbedding, 0.85, calculateSumID)

	if len(duplicates) > 0 {
		fmt.Printf("\n⚠️  Found %d potential duplicate(s):\n", len(duplicates))
		for _, dup := range duplicates {
			similarity := dup.Similarity * 100
			fmt.Printf("  • %s - %.1f%% similar\n", dup.FuncName, similarity)
			fmt.Printf("    Location: %s (lines %d-%d)\n", dup.FilePath, dup.StartLine, dup.EndLine)
		}
	} else {
		fmt.Println("No duplicates found")
	}
}
