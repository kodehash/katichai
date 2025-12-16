package review

import (
	"context"
	"fmt"
	"strings"

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
}

// NewEngine creates a new ReviewEngine
func NewEngine(cfg *config.Config, reviewer *Reviewer) (*ReviewEngine, error) {
	// Initialize LLM Client
	client, err := llm.NewClient(cfg.LLM)
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
func (e *ReviewEngine) Review(diff *git.Diff) (*ReviewReport, error) {
	// 1. Run Local Analysis (Static + Similarity)
	// This uses the existing Reviewer logic
	localResult, err := e.reviewer.ReviewDiff(diff)
	if err != nil {
		return nil, fmt.Errorf("local analysis failed: %w", err)
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

	// Initialize detectors
	driftDetector := analysis.NewDriftDetector()
	exactDupDetector := analysis.NewExactDuplicationDetector()

	// Create a map of actually changed files for validation
	changedFileMap := make(map[string]bool)
	for _, file := range diff.Files {
		if file.Additions > 0 || file.Deletions > 0 {
			changedFileMap[file.Path] = true
		}
	}

	// Populate exact match index (should be done from context in real app, here we demo on local files)
	// For now we skip repo-wide indexing for speed and just check self-duplication in diff for demo

	for path, fAnalysis := range localResult.FileAnalysis {
		// Extra safety: only include issues from files that actually changed
		if !changedFileMap[path] {
			continue
		}
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
	fmt.Printf("  • Token reduction: %s\n", samplingReport.Summary())
	
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
	
	// 4. Build Review Context for LLM with sampled diff
	diffString := sampledDiff.Format()
	
	reviewCtx := llm.ReviewContext{
		Diff:           diffString,
		Frameworks:     []string{}, // TODO: Load from context.json if available
		Languages:      detectLanguages(diff),
		StaticIssues:   staticIssues,
		SimilarCode:    duplicateWarnings,
		FileContext:    fmt.Sprintf("%s (Sampled: %d/%d files)", summarizeFiles(diff), samplingReport.SampledFiles, samplingReport.TotalFiles),
		Classification: fmt.Sprintf("%s (Confidence: %.2f)", classification.Type, classification.Confidence),
	}

	// 5. Generate Prompt
	prompt := e.pmtBuilder.BuildReviewPrompt(reviewCtx)
	
	// 6. Final token validation
	totalTokens := llm.EstimateTokens(llm.SystemPrompt) + llm.EstimateTokens(prompt)
	if totalTokens > TOTAL_INPUT_BUDGET {
		return nil, fmt.Errorf("input too large after sampling (%d tokens, limit: %d). Try committing smaller changes", totalTokens, TOTAL_INPUT_BUDGET)
	}
	
	fmt.Printf("📏 Token usage: %d/%d (System: ~%d, Context: ~%d, Diff: ~%d)\n\n", 
		totalTokens, TOTAL_INPUT_BUDGET,
		llm.EstimateTokens(llm.SystemPrompt),
		llm.EstimateTokens(prompt) - llm.EstimateTokens(diffString),
		llm.EstimateTokens(diffString))
	
	// 7. Query LLM
	// We'll use a large context window for the review
	fmt.Println("🤖 Querying LLM for review...")
	resp, err := e.llmClient.GenerateCompletion(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: llm.SystemPrompt},
			{Role: llm.RoleUser, Content: prompt},
		},
		Temperature: 0.2, // Low temp for more analytical output
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM review failed: %w", err)
	}

	// 8. Synthesize Report
	report := e.synthesizer.Synthesize(resp.Content, staticIssues, duplicateWarnings)

	return report, nil
}

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
