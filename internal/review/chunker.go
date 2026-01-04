package review

import (
	"fmt"
	"path/filepath"
	"strings"

	analysispkg "github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/llm"
)

// ReviewChunker handles splitting large reviews into manageable chunks
type ReviewChunker struct {
	maxTokens    int
	bufferTokens int
}

// ReviewChunk represents a single chunk of a review
type ReviewChunk struct {
	ID                int
	Files             []*SampledFile
	StaticIssues      []analysispkg.Issue
	DuplicateWarnings []string
	Context           ChunkContext
	TokenCount        int
}

// ChunkContext contains metadata about the chunk
type ChunkContext struct {
	ChunkNumber    int
	TotalChunks    int
	Frameworks     []string
	Languages      []string
	Classification string
}

// NewReviewChunker creates a new review chunker
func NewReviewChunker(maxTokens, bufferTokens int) *ReviewChunker {
	return &ReviewChunker{
		maxTokens:    maxTokens,
		bufferTokens: bufferTokens,
	}
}

// CreateChunks splits a review into multiple chunks based on token limits
func (rc *ReviewChunker) CreateChunks(
	sampledDiff *SampledDiff,
	staticIssues []analysispkg.Issue,
	duplicateWarnings []string,
	frameworks []string,
	languages []string,
	classification string,
) ([]*ReviewChunk, error) {
	
	// Reserve tokens for system prompt, context metadata, output, and buffer
	const SYSTEM_PROMPT_TOKENS = 1500
	const CONTEXT_METADATA_TOKENS = 500
	const OUTPUT_TOKENS_RESERVED = 2000 // Reserve tokens for LLM output
	
	// Calculate available tokens per chunk (input only, we need to reserve space for output)
	availableTokens := rc.maxTokens - SYSTEM_PROMPT_TOKENS - CONTEXT_METADATA_TOKENS - OUTPUT_TOKENS_RESERVED - rc.bufferTokens
	if availableTokens < 1000 {
		return nil, fmt.Errorf("insufficient tokens available for chunking: %d (model limit: %d)", availableTokens, rc.maxTokens)
	}
	
	// For small model limits (like GPT-4 8K), be more aggressive
	if rc.maxTokens <= 10000 {
		// Use only 30% of model capacity for input to leave room for output
		availableTokens = (rc.maxTokens * 30) / 100
		if availableTokens < 500 {
			availableTokens = 500 // Absolute minimum
		}
	}
	
	// Group files by priority and relatedness
	fileGroups := rc.groupFiles(sampledDiff.Files)
	
	// Create chunks from file groups
	chunks := make([]*ReviewChunk, 0)
	currentChunk := &ReviewChunk{
		ID:                len(chunks) + 1,
		Files:             make([]*SampledFile, 0),
		StaticIssues:      make([]analysispkg.Issue, 0),
		DuplicateWarnings: make([]string, 0),
		Context: ChunkContext{
			Frameworks:     frameworks,
			Languages:      languages,
			Classification: classification,
		},
		TokenCount: 0,
	}
	
	for _, group := range fileGroups {
		for _, file := range group {
			fileTokens := llm.EstimateTokens(file.Content) + 100 // +100 for metadata
			
			// If a single file is too large, truncate it
			if fileTokens > availableTokens {
				fmt.Printf("⚠️  Warning: File %s (%d tokens) exceeds chunk limit (%d tokens), truncating...\n", 
					file.Path, fileTokens, availableTokens)
				// Truncate the file content to fit
				file.Content = llm.TruncateToTokenLimit(file.Content, availableTokens-100)
				fileTokens = llm.EstimateTokens(file.Content) + 100
				file.Truncated = true
			}
			
			// If adding this file would exceed the limit, start a new chunk
			if currentChunk.TokenCount+fileTokens > availableTokens && len(currentChunk.Files) > 0 {
				chunks = append(chunks, currentChunk)
				currentChunk = &ReviewChunk{
					ID:                len(chunks) + 1,
					Files:             make([]*SampledFile, 0),
					StaticIssues:      make([]analysispkg.Issue, 0),
					DuplicateWarnings: make([]string, 0),
					Context: ChunkContext{
						Frameworks:     frameworks,
						Languages:      languages,
						Classification: classification,
					},
					TokenCount: 0,
				}
			}
			
			// Add file to current chunk
			currentChunk.Files = append(currentChunk.Files, file)
			currentChunk.TokenCount += fileTokens
		}
	}
	
	// Add final chunk if it has files
	if len(currentChunk.Files) > 0 {
		chunks = append(chunks, currentChunk)
	}
	
	// If no chunks were created, return error
	if len(chunks) == 0 {
		return nil, fmt.Errorf("failed to create any chunks")
	}
	
	// Distribute static issues and duplicate warnings across chunks
	rc.distributeIssues(chunks, staticIssues, duplicateWarnings)
	
	// Update chunk numbers
	totalChunks := len(chunks)
	for i, chunk := range chunks {
		chunk.Context.ChunkNumber = i + 1
		chunk.Context.TotalChunks = totalChunks
	}
	
	return chunks, nil
}

