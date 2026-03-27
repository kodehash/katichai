package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/llm"
)

// ReviewEngine orchestrates the complete review pipeline
type ReviewEngine struct {
	config      *config.Config
	llmClient   llm.LLMProvider
	reviewer    *Reviewer
	synthesizer *Synthesizer
	pmtBuilder  *llm.PromptBuilder
	diffRange   string // Stores the diff range for HTML report
}

// Limits for full repository reviews to prevent excessive runtime
// These are hard limits that prevent the review from starting
const (
	MAX_FILES_FULL_REVIEW_LIMIT = 3000   // Maximum number of files allowed for full review
	MAX_LINES_FULL_REVIEW_LIMIT = 300000 // Maximum total lines of code allowed for full review
)

// LineRange represents a range of line numbers
type LineRange struct {
	Start int
	End   int
}

// NewEngine creates a new ReviewEngine
func NewEngine(cfg *config.Config, reviewer *Reviewer) (*ReviewEngine, error) {
	// Get project name from config or extract from Git
	projectName, err := cfg.GetProjectName()
	if err != nil {
		// Try to extract from Git repository
		repo, repoErr := git.FindRepository()
		if repoErr != nil {
			return nil, fmt.Errorf("failed to get project name: %w (also failed to find Git repo: %v)", err, repoErr)
		}
		projectName, repoErr = repo.GetProjectName()
		if repoErr != nil {
			return nil, fmt.Errorf("failed to get project name: %w (also failed to extract from Git: %v)", err, repoErr)
		}
	}
	
	// Create KeyFetcher if API server is enabled
	var keyFetcher *llm.KeyFetcher
	if cfg.APIServer.Enabled {
		if cfg.APIServer.URL == "" {
			return nil, fmt.Errorf("API server URL is required when API server is enabled")
		}
		if cfg.APIServer.Token == "" {
			return nil, fmt.Errorf("API server token is required when API server is enabled")
		}
		keyFetcher = llm.NewKeyFetcher(cfg.APIServer.URL, cfg.APIServer.Token)
	}
	
	// Initialize LLM Client
	client, err := llm.NewClient(cfg.LLM, projectName, keyFetcher)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	return &ReviewEngine{
		config:      cfg,
		llmClient:   client,
		reviewer:    reviewer,
		synthesizer: NewSynthesizer(),
		pmtBuilder:  llm.NewPromptBuilder(),
	}, nil
}

// Review performs a comprehensive review of the code changes
// diffRange is optional and used for HTML report display (e.g., "main..feature" or "HEAD~3..HEAD")
func (e *ReviewEngine) Review(diff *git.Diff, diffRange ...string) (*ReviewReport, error) {
	// Store diff range if provided
	if len(diffRange) > 0 && diffRange[0] != "" {
		e.diffRange = diffRange[0]
	}
	// 1. Run Local Analysis (Static + Similarity)
	// This uses the existing Reviewer logic
	localResult, err := e.reviewer.ReviewDiff(diff)
	if err != nil {
		return nil, fmt.Errorf("local analysis failed: %w", err)
	}
	
	// Show similarity check status
	if localResult.SimilarityCheckLimited {
		if strings.Contains(localResult.SimilarityCheckReason, "no context") {
			fmt.Printf("   ⚠️  Similarity check disabled: %s\n", localResult.SimilarityCheckReason)
		} else {
			fmt.Printf("   ⚠️  Similarity check %s\n", localResult.SimilarityCheckReason)
		}
	}

	// 1.5 Run Classifier
	// We run this early to potentially adjust strategy, though for now we just record it
	classifier := llm.NewClassifier(e.llmClient)
	diffStr := formatDiff(diff) // We need diff string earlier now
	
	// Truncate diff for classifier if too large to save tokens
	classifyDiff := diffStr
	if len(classifyDiff) > 2000 {
		classifyDiff = classifyDiff[:2000] + "\n... (truncated)"
	}
	
	classification, err := classifier.ClassifyChanges(context.Background(), classifyDiff)
	if err != nil {
		// Non-critical failure, log and continue
		fmt.Printf("Warning: Classification failed: %v\n", err)
		classification = llm.ClassificationResult{Type: llm.ClassUnknown}
	}


	// 2. Aggregate Issues for Prompt
	var staticIssues []analysis.Issue
	var duplicateWarnings []string
	var dbQueryReviews []DBQueryReview

	// Initialize detectors
	driftDetector := analysis.NewDriftDetector()
	exactDupDetector := analysis.NewExactDuplicationDetector()
	
	// Analyze DB/ORM queries
	for path, fileAnalysis := range localResult.FileAnalysis {
		// Check if file has database/ORM calls
		if e.hasDatabaseOrORMCalls(fileAnalysis) {
			queries := analysis.AnalyzeDBQueries(fileAnalysis, path)
			for _, query := range queries {
				dbQueryReviews = append(dbQueryReviews, DBQueryReview{
					File:            query.File,
					Line:            query.Line,
					Function:        query.Function,
					QueryType:       query.QueryType,
					EfficiencyScore: query.EfficiencyScore,
					Issues:          e.convertDBQueryIssues(query.Issues),
					Recommendation:  query.Recommendation,
					QuerySnippet:    query.Query,
				})
			}
		}
	}

	// Populate exact match index (should be done from context in real app, here we demo on local files)
	// For now we skip repo-wide indexing for speed and just check self-duplication in diff for demo

	for path, fAnalysis := range localResult.FileAnalysis {
		// Run Drift Detection
		// We need the raw content, which we'd get from Git or fAnalysis if it stored it
		// For now we assume we can read the file or Analysis has Body. 
		// Parser puts Body in FunctionInfo, but we need file content for Drift.
		// We'll skip reading file again for now and just rely on what we have.
		
		// Run Drift on functions if Body is available
		for _, fn := range fAnalysis.Functions {
			if fn.Body != "" {
				driftReport := driftDetector.CheckDrift(fn.Body, fAnalysis.Language)
				staticIssues = append(staticIssues, driftReport.Issues...)
				
				// Exact Duplication Check (Self-check within PR)
				exactDupDetector.AddFunction(path, fn)
			}
		}

		for _, issue := range fAnalysis.Issues {
			if issue.File == "" {
				issue.File = path
			}
			staticIssues = append(staticIssues, issue)
		}
	}

	// Check for exact duplicates within the changes themselves
	for _, dups := range exactDupDetector.GetDuplicates() {
		if len(dups) > 1 {
			// Reported multiple times
			// We only want to report each group once
			// For simplicity, we report for the first instance found in the diff iteration
			// But since we iterate map, let's just format them
			var locs []string
			for _, loc := range dups {
				locs = append(locs, fmt.Sprintf("%s:%d", loc.FilePath, loc.StartLine))
			}
			duplicateWarnings = append(duplicateWarnings, 
				fmt.Sprintf("Exact duplicate code found in: %s", strings.Join(locs, ", ")))
		}
	}


	for _, dups := range localResult.Duplicates {
		for _, d := range dups {
			duplicateWarnings = append(duplicateWarnings, 
				fmt.Sprintf("Potential duplicate of %s (Match: %.1f%%)", d.FilePath, d.Similarity*100))
		}
	}
	
	// Report Reuse Candidates (Refactoring Opportunities)
	for _, candidates := range localResult.ReuseCandidates {
		for _, c := range candidates {
			duplicateWarnings = append(duplicateWarnings, 
				fmt.Sprintf("Refactoring Opportunity: Similar logic in %s (Match: %.1f%%)", c.FilePath, c.Similarity*100))
		}
	}

	// 3. Calculate Token Budget and Sample Diff
	// Estimate context tokens first
	contextTokens := llm.EstimateTokens(llm.SystemPrompt)
	
	// Build preliminary context to estimate its size
	prelimCtx := llm.ReviewContext{
		Frameworks:     []string{}, // TODO: Load from context.json if available
		Languages:      detectLanguages(diff),
		StaticIssues:   staticIssues[:min(len(staticIssues), 10)], // Limit static issues for estimate
		SimilarCode:    duplicateWarnings[:min(len(duplicateWarnings), 5)],
		FileContext:    summarizeFiles(diff),
		Classification: fmt.Sprintf("%s (Confidence: %.2f)", classification.Type, classification.Confidence),
	}
	prelimPrompt := e.pmtBuilder.BuildReviewPrompt(prelimCtx)
	contextTokens += llm.EstimateTokens(prelimPrompt) - llm.EstimateTokens(formatDiff(diff)) // Subtract placeholder diff
	
	// Calculate available budget for diff
	diffBudget := TOTAL_INPUT_BUDGET - contextTokens - BUFFER_TOKENS
	if diffBudget < 1000 {
		diffBudget = 1000 // Minimum
	}
	if diffBudget > 16000 {
		diffBudget = 16000 // Maximum
	}
	
	// Sample the diff using intelligent sampling
	sampler := NewDiffSampler(diffBudget)
	sampledDiff, samplingReport := sampler.SampleDiff(diff, localResult.FileAnalysis)
	
	// Display sampling report to user
	fmt.Println("\n📊 Sampling Report:")
	fmt.Printf("  • Total files changed: %d\n", samplingReport.TotalFiles)
	fmt.Printf("  • Filtered: %d (", samplingReport.FilteredFiles)
	reasons := []string{}
	for reason, count := range samplingReport.FilteredReasons {
		reasons = append(reasons, fmt.Sprintf("%s: %d", reason, count))
	}
	fmt.Printf("%s)\n", strings.Join(reasons, ", "))
	fmt.Printf("  • Reviewing: %d files\n", samplingReport.SampledFiles)
	
	if len(samplingReport.TopRiskFiles) > 0 {
		fmt.Println("\n🔴 High Priority Files:")
		for i, risk := range samplingReport.TopRiskFiles {
			if i >= 5 {
				break
			}
			fmt.Printf("  • %s (Risk: %d) - %s\n", risk.File.Path, risk.Score, strings.Join(risk.Reasons, ", "))
		}
	}
	fmt.Println()
	
	// 4. Always use chunked review -- the chunker handles both small and large
	// prompts correctly, computing output tokens dynamically per chunk.
	return e.performChunkedReview(diff, localResult, staticIssues, duplicateWarnings, &classification, sampledDiff, samplingReport, exactDupDetector, dbQueryReviews)
}

