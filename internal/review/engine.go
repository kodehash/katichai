package review

import (
	"context"
	"fmt"
	"regexp"
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
	
	// 5. Query LLM
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

	// 6. Filter LLM output to only issues related to actual code changes
	filteredLLMOutput := e.filterLLMOutputToChanges(resp.Content, diff, localResult.FileAnalysis)

	// 7. Synthesize Report with token usage
	tokenUsage := TokenUsage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}
	report := e.synthesizer.Synthesize(filteredLLMOutput, staticIssues, duplicateWarnings, localResult.FileAnalysis, tokenUsage)

	return report, nil
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

// removeCriticalIssues removes the critical issues section when no real code changes exist
func (e *ReviewEngine) removeCriticalIssues(llmOutput string) string {
	issuesSectionRegex := regexp.MustCompile(`(?s)(## Critical Issues.*?\n)(.*?)(##|$)`)
	return issuesSectionRegex.ReplaceAllString(llmOutput, "$1- None. The changes do not introduce new issues.\n\n$3")
}

