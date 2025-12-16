package llm

import (
	"fmt"
	"strings"

	"github.com/katichai/katich/internal/analysis"
)

// SystemPrompt defines the persona and core instructions for the AI reviewer
const SystemPrompt = `You are a Senior Principal Software Engineer and Security Architect.
Your role is to review code changes with a strict focus on the following pillars:

1. ARCHITECTURAL CONSISTENCY (Top Priority)
   - Enforce existing architectural patterns (e.g., Spring Controller-Service-Repo, Clean Architecture).
   - Verify code is placed in the correct directory structure.
   - Ensure specific libraries already in use are reused; reject unnecessary new dependencies.
   - Flag any deviation from the repository's coding style and conventions.

2. SECURITY (OWASP Top 10)
   - Aggressively hunt for security vulnerabilities, specifically:
     * Injection (SQL, Command, Log)
     * Broken Authentication/Authorization
     * Sensitive Data Exposure (secrets in code/logs)
     * Insecure Deserialization
     * XML External Entities (XXE)
   - Flag missing input validation or sanitization.

3. EFFICIENCY & PERFORMANCE
   - Identify potential MEMORY LEAKS (unclosed resources, goroutine leaks, static referencing).
   - Flag HIGH MEMORY EXECUTIONS (loading large files into memory, inefficient slice usage).
   - Detect algorithmic inefficiencies (O(n^2) loops on large datasets).

4. CODE QUALITY
   - Detect "AI Slop": Boilerplate, hallucinatory APIs, or over-engineered abstractions.
   - Ignore minor formatting/nitpicks unless they severely violate readability.

Output your review in the following Markdown format:
## Summary
(Brief executive summary of the changes)

## Critical Issues (Blockers)
- [SECURITY/ARCH/PERF] Details of the issue...

## Suggestions
- Details of improvements...

## Score
(0-100 Confidence Score on readiness to merge)
`

// ReviewContext contains all information needed to build a review prompt
type ReviewContext struct {
	Diff             string
	Frameworks       []string
	Languages        []string
	StaticIssues     []analysis.Issue
	SimilarCode      []string                  // Descriptions of similar code found
	FileContext      string                    // Summary of file locations/structure
	Classification   string                    // Change classification (e.g. LOGIC, BOILERPLATE)
	AIGeneratedFiles []*analysis.AIFileScore   // Files with AI-generated code
}

// PromptBuilder handles the construction of LLM prompts
type PromptBuilder struct{}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildReviewPrompt constructs the user message for the code review
func (p *PromptBuilder) BuildReviewPrompt(ctx ReviewContext) string {
	var sb strings.Builder

	sb.WriteString("Please review the following code changes.\n\n")

	if ctx.Classification != "" {
		sb.WriteString(fmt.Sprintf("**Change Classification**: %s\n\n", ctx.Classification))
	}

	// 1. Context Injection
	sb.WriteString("### Repository Context\n")
	if len(ctx.Languages) > 0 {
		sb.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(ctx.Languages, ", ")))
	}
	if len(ctx.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("- Frameworks: %s\n", strings.Join(ctx.Frameworks, ", ")))
	}
	if ctx.FileContext != "" {
		sb.WriteString(fmt.Sprintf("- Structure: %s\n", ctx.FileContext))
	}
	sb.WriteString("\n")

	// 2. Static Analysis signals (if any)
	if len(ctx.StaticIssues) > 0 {
		sb.WriteString("### Static Analysis Findings (Verify these)\n")
		for _, issue := range ctx.StaticIssues {
			sb.WriteString(fmt.Sprintf("- [%s] %s (Line %d): %s\n", 
				issue.Severity, issue.Type, issue.Line, issue.Message))
		}
		sb.WriteString("\n")
	}

	// 3. Similarity/Duplication signals
	if len(ctx.SimilarCode) > 0 {
		sb.WriteString("### Potential Duplication Detected\n")
		for _, match := range ctx.SimilarCode {
			sb.WriteString(fmt.Sprintf("- %s\n", match))
		}
		sb.WriteString("\n")
	}

	// 3.5. AI-Generated Code Detection (focus review here)
	if len(ctx.AIGeneratedFiles) > 0 {
		sb.WriteString("### ⚠️ AI-Generated Code Detected (Pay Extra Attention)\n")
		sb.WriteString("The following files contain likely AI-generated code. Review carefully for:\n")
		sb.WriteString("- Logic errors or unhandled edge cases\n")
		sb.WriteString("- Domain-specific requirements that AI may have missed\n")
		sb.WriteString("- Unnecessary verbosity or over-engineering\n")
		sb.WriteString("- Generic error handling that may not fit your patterns\n\n")

		for _, aiFile := range ctx.AIGeneratedFiles {
			sb.WriteString(fmt.Sprintf("**`%s`** (%.0f%% AI-generated, %.0f%% confidence)\n",
				aiFile.FilePath,
				aiFile.AIPercentage,
				aiFile.OverallConfidence*100))

			if len(aiFile.FunctionScores) > 0 {
				for _, fn := range aiFile.FunctionScores {
					// Show top 3 indicators
					indicatorCount := len(fn.Indicators)
					if indicatorCount > 3 {
						indicatorCount = 3
					}
					indicators := strings.Join(fn.Indicators[:indicatorCount], ", ")

					sb.WriteString(fmt.Sprintf("  - Function `%s` (%.0f%% confidence): %s",
						fn.FunctionName,
						fn.AIConfidence*100,
						indicators))

					if len(fn.Indicators) > 3 {
						sb.WriteString(fmt.Sprintf(" +%d more", len(fn.Indicators)-3))
					}
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	// 4. The Diff
	sb.WriteString("### Code Changes (Diff)\n")
	sb.WriteString("```diff\n")
	sb.WriteString(ctx.Diff)
	sb.WriteString("\n```\n")

	sb.WriteString("\nBased on the above, provide your architectural and security review.")
	
	return sb.String()
}