// ChunkResult represents the result of reviewing a single chunk
type ChunkResult struct {
	ChunkID      int
	Content      string
	InputTokens  int
	OutputTokens int
	Error        error
}

// buildChunkPrompt builds a review prompt for a single chunk
func buildChunkPrompt(chunk *ReviewChunk) string {
	var sb strings.Builder
	
	// Add chunk context
	sb.WriteString(fmt.Sprintf("**CHUNK %d of %d** - This is part of a larger code review.\n\n", 
		chunk.Context.ChunkNumber, chunk.Context.TotalChunks))
	
	sb.WriteString("Please review the following code changes in this chunk.\n\n")
	
	// Include context
	sb.WriteString("### Repository Context\n")
	if len(chunk.Context.Languages) > 0 {
		sb.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(chunk.Context.Languages, ", ")))
	}
	if len(chunk.Context.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("- Frameworks: %s\n", strings.Join(chunk.Context.Frameworks, ", ")))
	}
	if chunk.Context.Classification != "" {
		sb.WriteString(fmt.Sprintf("- Classification: %s\n", chunk.Context.Classification))
	}
	sb.WriteString("\n")
	
	// Static issues for this chunk
	if len(chunk.StaticIssues) > 0 {
		sb.WriteString("### Static Analysis Findings (Verify these)\n")
		for _, issue := range chunk.StaticIssues {
			sb.WriteString(fmt.Sprintf("- [%s] %s (Line %d): %s\n", 
				issue.Severity, issue.Type, issue.Line, issue.Message))
		}
		sb.WriteString("\n")
	}
	
	// Duplication warnings for this chunk
	if len(chunk.DuplicateWarnings) > 0 {
		sb.WriteString("### Potential Duplication Detected\n")
		for _, match := range chunk.DuplicateWarnings {
			sb.WriteString(fmt.Sprintf("- %s\n", match))
		}
		sb.WriteString("\n")
	}
	
	// Files in this chunk
	sb.WriteString("### Code Changes\n")
	for _, file := range chunk.Files {
		sb.WriteString(fmt.Sprintf("\n**File: %s** (Status: %s, +%d -%d lines)\n", 
			file.Path, file.Status, file.Additions, file.Deletions))
		sb.WriteString("```diff\n")
		sb.WriteString(file.Content)
		sb.WriteString("\n```\n")
	}
	
	sb.WriteString("\nProvide your review for this chunk following the standard format.\n")
	sb.WriteString("\nNote: You are reviewing only a subset of changes. Focus on issues within this chunk.\n")
	
	return sb.String()
}