// groupFiles groups files by priority and relatedness
func (rc *ReviewChunker) groupFiles(files []*SampledFile) [][]*SampledFile {
	groups := make([][]*SampledFile, 0)
	
	// Group 1: Security-related files (high priority)
	securityFiles := make([]*SampledFile, 0)
	// Group 2: Files by directory/module
	moduleMap := make(map[string][]*SampledFile)
	// Group 3: Large files (get their own group)
	largeFiles := make([]*SampledFile, 0)
	// Group 4: Other files
	otherFiles := make([]*SampledFile, 0)
	
	for _, file := range files {
		fileTokens := llm.EstimateTokens(file.Content)
		
		// Check if it's a large file (more than 2000 tokens)
		if fileTokens > 2000 {
			largeFiles = append(largeFiles, file)
			continue
		}
		
		// Check if it's a security-related file
		if rc.isSecurityRelated(file) {
			securityFiles = append(securityFiles, file)
			continue
		}
		
		// Group by directory
		dir := filepath.Dir(file.Path)
		moduleMap[dir] = append(moduleMap[dir], file)
	}
	
	// Add security files as first group
	if len(securityFiles) > 0 {
		groups = append(groups, securityFiles)
	}
	
	// Add module groups
	for _, moduleFiles := range moduleMap {
		groups = append(groups, moduleFiles)
	}
	
	// Add large files (each in its own group)
	for _, file := range largeFiles {
		groups = append(groups, []*SampledFile{file})
	}
	
	// Add other files
	if len(otherFiles) > 0 {
		groups = append(groups, otherFiles)
	}
	
	return groups
}

// isSecurityRelated checks if a file is security-related
func (rc *ReviewChunker) isSecurityRelated(file *SampledFile) bool {
	lowerPath := strings.ToLower(file.Path)
	securityKeywords := []string{
		"auth", "security", "crypto", "password", "token", "session",
		"permission", "role", "access", "login", "jwt", "oauth",
	}
	
	for _, keyword := range securityKeywords {
		if strings.Contains(lowerPath, keyword) {
			return true
		}
	}
	
	// Check if file has high risk score
	for _, reason := range file.RiskReasons {
		if strings.Contains(strings.ToLower(reason), "security") {
			return true
		}
	}
	
	return false
}

// distributeIssues distributes static issues and duplicate warnings across chunks
func (rc *ReviewChunker) distributeIssues(
	chunks []*ReviewChunk,
	staticIssues []analysispkg.Issue,
	duplicateWarnings []string,
) {
	// Map issues to files
	fileIssueMap := make(map[string][]analysispkg.Issue)
	for _, issue := range staticIssues {
		fileIssueMap[issue.File] = append(fileIssueMap[issue.File], issue)
	}
	
	// Assign issues to chunks based on which files they contain
	for _, chunk := range chunks {
		for _, file := range chunk.Files {
			if issues, ok := fileIssueMap[file.Path]; ok {
				chunk.StaticIssues = append(chunk.StaticIssues, issues...)
			}
		}
	}
	
	// Distribute duplicate warnings proportionally
	// For simplicity, add all warnings to all chunks
	// (they're typically references to other files)
	for _, chunk := range chunks {
		chunk.DuplicateWarnings = duplicateWarnings
	}
}