// performChunkedReview handles reviews that exceed token limits by splitting into chunks
func (e *ReviewEngine) performChunkedReview(
	diff *git.Diff,
	localResult *ReviewResult,
	staticIssues []analysis.Issue,
	duplicateWarnings []string,
	classification *llm.ClassificationResult,
	sampledDiff *SampledDiff,
	samplingReport *SamplingReport,
	exactDupDetector *analysis.ExactDuplicationDetector,
	dbQueryReviews []DBQueryReview,
) (*ReviewReport, error) {
	
	// 1. Create chunks
	modelLimit := llm.GetModelTokenLimitWithConfig(e.llmClient.GetModel(), e.config.LLM.MaxInputTokens)
	chunker := NewReviewChunker(
		modelLimit,
		1000, // buffer (reduced from 2000 to allow more content)
	)
	
	frameworks := []string{} // TODO: Load from context.json if available
	languages := detectLanguages(diff)
	classificationStr := fmt.Sprintf("%s (Confidence: %.2f)", classification.Type, classification.Confidence)
	
	chunks, err := chunker.CreateChunks(sampledDiff, staticIssues, duplicateWarnings, frameworks, languages, classificationStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks: %w", err)
	}
	
	fmt.Printf("📦 Split into %d chunks for parallel review\n", len(chunks))

	// 2. Schedule chunks into TPM-aware minute-windows, then process each window
	// in parallel using the existing semaphore (max 3 concurrent requests).
	tpmLimit := e.config.LLM.TokensPerMinute
	windows := scheduleChunkWindows(chunks, tpmLimit)

	if len(windows) > 1 {
		fmt.Printf("⏱  TPM rate limiting active: scheduling %d chunks across %d minute-windows (limit: %d tokens/min)\n",
			len(chunks), len(windows), tpmLimit)
	}

	// Semaphore: limits max concurrent in-flight API calls (unchanged behaviour)
	maxConcurrent := 3
	semaphore := make(chan struct{}, maxConcurrent)

	results := make([]ChunkResult, 0, len(chunks))

	for winIdx, window := range windows {
		windowStart := time.Now()
		windowChan := make(chan ChunkResult, len(window.Chunks))

		for _, chunk := range window.Chunks {
			// Acquire semaphore
			semaphore <- struct{}{}

			go func(c *ReviewChunk) {
				defer func() { <-semaphore }()

				fmt.Printf("🤖 Reviewing chunk %d/%d...\n", c.ID, len(chunks))

				chunkPrompt := buildChunkPrompt(c)

				// Calculate available tokens for completion
				inputTokens := llm.EstimateTokens(llm.SystemPrompt) + llm.EstimateTokens(chunkPrompt)
				maxOutputTokens := modelLimit - inputTokens - 500 // 500 buffer

				// Cap at reasonable maximum
				if maxOutputTokens > 4096 {
					maxOutputTokens = 4096
				}
				if maxOutputTokens < 512 {
					maxOutputTokens = 512 // Minimum to provide useful response
				}

				resp, err := e.llmClient.GenerateCompletion(context.Background(), llm.CompletionRequest{
					Messages: []llm.Message{
						{Role: llm.RoleSystem, Content: llm.SystemPrompt},
						{Role: llm.RoleUser, Content: chunkPrompt},
					},
					Temperature: 0.2,
					MaxTokens:   maxOutputTokens,
				})

				result := ChunkResult{ChunkID: c.ID}
				if err != nil {
					result.Error = fmt.Errorf("chunk %d review failed: %w", c.ID, err)
				} else {
					result.Content = resp.Content
					result.InputTokens = resp.Usage.PromptTokens
					result.OutputTokens = resp.Usage.CompletionTokens
				}

				windowChan <- result
			}(chunk)
		}

		// Collect results for this window
		for range window.Chunks {
			result := <-windowChan
			if result.Error != nil {
				close(windowChan)
				return nil, result.Error
			}
			results = append(results, result)
		}
		close(windowChan)

		// If there are more windows, sleep for the remainder of the 60-second window
		// so we don't exceed the tokens-per-minute limit when the next window fires.
		if winIdx < len(windows)-1 {
			elapsed := time.Since(windowStart)
			wait := 60*time.Second - elapsed
			if wait > 0 {
				fmt.Printf("⏳ Window %d/%d done. Waiting %s before next batch to respect TPM limit...\n",
					winIdx+1, len(windows), wait.Round(time.Second))
				time.Sleep(wait)
			}
		}
	}
	
	// Sort results by chunk ID to maintain order
	// Using a simple bubble sort since the list is likely small
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].ChunkID > results[j+1].ChunkID {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
	
	// Extract responses in order
	var allResponses []string
	var totalInputTokens, totalOutputTokens int
	for _, result := range results {
		allResponses = append(allResponses, result.Content)
		totalInputTokens += result.InputTokens
		totalOutputTokens += result.OutputTokens
	}
	
	// 3. Merge chunk results
	fmt.Println("🔄 Merging chunk results...")
	mergedContent := e.mergeChunkResults(allResponses)
	
	// 4. Filter LLM output to only issues related to actual code changes
	filteredLLMOutput := e.filterLLMOutputToChanges(mergedContent, diff, localResult.FileAnalysis)
	
	// 5. Synthesize final report
	tokenUsage := TokenUsage{
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		TotalTokens:  totalInputTokens + totalOutputTokens,
	}
	
	report := e.synthesizer.Synthesize(filteredLLMOutput, staticIssues, duplicateWarnings, localResult.FileAnalysis, tokenUsage, localResult.SimilarityCheckLimited, localResult.SimilarityCheckReason, dbQueryReviews)
	
	// 6. Populate sampling information
	e.populateSamplingInfo(report, diff, sampledDiff, samplingReport)
	
	// 7. Populate duplicate blocks and AI patterns for HTML report
	e.populateDuplicateBlocks(report, exactDupDetector, localResult, diff)
	if e.config.Review.DetectAICode {
		e.populateAIPatterns(report, localResult, diff)
	} else {
		// Strip AIScore from file analysis so formatters don't render it
		for _, fa := range report.FileAnalysis {
			if fa != nil {
				fa.AIScore = nil
			}
		}
	}

	// 7.5. Generate fix prompt if enabled
	if e.config.Review.GenerateFixPrompt {
		report.FixPrompt = e.synthesizer.BuildFixPrompt(report)
	}
	
	// 8. Generate HTML report if enabled
	if e.shouldGenerateHTML() {
		if err := e.generateHTMLReport(report, diff, e.diffRange); err != nil {
			// Log error but don't fail the review
			fmt.Printf("Warning: Failed to generate HTML report: %v\n", err)
		}
	}

	// 9. Generate GFM report if enabled
	if e.shouldGenerateGFM() {
		if err := e.generateGFMReport(report, diff, e.diffRange); err != nil {
			// Log error but don't fail the review
			fmt.Printf("Warning: Failed to generate GFM report: %v\n", err)
		}
	}

	return report, nil
}

// ReviewFullRepository performs a comprehensive review of the entire repository
// This is different from Review() which reviews diffs - this reviews all tracked files
func (e *ReviewEngine) ReviewFullRepository(diff *git.Diff) (*ReviewReport, error) {
	// Set diff range to indicate full repository review
	e.diffRange = "full repository"

	// Validate repository size before starting review
	// Note: Lines of code is not a problem, only number of files matters
	fileCount := len(diff.Files)
	
	if fileCount > MAX_FILES_FULL_REVIEW_LIMIT {
		return nil, fmt.Errorf(
			"repository too large for full review: %d files (max: %d). "+
				"Full repository reviews are limited to prevent excessive runtime (>15 min). "+
				"Consider reviewing specific files or commits instead.",
			fileCount, MAX_FILES_FULL_REVIEW_LIMIT,
		)
	}
	
	// Calculate total lines for display only (not used for validation)
	totalLines := 0
	for _, file := range diff.Files {
		totalLines += file.Additions
	}
	
	fmt.Printf("📏 Repository size: %d files, %d lines of code\n", fileCount, totalLines)

	// 1. Run Local Analysis (Static + Similarity)
	fmt.Println("📊 Running static analysis on repository files...")
	localResult, err := e.reviewer.ReviewDiff(diff)
	if err != nil {
		return nil, fmt.Errorf("local analysis failed: %w", err)
	}
	fmt.Printf("   ✓ Analyzed %d files\n", len(localResult.FileAnalysis))
	fmt.Println("   ⏳ Review in progress...")

	// Show similarity check status
	if localResult.SimilarityCheckLimited {
		if strings.Contains(localResult.SimilarityCheckReason, "no context") {
			fmt.Printf("   ⚠️  Similarity check disabled: %s\n", localResult.SimilarityCheckReason)
		} else {
			fmt.Printf("   ⚠️  Similarity check %s\n", localResult.SimilarityCheckReason)
		}
	}

	// 1.5 Run Classifier (optional for full repo, but useful for context)
	fmt.Println("🔍 Classifying repository changes...")
	fmt.Println("   ⏳ Preparing classification input...")
	classifier := llm.NewClassifier(e.llmClient)
	
	// For full repo reviews, don't format the entire diff (too slow for 263 files)
	// Just use a summary instead
	fmt.Println("   ⏳ Building classification summary...")
	var classifyDiff string
	if len(diff.Files) > 50 {
		// For large repos, just use file names and stats
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Full repository review: %d files, %d lines\n", len(diff.Files), totalLines))
		sb.WriteString("Sample files:\n")
		for i, file := range diff.Files {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("  - %s (%d additions)\n", file.Path, file.Additions))
		}
		classifyDiff = sb.String()
		fmt.Printf("   ✓ Classification summary ready (%d chars)\n", len(classifyDiff))
	} else {
		fmt.Println("   ⏳ Formatting diff for classification...")
		diffStr := formatDiff(diff)
		// Truncate diff for classifier if too large
		classifyDiff = diffStr
		if len(classifyDiff) > 2000 {
			classifyDiff = classifyDiff[:2000] + "\n... (truncated)"
		}
		fmt.Printf("   ✓ Diff formatted (%d chars)\n", len(classifyDiff))
	}
	
	// Use timeout for classifier too
	fmt.Println("   ⏳ Calling LLM classifier (timeout: 2 minutes)...")
	classifyCtx, classifyCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer classifyCancel()
	
	classification, err := classifier.ClassifyChanges(classifyCtx, classifyDiff)
	if err != nil {
		if classifyCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("⚠️  Classification timed out, continuing without classification\n")
		} else {
			fmt.Printf("⚠️  Classification failed: %v\n", err)
		}
		classification = llm.ClassificationResult{Type: llm.ClassUnknown}
	} else {
		fmt.Printf("   ✓ Classification: %s (%.0f%% confidence)\n", classification.Type, classification.Confidence*100)
	}

	// 2. Extract Static Issues and Duplicates
	fmt.Println("🔎 Extracting static analysis issues and duplicates...")
	fmt.Println("   ⏳ Initializing duplicate detector...")
	staticIssues := make([]analysis.Issue, 0)
	duplicateWarnings := make([]string, 0)
	dbQueryReviews := make([]DBQueryReview, 0)

	// Initialize detectors
	exactDupDetector := analysis.NewExactDuplicationDetector()
	fmt.Println("   ✓ Duplicate detector initialized")

	analysisFileCount := len(localResult.FileAnalysis)
	fmt.Printf("   ⏳ Processing %d files for issues and duplicates...\n", analysisFileCount)
	processedFiles := 0
	for path, fileAnalysis := range localResult.FileAnalysis {
		// Add functions for exact duplication detection
		for _, fn := range fileAnalysis.Functions {
			if fn.Body != "" {
				exactDupDetector.AddFunction(path, fn)
			}
		}

		// Collect static issues
		for _, issue := range fileAnalysis.Issues {
			if issue.File == "" {
				issue.File = path
			}
			staticIssues = append(staticIssues, issue)
		}
		
		processedFiles++
		// Show progress for large batches
		if analysisFileCount > 100 && processedFiles%50 == 0 {
			fmt.Printf("   ⏳ Processing files: %d/%d\r", processedFiles, analysisFileCount)
		}
	}
	
	if analysisFileCount > 100 {
		fmt.Printf("   ✓ Processed %d files\n", processedFiles)
	}
	fmt.Printf("   ✓ Collected %d static issues\n", len(staticIssues))

	// Check for exact duplicates
	fmt.Println("   ⏳ Checking for exact duplicates...")
	for _, dups := range exactDupDetector.GetDuplicates() {
		if len(dups) > 1 {
			var locs []string
			for _, loc := range dups {
				locs = append(locs, fmt.Sprintf("%s:%d", loc.FilePath, loc.StartLine))
			}
			duplicateWarnings = append(duplicateWarnings,
				fmt.Sprintf("Exact duplicate code found in: %s", strings.Join(locs, ", ")))
		}
	}

	// Extract similarity-based duplicate warnings
	fmt.Println("   ⏳ Checking similarity-based duplicates...")
	for _, dups := range localResult.Duplicates {
		for _, d := range dups {
			duplicateWarnings = append(duplicateWarnings,
				fmt.Sprintf("Potential duplicate of %s (Match: %.1f%%)", d.FilePath, d.Similarity*100))
		}
	}

	// Report Reuse Candidates
	fmt.Println("   ⏳ Checking reuse candidates...")
	for _, candidates := range localResult.ReuseCandidates {
		for _, c := range candidates {
			duplicateWarnings = append(duplicateWarnings,
				fmt.Sprintf("Refactoring Opportunity: Similar logic in %s (Match: %.1f%%)", c.FilePath, c.Similarity*100))
		}
	}
	fmt.Printf("   ✓ Found %d duplicate warnings\n", len(duplicateWarnings))

	// 3. Calculate Token Budget and Sample Repository
	fmt.Println("📦 Sampling repository files to fit token budget...")
	fmt.Println("   ⏳ Calculating token budget...")
	// Use larger budget for full repository review
	diffBudget := FULL_REVIEW_TOKEN_BUDGET - SYSTEM_PROMPT_TOKENS - CONTEXT_TOKENS_AVG - BUFFER_TOKENS
	if diffBudget > 25000 {
		diffBudget = 25000 // Maximum
	}

	// Sample the repository using intelligent sampling
	fmt.Println("   ⏳ Initializing repository sampler...")
	sampler := NewRepositorySampler(diffBudget, e.config.Analysis.Sampling.MaxFiles)
	fmt.Printf("   ✓ Sampler initialized (token budget: %d)\n", diffBudget)
	
	// Adjust sampling for very large repositories
	fmt.Println("   ⏳ Adjusting sampler for repository size...")
	sampler.AdjustForLargeRepository(len(diff.Files))
	
	// Count Python files for additional optimization
	fmt.Println("   ⏳ Analyzing file types...")
	pythonFileCount := 0
	for _, file := range diff.Files {
		if strings.HasSuffix(file.Path, ".py") {
			pythonFileCount++
		}
	}
	
	// For Python-heavy repos, be more aggressive with sampling
	if pythonFileCount > len(diff.Files)/2 {
		fmt.Printf("  📊 Python-heavy repository detected (%d Python files). Applying optimized sampling.\n", pythonFileCount)
	}
	
	fmt.Println("   ⏳ Sampling repository files (this may take a moment)...")
	sampledDiff, samplingReport := sampler.SampleRepository(diff, localResult.FileAnalysis)
	fmt.Println("   ✓ Sampling complete")
	fmt.Printf("   ✓ Sampling complete: %d files selected from %d total\n", samplingReport.SampledFiles, samplingReport.TotalFiles)

	// Display sampling report to user
	fmt.Println("\n📊 Sampling Report:")
	fmt.Printf("  • Total files in repository: %d\n", samplingReport.TotalFiles)
	fmt.Printf("  • Filtered: %d (", samplingReport.FilteredFiles)
	reasons := []string{}
	for reason, count := range samplingReport.FilteredReasons {
		reasons = append(reasons, fmt.Sprintf("%s: %d", reason, count))
	}
	fmt.Printf("%s)\n", strings.Join(reasons, ", "))
	fmt.Printf("  • Reviewing: %d files\n", samplingReport.SampledFiles)

	if len(samplingReport.TopRiskFiles) > 0 {
		fmt.Println("\n🔴 High Priority Files:")
		for i, risk := range samplingReport.TopRiskFiles {
			if i >= 10 {
				break
			}
			fmt.Printf("  • %s (Risk: %d) - %s\n", risk.File.Path, risk.Score, strings.Join(risk.Reasons, ", "))
		}
	}
	fmt.Println()

	// 4. Always use chunked review -- the chunker handles both small and large
	// prompts correctly, computing output tokens dynamically per chunk.
	return e.performChunkedFullRepositoryReview(diff, localResult, staticIssues, duplicateWarnings, &classification, sampledDiff, samplingReport, exactDupDetector, dbQueryReviews)
}

// performChunkedFullRepositoryReview handles full repository reviews that exceed token limits by splitting into chunks
func (e *ReviewEngine) performChunkedFullRepositoryReview(
	diff *git.Diff,
	localResult *ReviewResult,
	staticIssues []analysis.Issue,
	duplicateWarnings []string,
	classification *llm.ClassificationResult,
	sampledDiff *SampledDiff,
	samplingReport *SamplingReport,
	exactDupDetector *analysis.ExactDuplicationDetector,
	dbQueryReviews []DBQueryReview,
) (*ReviewReport, error) {
	
	// 1. Create chunks
	modelLimit := llm.GetModelTokenLimitWithConfig(e.llmClient.GetModel(), e.config.LLM.MaxInputTokens)
	chunker := NewReviewChunker(
		modelLimit,
		1000, // buffer
	)
	
	frameworks := []string{} // TODO: Load from context.json if available
	languages := detectLanguages(diff)
	classificationStr := fmt.Sprintf("%s (Confidence: %.2f)", classification.Type, classification.Confidence)
	
	chunks, err := chunker.CreateChunks(sampledDiff, staticIssues, duplicateWarnings, frameworks, languages, classificationStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks: %w", err)
	}
	
	fmt.Printf("📦 Split full repository review into %d chunks for parallel processing\n", len(chunks))
	
	// 2. Schedule chunks into TPM-aware minute-windows, then process each window
	// in parallel using the existing semaphore (max 3 concurrent requests).
	// An overall 20-minute context guards against infinite hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	tpmLimit := e.config.LLM.TokensPerMinute
	windows := scheduleChunkWindows(chunks, tpmLimit)

	if len(windows) > 1 {
		fmt.Printf("⏱  TPM rate limiting active: scheduling %d chunks across %d minute-windows (limit: %d tokens/min)\n",
			len(chunks), len(windows), tpmLimit)
	}

	// Semaphore: limits max concurrent in-flight API calls (unchanged behaviour)
	maxConcurrent := 3
	semaphore := make(chan struct{}, maxConcurrent)

	results := make([]ChunkResult, 0, len(chunks))
	completed := 0

	for winIdx, window := range windows {
		windowStart := time.Now()
		windowChan := make(chan ChunkResult, len(window.Chunks))

		for _, chunk := range window.Chunks {
			// Acquire semaphore
			semaphore <- struct{}{}

			go func(c *ReviewChunk) {
				defer func() { <-semaphore }()

				fmt.Printf("🤖 Reviewing chunk %d/%d...\n", c.ID, len(chunks))

				chunkPrompt := buildChunkPrompt(c)

				// Calculate available tokens for completion
				inputTokens := llm.EstimateTokens(llm.SystemPrompt) + llm.EstimateTokens(chunkPrompt)
				maxOutputTokens := modelLimit - inputTokens - 500 // 500 buffer

				// Cap at reasonable maximum
				if maxOutputTokens > 4096 {
					maxOutputTokens = 4096
				}
				if maxOutputTokens < 512 {
					maxOutputTokens = 512 // Minimum to provide useful response
				}

				// Use per-chunk context with timeout, inheriting the 20-min overall limit
				chunkCtx, chunkCancel := context.WithTimeout(ctx, 5*time.Minute)
				defer chunkCancel()

				resp, err := e.llmClient.GenerateCompletion(chunkCtx, llm.CompletionRequest{
					Messages: []llm.Message{
						{Role: llm.RoleSystem, Content: llm.SystemPrompt},
						{Role: llm.RoleUser, Content: chunkPrompt},
					},
					Temperature: 0.2,
					MaxTokens:   maxOutputTokens,
				})

				result := ChunkResult{ChunkID: c.ID}
				if err != nil {
					if chunkCtx.Err() == context.DeadlineExceeded {
						result.Error = fmt.Errorf("chunk %d review timed out after 5 minutes", c.ID)
					} else {
						result.Error = fmt.Errorf("chunk %d review failed: %w", c.ID, err)
					}
				} else {
					result.Content = resp.Content
					result.InputTokens = resp.Usage.PromptTokens
					result.OutputTokens = resp.Usage.CompletionTokens
				}

				windowChan <- result
			}(chunk)
		}

		// Collect results for this window, respecting the overall timeout
		for range window.Chunks {
			select {
			case result := <-windowChan:
				completed++
				if result.Error != nil {
					fmt.Printf("⚠️  Chunk %d failed: %v\n", result.ChunkID, result.Error)
					// Continue processing other chunks instead of failing immediately
					continue
				}
				results = append(results, result)
			case <-ctx.Done():
				close(windowChan)
				return nil, fmt.Errorf("full repository review timed out after 20 minutes. Only %d/%d chunks completed", completed, len(chunks))
			}
		}
		close(windowChan)

		// If there are more windows, sleep for the remainder of the 60-second window
		// so we don't exceed the tokens-per-minute limit when the next window fires.
		if winIdx < len(windows)-1 {
			elapsed := time.Since(windowStart)
			wait := 60*time.Second - elapsed
			if wait > 0 {
				fmt.Printf("⏳ Window %d/%d done. Waiting %s before next batch to respect TPM limit...\n",
					winIdx+1, len(windows), wait.Round(time.Second))
				time.Sleep(wait)
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all chunks failed during full repository review")
	}
	
	// Sort results by chunk ID to maintain order
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].ChunkID > results[j+1].ChunkID {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
	
	// Extract responses in order
	var allResponses []string
	var totalInputTokens, totalOutputTokens int
	for _, result := range results {
		allResponses = append(allResponses, result.Content)
		totalInputTokens += result.InputTokens
		totalOutputTokens += result.OutputTokens
	}
	
	// 3. Merge chunk results
	fmt.Println("🔄 Merging chunk results...")
	mergedContent := e.mergeChunkResults(allResponses)
	
	// 4. For full repo reviews, don't filter - we want all issues
	// (filterLLMOutputToChanges is for diff reviews only)
	
	// 5. Synthesize final report
	tokenUsage := TokenUsage{
		InputTokens:  totalInputTokens,
		OutputTokens: totalOutputTokens,
		TotalTokens:  totalInputTokens + totalOutputTokens,
	}
	
	report := e.synthesizer.Synthesize(mergedContent, staticIssues, duplicateWarnings, localResult.FileAnalysis, tokenUsage, localResult.SimilarityCheckLimited, localResult.SimilarityCheckReason, dbQueryReviews)
	
	// 6. Populate sampling information
	e.populateSamplingInfo(report, diff, sampledDiff, samplingReport)
	
	// 7. Populate duplicate blocks and AI patterns for HTML report
	e.populateDuplicateBlocks(report, exactDupDetector, localResult, diff)
	if e.config.Review.DetectAICode {
		e.populateAIPatternsFullRepo(report, localResult)
	} else {
		for _, fa := range report.FileAnalysis {
			if fa != nil {
				fa.AIScore = nil
			}
		}
	}

	// 7.5. Generate fix prompt if enabled
	if e.config.Review.GenerateFixPrompt {
		report.FixPrompt = e.synthesizer.BuildFixPrompt(report)
	}
	
	// 8. Generate HTML report if enabled
	if e.shouldGenerateHTML() {
		if err := e.generateHTMLReport(report, diff, e.diffRange); err != nil {
			fmt.Printf("Warning: Failed to generate HTML report: %v\n", err)
		}
	}
	
	return report, nil
}

// populateAIPatternsFullRepo extracts AI patterns from all files (not just changed code)
func (e *ReviewEngine) populateAIPatternsFullRepo(report *ReviewReport, localResult *ReviewResult) {
	if report.AIPatterns == nil {
		report.AIPatterns = make(map[string][]analysis.AICodePattern)
	}

	aiDetector := analysis.NewAICodeDetector()
	for filePath, fileAnalysis := range localResult.FileAnalysis {
		patterns := aiDetector.DetectAIPatterns(fileAnalysis)
		if len(patterns) > 0 {
			report.AIPatterns[filePath] = patterns
		}
	}
}

// Helper function for min

// formatDiff converts structured diff back to string representation
func formatDiff(diff *git.Diff) string {
	var sb strings.Builder
	for _, file := range diff.Files {
		sb.WriteString(fmt.Sprintf("--- %s\n+++ %s\n", file.Path, file.Path))
		if file.Patch != "" {
			sb.WriteString(file.Patch)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// prioritizeIssues sorts issues by severity and returns the top N
// Priority order: Error > Warning > Info
func prioritizeIssues(issues []analysis.Issue, maxCount int) []analysis.Issue {
	if len(issues) <= maxCount {
		return issues
	}
	
	// Separate by severity
	var errors, warnings, infos []analysis.Issue
	for _, issue := range issues {
		switch strings.ToLower(string(issue.Severity)) {
		case "error", "critical", "high":
			errors = append(errors, issue)
		case "warning", "medium":
			warnings = append(warnings, issue)
		default:
			infos = append(infos, issue)
		}
	}
	
	// Build prioritized list
	prioritized := make([]analysis.Issue, 0, maxCount)
	
	// Add all errors first
	for _, issue := range errors {
		if len(prioritized) >= maxCount {
			break
		}
		prioritized = append(prioritized, issue)
	}
	
	// Then warnings
	for _, issue := range warnings {
		if len(prioritized) >= maxCount {
			break
		}
		prioritized = append(prioritized, issue)
	}
	
	// Finally infos
	for _, issue := range infos {
		if len(prioritized) >= maxCount {
			break
		}
		prioritized = append(prioritized, issue)
	}
	
	return prioritized
}

func detectLanguages(diff *git.Diff) []string {
	langs := make(map[string]bool)
	for _, file := range diff.Files {
		if strings.HasSuffix(file.Path, ".go") {
			langs["Go"] = true
		} else if strings.HasSuffix(file.Path, ".java") {
			langs["Java"] = true
		} else if strings.HasSuffix(file.Path, ".ts") || strings.HasSuffix(file.Path, ".tsx") {
			langs["TypeScript"] = true
		} else if strings.HasSuffix(file.Path, ".js") {
			langs["JavaScript"] = true
		} else if strings.HasSuffix(file.Path, ".py") {
			langs["Python"] = true
		}
	}
	
	var result []string
	for l := range langs {
		result = append(result, l)
	}
	return result
}

func summarizeFiles(diff *git.Diff) string {
	var files []string
	for _, f := range diff.Files {
		files = append(files, f.Path)
	}
	return strings.Join(files, ", ")
}

// filterLLMOutputToChanges filters LLM output to only include issues related to actual code changes
func (e *ReviewEngine) filterLLMOutputToChanges(llmOutput string, diff *git.Diff, fileAnalysis map[string]*analysis.FileAnalysis) string {
	// Build a list of functions with actual code changes (not just comments/formatting)
	changedFunctions := e.findActuallyChangedFunctions(diff, fileAnalysis)
	
	if len(changedFunctions) == 0 {
		// If no functions had real code changes, likely only comments/formatting
		// Remove all critical issues from LLM output
		return e.removeCriticalIssues(llmOutput)
	}
	
	// Parse the critical issues section
	issuesSectionRegex := regexp.MustCompile(`(?s)(## Critical Issues.*?\n)(.*?)(##|$)`)
	matches := issuesSectionRegex.FindStringSubmatch(llmOutput)
	
	if len(matches) < 3 {
		// No critical issues section found
		return llmOutput
	}
	
	header := matches[1]
	issuesText := matches[2]
	rest := ""
	if len(matches) > 3 {
		rest = matches[3]
	}
	
	// Filter issues line by line
	filteredIssues := []string{}
	for _, line := range strings.Split(issuesText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "-") {
			// Check if this issue mentions any of the changed functions
			includeIssue := false
			lineLower := strings.ToLower(line)
			
			for funcName := range changedFunctions {
				funcLower := strings.ToLower(funcName)
				// Check if function name is mentioned in the issue
				if strings.Contains(lineLower, funcLower) || strings.Contains(lineLower, "`"+funcLower+"`") {
					includeIssue = true
					break
				}
			}
			
			if includeIssue {
				filteredIssues = append(filteredIssues, line)
			}
		} else {
			// Keep non-issue lines (like subsection headers)
			filteredIssues = append(filteredIssues, line)
		}
	}
	
	// Reconstruct the output
	beforeIssues := llmOutput[:strings.Index(llmOutput, header)]
	afterIssues := ""
	if rest != "" && strings.HasPrefix(rest, "##") {
		afterIssues = llmOutput[strings.Index(llmOutput, rest):]
	}
	
	var result strings.Builder
	result.WriteString(beforeIssues)
	result.WriteString(header)
	
	if len(filteredIssues) == 0 {
		result.WriteString("- None. The changes do not introduce new issues.\n\n")
	} else {
		result.WriteString(strings.Join(filteredIssues, "\n"))
		result.WriteString("\n\n")
	}
	
	result.WriteString(afterIssues)
	
	return result.String()
}

// findActuallyChangedFunctions identifies functions with real code changes (not just comments)
func (e *ReviewEngine) findActuallyChangedFunctions(diff *git.Diff, fileAnalysis map[string]*analysis.FileAnalysis) map[string]bool {
	changedFunctions := make(map[string]bool)
	
	for _, file := range diff.Files {
		analysis, exists := fileAnalysis[file.Path]
		if !exists {
			continue
		}
		
		// Parse the patch to find what changed
		lines := strings.Split(file.Patch, "\n")
		lineNum := 0
		
		for _, line := range lines {
			// Parse @@ header
			if strings.HasPrefix(line, "@@") {
				re := regexp.MustCompile(`\+(\d+)`)
				if match := re.FindStringSubmatch(line); len(match) > 1 {
					fmt.Sscanf(match[1], "%d", &lineNum)
				}
				continue
			}
			
			// Check for actual code changes (not just comments)
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				trimmed := strings.TrimSpace(strings.TrimPrefix(line, "+"))
				
				// Skip pure comment lines
				if trimmed == "" || strings.HasPrefix(trimmed, "//") || 
				   strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || 
				   strings.HasPrefix(trimmed, "#") {
					lineNum++
					continue
				}
				
				// This is an actual code change - find which function it's in
				for _, fn := range analysis.Functions {
					if lineNum >= fn.StartLine && lineNum <= fn.EndLine {
						changedFunctions[fn.Name] = true
						break
					}
				}
				
				lineNum++
			} else if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "\\") {
				lineNum++
			}
		}
	}
	
	return changedFunctions
}

// hasDatabaseOrORMCalls checks if a file analysis contains database/ORM calls
func (e *ReviewEngine) hasDatabaseOrORMCalls(fileAnalysis *analysis.FileAnalysis) bool {
	// Check imports for database/ORM libraries
	for _, imp := range fileAnalysis.Imports {
		importPath := strings.ToLower(imp.Path)
		
		// Python database/ORM imports
		if strings.Contains(importPath, "sqlalchemy") ||
			strings.Contains(importPath, "django.db") ||
			strings.Contains(importPath, "peewee") ||
			strings.Contains(importPath, "sqlite3") ||
			strings.Contains(importPath, "psycopg2") ||
			strings.Contains(importPath, "mysql") ||
			strings.Contains(importPath, "pymongo") ||
			strings.Contains(importPath, "sqlmodel") {
			return true
		}
		
		// JavaScript/TypeScript database/ORM imports
		if strings.Contains(importPath, "sequelize") ||
			strings.Contains(importPath, "prisma") ||
			strings.Contains(importPath, "typeorm") ||
			strings.Contains(importPath, "mongoose") ||
			strings.Contains(importPath, "knex") ||
			strings.Contains(importPath, "bookshelf") {
			return true
		}
		
		// Go database/ORM imports
		if strings.Contains(importPath, "database/sql") ||
			strings.Contains(importPath, "gorm.io/gorm") ||
			strings.Contains(importPath, "github.com/jmoiron/sqlx") ||
			strings.Contains(importPath, "gorm.io/driver") {
			return true
		}
		
		// Java database/ORM imports
		if strings.Contains(importPath, "javax.persistence") ||
			strings.Contains(importPath, "org.hibernate") ||
			strings.Contains(importPath, "org.springframework.data") ||
			strings.Contains(importPath, "jakarta.persistence") {
			return true
		}
	}
	
	// Check function bodies for database/ORM method calls
	dbMethodPatterns := []string{
		`\.query\(`, `\.filter\(`, `\.get\(`, `\.save\(`, `\.delete\(`, `\.create\(`, `\.update\(`,
		`execute\(`, `cursor\(`, `\.session\(`, `\.commit\(`, `\.rollback\(`,
		`\.findOne\(`, `\.findAll\(`, `\.create\(`, `\.update\(`, `\.destroy\(`, `\.save\(`, `\.query\(`,
		`\.find\(`, `\.findById\(`, `\.findOneAndUpdate\(`, `\.findOneAndDelete\(`,
		`\.Query\(`, `\.QueryRow\(`, `\.Exec\(`, `\.First\(`, `\.Find\(`, `\.Create\(`, `\.Save\(`,
		`\.Update\(`, `\.Delete\(`, `\.Where\(`, `\.Select\(`,
		`\.save\(`, `\.findById\(`, `\.findAll\(`, `\.delete\(`, `\.persist\(`, `\.merge\(`,
		`\.createQuery\(`, `\.getResultList\(`, `\.executeUpdate\(`,
	}
	
	for _, fn := range fileAnalysis.Functions {
		if fn.Body == "" {
			continue
		}
		
		bodyLower := strings.ToLower(fn.Body)
		for _, pattern := range dbMethodPatterns {
			re := regexp.MustCompile("(?i)" + pattern)
			if re.MatchString(fn.Body) {
				return true
			}
		}
		
		if strings.Contains(bodyLower, "db.query") ||
			strings.Contains(bodyLower, "db.execute") ||
			strings.Contains(bodyLower, "db.session") ||
			strings.Contains(bodyLower, "model.save") ||
			strings.Contains(bodyLower, "model.create") ||
			strings.Contains(bodyLower, "model.update") ||
			strings.Contains(bodyLower, "model.delete") ||
			strings.Contains(bodyLower, "orm.query") ||
			strings.Contains(bodyLower, "orm.save") {
			return true
		}
	}
	
	return false
}

// convertDBQueryIssues converts analysis.DBQueryIssue to review.DBQueryIssue
func (e *ReviewEngine) convertDBQueryIssues(issues []analysis.DBQueryIssue) []DBQueryIssue {
	result := make([]DBQueryIssue, len(issues))
	for i, issue := range issues {
		result[i] = DBQueryIssue{
			Type:        issue.Type,
			Severity:    issue.Severity,
			Description: issue.Description,
			Suggestion:  issue.Suggestion,
		}
	}
	return result
}

// removeCriticalIssues removes the critical issues section when no real code changes exist
func (e *ReviewEngine) removeCriticalIssues(llmOutput string) string {
	issuesSectionRegex := regexp.MustCompile(`(?s)(## Critical Issues.*?\n)(.*?)(##|$)`)
	return issuesSectionRegex.ReplaceAllString(llmOutput, "$1- None. The changes do not introduce new issues.\n\n$3")
}

// populateDuplicateBlocks extracts structured duplicate block information
func (e *ReviewEngine) populateDuplicateBlocks(report *ReviewReport, exactDupDetector *analysis.ExactDuplicationDetector, localResult *ReviewResult, diff *git.Diff) {
	// Extract from exact duplicates
	for _, dups := range exactDupDetector.GetDuplicates() {
		if len(dups) < 2 {
			continue
		}
		// Use first as original, rest as duplicates
		original := dups[0]
		for i := 1; i < len(dups); i++ {
			dup := dups[i]
			report.DuplicateBlocks = append(report.DuplicateBlocks, DuplicateBlockInfo{
				OriginalFile:   original.FilePath,
				OriginalStart:  original.StartLine,
				OriginalEnd:    original.EndLine,
				DuplicateFile:   dup.FilePath,
				DuplicateStart: dup.StartLine,
				DuplicateEnd:   dup.EndLine,
				Similarity:     1.0, // Exact duplicate
				Lines:          original.EndLine - original.StartLine + 1,
			})
		}
	}

	// Extract from similarity results
	for _, simResults := range localResult.Duplicates {
		for _, simResult := range simResults {
			// Find the original function in the diff
			for _, file := range diff.Files {
				if file.Path == simResult.FilePath {
					// Try to find matching function in file analysis
					if analysis, exists := localResult.FileAnalysis[file.Path]; exists {
						for _, fn := range analysis.Functions {
							if fn.Name == simResult.FuncName {
								report.DuplicateBlocks = append(report.DuplicateBlocks, DuplicateBlockInfo{
									OriginalFile:    file.Path,
									OriginalStart:   fn.StartLine,
									OriginalEnd:     fn.EndLine,
									DuplicateFile:    simResult.FilePath,
									DuplicateStart:   simResult.StartLine,
									DuplicateEnd:     simResult.EndLine,
									Similarity:      float64(simResult.Similarity),
									Lines:           fn.EndLine - fn.StartLine + 1,
								})
								break
							}
						}
					}
					break
				}
			}
		}
	}
}

// populateAIPatterns extracts AI patterns from file analysis, filtering to only changed code
func (e *ReviewEngine) populateAIPatterns(report *ReviewReport, localResult *ReviewResult, diff *git.Diff) {
	if report.AIPatterns == nil {
		report.AIPatterns = make(map[string][]analysis.AICodePattern)
	}

	// Extract changed line ranges from diff
	changedRanges := e.extractChangedLineRanges(diff)

	aiDetector := analysis.NewAICodeDetector()
	for filePath, fileAnalysis := range localResult.FileAnalysis {
		// Get all AI patterns for this file
		allPatterns := aiDetector.DetectAIPatterns(fileAnalysis)
		
		// Filter patterns to only include those in changed code sections
		filteredPatterns := make([]analysis.AICodePattern, 0)
		fileRanges, hasChanges := changedRanges[filePath]
		
		// Find the diff file to check its status
		var diffFile *git.DiffFile
		for _, df := range diff.Files {
			if df.Path == filePath {
				diffFile = df
				break
			}
		}
		
		// Handle edge cases
		if diffFile != nil {
			if diffFile.Status == "D" {
				// Deleted file - skip AI analysis
				continue
			}
			if diffFile.Status == "A" {
				// New file - all lines are changed, include all patterns
				filteredPatterns = allPatterns
			} else if hasChanges {
				// Modified file - only include patterns in changed sections
				for _, pattern := range allPatterns {
					if e.isPatternInChangedRange(pattern.StartLine, pattern.EndLine, fileRanges) {
						filteredPatterns = append(filteredPatterns, pattern)
					}
				}
			}
		} else if hasChanges {
			// File exists but not in diff (shouldn't happen, but handle gracefully)
			for _, pattern := range allPatterns {
				if e.isPatternInChangedRange(pattern.StartLine, pattern.EndLine, fileRanges) {
					filteredPatterns = append(filteredPatterns, pattern)
				}
			}
		}
		
		if len(filteredPatterns) > 0 {
			report.AIPatterns[filePath] = filteredPatterns
		}
	}
}

// extractChangedLineRanges parses git diff patches to extract changed line ranges
func (e *ReviewEngine) extractChangedLineRanges(diff *git.Diff) map[string][]LineRange {
	changedRanges := make(map[string][]LineRange)
	
	for _, file := range diff.Files {
		if file.Patch == "" {
			continue
		}
		
		ranges := make([]LineRange, 0)
		lines := strings.Split(file.Patch, "\n")
		currentNewLine := 0
		inChangedBlock := false
		blockStart := 0
		
		for _, line := range lines {
			// Parse @@ header to track line numbers
			// Format: @@ -old_start,old_count +new_start,new_count @@
			if strings.HasPrefix(line, "@@") {
				// Extract new file line number
				re := regexp.MustCompile(`\+(\d+)`)
				if match := re.FindStringSubmatch(line); len(match) > 1 {
					fmt.Sscanf(match[1], "%d", &currentNewLine)
					// Close previous block if open
					if inChangedBlock {
						ranges = append(ranges, LineRange{Start: blockStart, End: currentNewLine - 1})
						inChangedBlock = false
					}
				}
				continue
			}
			
			// Skip file header lines
			if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
				continue
			}
			
			// Track added/modified lines (lines starting with +)
			if strings.HasPrefix(line, "+") {
				if !inChangedBlock {
					// Start of a new changed block
					blockStart = currentNewLine
					inChangedBlock = true
				}
				currentNewLine++
			} else if strings.HasPrefix(line, "-") {
				// Removed line - don't increment current line in new file
				// Close block if we were tracking changes
				if inChangedBlock {
					ranges = append(ranges, LineRange{Start: blockStart, End: currentNewLine - 1})
					inChangedBlock = false
				}
			} else if strings.HasPrefix(line, "\\") {
				// Continuation marker (for binary files or no newline at EOF)
				// Don't increment line number
				continue
			} else {
				// Context line (unchanged)
				if inChangedBlock {
					// End of changed block
					ranges = append(ranges, LineRange{Start: blockStart, End: currentNewLine - 1})
					inChangedBlock = false
				}
				currentNewLine++
			}
		}
		
		// Close any open block at the end
		if inChangedBlock {
			ranges = append(ranges, LineRange{Start: blockStart, End: currentNewLine - 1})
		}
		
		if len(ranges) > 0 {
			changedRanges[file.Path] = ranges
		}
	}
	
	return changedRanges
}

// isPatternInChangedRange checks if a pattern's line range overlaps with any changed range
func (e *ReviewEngine) isPatternInChangedRange(patternStart, patternEnd int, changedRanges []LineRange) bool {
	for _, r := range changedRanges {
		// Overlap logic: pattern overlaps if patternStart <= rangeEnd && patternEnd >= rangeStart
		if patternStart <= r.End && patternEnd >= r.Start {
			return true
		}
	}
	return false
}

// populateSamplingInfo populates sampling information in the report
func (e *ReviewEngine) populateSamplingInfo(report *ReviewReport, diff *git.Diff, sampledDiff *SampledDiff, samplingReport *SamplingReport) {
	// Create a map of reviewed file paths
	reviewedFiles := make(map[string]bool)
	for _, sampledFile := range sampledDiff.Files {
		reviewedFiles[sampledFile.Path] = true
	}

	// Determine which files were reviewed and which were ignored
	reviewedPaths := make([]string, 0)
	ignoredFiles := make([]IgnoredFile, 0)

	for _, file := range diff.Files {
		if reviewedFiles[file.Path] {
			reviewedPaths = append(reviewedPaths, file.Path)
		} else {
			// Determine why this file was ignored
			reason := e.determineIgnoreReason(file)
			ignoredFiles = append(ignoredFiles, IgnoredFile{
				Path:   file.Path,
				Reason: reason,
			})
		}
	}

	report.SamplingInfo = &SamplingInfo{
		TotalFiles:      samplingReport.TotalFiles,
		ReviewedFiles:   reviewedPaths,
		IgnoredFiles:    ignoredFiles,
		FilteredReasons: samplingReport.FilteredReasons,
	}
}

// determineIgnoreReason determines why a file was ignored during sampling
func (e *ReviewEngine) determineIgnoreReason(file *git.DiffFile) string {
	// Check various filter criteria
	if file.Additions == 0 && file.Deletions == 0 {
		return "no_changes"
	}
	if strings.HasPrefix(file.Path, ".") || strings.Contains(file.Path, "/.") {
		return "hidden"
	}
	if isGeneratedFile(file.Path) {
		return "generated"
	}
	if isLockFile(file.Path) {
		return "lock_file"
	}
	if isPackageManagementFile(file.Path) {
		return "package_management"
	}
	if isBinaryFile(file.Path) {
		return "binary"
	}
	if isTestFile(file.Path) {
		return "test"
	}
	// If none of the above, it was likely filtered due to token budget or low risk score
	return "low_priority"
}

// shouldGenerateHTML checks if HTML generation should be enabled
func (e *ReviewEngine) shouldGenerateHTML() bool {
	return e.config.Review.GenerateHTML
}

// shouldGenerateGFM checks if GFM generation should be enabled
func (e *ReviewEngine) shouldGenerateGFM() bool {
	return e.config.Review.GenerateGFM
}

// generateHTMLReport generates and saves HTML report
func (e *ReviewEngine) generateHTMLReport(report *ReviewReport, diff *git.Diff, diffRange string) error {
	// Get repo root for file reading
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	// Collect file paths
	filePaths := make([]string, 0)
	for filePath := range report.FileAnalysis {
		filePaths = append(filePaths, filePath)
	}

	// Read file contents
	fileReader := NewFileReader(repo.RootPath)
	fileContents, err := fileReader.ReadFiles(filePaths)
	if err != nil {
		// Continue with partial content
	}

	// Extract diff information
	diffInfo := &DiffInfo{}
	
	// Get project name
	projectName, err := e.config.GetProjectName()
	if err != nil {
		// Try to extract from Git repository
		if repo != nil {
			if name, repoErr := repo.GetProjectName(); repoErr == nil {
				projectName = name
			}
		}
	}
	diffInfo.ProjectName = projectName
	
	// Check if this is a full repository review
	if diffRange == "full repository" {
		diffInfo.IsFullRepository = true
		diffInfo.Range = "full repository"
	} else if diffRange != "" {
		diffInfo.Range = diffRange
		// Extract commit SHAs from range (e.g., "main..feature" -> get SHAs for both)
		if strings.Contains(diffRange, "..") {
			parts := strings.Split(diffRange, "..")
			if len(parts) == 2 {
				// Get commit SHAs for both sides
				if fromCommit, err := repo.GetCommit(parts[0]); err == nil {
					diffInfo.FromCommit = fromCommit.ShortHash
				}
				if toCommit, err := repo.GetCommit(parts[1]); err == nil {
					diffInfo.ToCommit = toCommit.ShortHash
				}
			}
		}
	} else if diff.Commit != nil {
		// For single commit review, show parent and current commit
		diffInfo.ToCommit = diff.Commit.ShortHash
		// Try to get parent commit
		if parentCommit, err := repo.GetCommit(diff.Commit.Hash + "^"); err == nil {
			diffInfo.FromCommit = parentCommit.ShortHash
		}
	}
	
	// Generate HTML
	formatter := NewFormatter()
	htmlContent := formatter.FormatHTML(report, fileContents, diffInfo)

	// Determine output path
	outputPath := e.config.Review.HTMLOutputPath
	if outputPath == "" {
		outputPath = ".katich/reports"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("review-%s.html", timestamp)
	fullPath := filepath.Join(outputPath, filename)

	// Write file
	if err := os.WriteFile(fullPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	fmt.Printf("📄 HTML report saved to: %s\n", fullPath)
	return nil
}

// generateGFMReport generates and saves GFM report
func (e *ReviewEngine) generateGFMReport(report *ReviewReport, diff *git.Diff, diffRange string) error {
	// Get repo root for file reading
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	// Collect file paths
	filePaths := make([]string, 0)
	for filePath := range report.FileAnalysis {
		filePaths = append(filePaths, filePath)
	}

	// Read file contents
	fileReader := NewFileReader(repo.RootPath)
	fileContents, err := fileReader.ReadFiles(filePaths)
	if err != nil {
		// Continue with partial content
	}

	// Extract diff information
	diffInfo := &DiffInfo{}
	
	// Get project name
	projectName, err := e.config.GetProjectName()
	if err != nil {
		// Try to extract from Git repository
		if repo != nil {
			if name, repoErr := repo.GetProjectName(); repoErr == nil {
				projectName = name
			}
		}
	}
	diffInfo.ProjectName = projectName
	
	// Check if this is a full repository review
	if diffRange == "full repository" {
		diffInfo.IsFullRepository = true
		diffInfo.Range = "full repository"
	} else if diffRange != "" {
		diffInfo.Range = diffRange
		// Extract commit SHAs from range (e.g., "main..feature" -> get SHAs for both)
		if strings.Contains(diffRange, "..") {
			parts := strings.Split(diffRange, "..")
			if len(parts) == 2 {
				// Get commit SHAs for both sides
				if fromCommit, err := repo.GetCommit(parts[0]); err == nil {
					diffInfo.FromCommit = fromCommit.ShortHash
				}
				if toCommit, err := repo.GetCommit(parts[1]); err == nil {
					diffInfo.ToCommit = toCommit.ShortHash
				}
			}
		}
	} else if diff.Commit != nil {
		// For single commit review, show parent and current commit
		diffInfo.ToCommit = diff.Commit.ShortHash
		// Try to get parent commit
		if parentCommit, err := repo.GetCommit(diff.Commit.Hash + "^"); err == nil {
			diffInfo.FromCommit = parentCommit.ShortHash
		}
	}
	
	// Generate GFM
	formatter := NewFormatter()
	gfmContent := formatter.FormatGFM(report, fileContents, diffInfo)

	// Determine output path
	outputPath := e.config.Review.GFMOutputPath
	if outputPath == "" {
		outputPath = ".katich/reports"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("review-%s.md", timestamp)
	fullPath := filepath.Join(outputPath, filename)

	// Write file
	if err := os.WriteFile(fullPath, []byte(gfmContent), 0644); err != nil {
		return fmt.Errorf("failed to write GFM file: %w", err)
	}

	fmt.Printf("📄 GFM report saved to: %s\n", fullPath)
	return nil
}

