package review

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/katichai/katich/internal/analysis"
)

// ReviewReport represents the final synthesized review
type ReviewReport struct {
	Summary              string                            `json:"summary"`
	Score                int                               `json:"score"`
	Status               string                            `json:"status"` // PASS or FAIL
	Issues               []ReviewIssue                      `json:"issues"`
	Suggestions          []string                          `json:"suggestions"`
	FileAnalysis         map[string]*analysis.FileAnalysis `json:"file_analysis,omitempty"`
	TokensUsed           TokenUsage                        `json:"tokens_used"`
	DuplicateBlocks      []DuplicateBlockInfo              `json:"duplicate_blocks,omitempty"`
	AIPatterns           map[string][]analysis.AICodePattern `json:"ai_patterns,omitempty"`
	SamplingInfo         *SamplingInfo                     `json:"sampling_info,omitempty"`
	ComplexityIssues     []ComplexityIssue                 `json:"complexity_issues,omitempty"`
	SimilarityCheckLimited bool                            `json:"similarity_check_limited,omitempty"`
	SimilarityCheckReason  string                          `json:"similarity_check_reason,omitempty"`
	DBQueryReviews        []DBQueryReview                  `json:"db_query_reviews,omitempty"`
	FixPrompt             string                          `json:"fix_prompt,omitempty"`
}

// ComplexityIssue represents an unnecessarily complex code block or architectural pattern
type ComplexityIssue struct {
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Function    string  `json:"function"`
	Score       float64 `json:"score"` // 0.0-1.0, higher = more unnecessarily complex
	Description string  `json:"description"`
	Reasoning   string  `json:"reasoning"` // Detailed explanation of why it's unnecessarily complex
	Suggestion  string  `json:"suggestion"`
	Type        string  `json:"type"` // "code" or "architectural"
}

// DBQueryReview represents a review of database queries
type DBQueryReview struct {
	File            string  `json:"file"`
	Line            int     `json:"line"`
	Function        string  `json:"function"`
	QueryType       string  `json:"query_type"` // "ORM" or "RAW"
	EfficiencyScore float64 `json:"efficiency_score"` // 0.0-1.0
	Issues          []DBQueryIssue `json:"issues"`
	Recommendation  string  `json:"recommendation"`
	QuerySnippet    string  `json:"query_snippet"` // Code snippet showing the query
}

// DBQueryIssue represents a specific issue with a query
type DBQueryIssue struct {
	Type        string `json:"type"` // "N+1", "MISSING_INDEX", "INEFFICIENT_JOIN", "RAW_PREFERRED", "SELECT_ALL", "NO_LIMIT", etc.
	Severity    string `json:"severity"` // "CRITICAL", "WARNING", "INFO"
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// SamplingInfo contains information about which files were reviewed and which were ignored
type SamplingInfo struct {
	TotalFiles      int                      `json:"total_files"`
	ReviewedFiles   []string                 `json:"reviewed_files"`
	IgnoredFiles    []IgnoredFile            `json:"ignored_files"`
	FilteredReasons map[string]int           `json:"filtered_reasons"`
}

// IgnoredFile represents a file that was ignored during sampling
type IgnoredFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// DuplicateBlockInfo represents structured duplicate code information
type DuplicateBlockInfo struct {
	OriginalFile  string  `json:"original_file"`
	OriginalStart int     `json:"original_start"`
	OriginalEnd   int     `json:"original_end"`
	DuplicateFile string  `json:"duplicate_file"`
	DuplicateStart int    `json:"duplicate_start"`
	DuplicateEnd   int    `json:"duplicate_end"`
	Similarity    float64 `json:"similarity"`
	Lines         int     `json:"lines"`
}

// TokenUsage represents token consumption for the review
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ReviewIssue represents an issue found during review
type ReviewIssue struct {
	Category    string `json:"category"`    // SECURITY, ARCHITECTURE, PERFORMANCE, CODE_QUALITY, STATIC_ANALYSIS
	Subcategory string `json:"subcategory"` // For STATIC_ANALYSIS: complexity, function_length, naming, etc.
	Severity    string `json:"severity"`    // CRITICAL, WARNING, INFO
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
}

// Synthesizer aggregates review signals
type Synthesizer struct{}

// NewSynthesizer creates a new synthesizer
func NewSynthesizer() *Synthesizer {
	return &Synthesizer{}
}

// Synthesize combines LLM output with static analysis
func (s *Synthesizer) Synthesize(llmOutput string, staticIssues []analysis.Issue, duplicates []string, fileAnalysis map[string]*analysis.FileAnalysis, tokensUsed TokenUsage, similarityLimited bool, similarityReason string, dbQueryReviews []DBQueryReview) *ReviewReport {
	report := &ReviewReport{
		// Status:       "PASS",  // Commented out - scoring is subjective
		// Score:        100,      // Commented out - scoring is subjective
		FileAnalysis:           fileAnalysis,
		TokensUsed:             tokensUsed,
		SimilarityCheckLimited: similarityLimited,
		SimilarityCheckReason:  similarityReason,
	}

	// 1. Parse LLM Output
	s.parseLLMOutput(llmOutput, report)

	// 1.5. Check for summary/issues mismatch and warn
	s.checkSummaryIssuesMismatch(report, llmOutput)

	// 1.6. Deduplicate security issues
	s.deduplicateSecurityIssues(report)

	// 2. Incorporate Static Analysis
	for _, issue := range staticIssues {
		// Convert analysis.Issue to ReviewIssue
		// We might want to deduplicate if LLM found the same thing, but simple append is safer now
		rIssue := ReviewIssue{
			Category:    "STATIC_ANALYSIS",
			Subcategory: string(issue.Type), // complexity, function_length, naming, etc.
			Severity:    strings.ToUpper(string(issue.Severity)),
			Description: issue.Message,
			Location:    issue.File,
		}
		report.Issues = append(report.Issues, rIssue)
		
		// Static analysis issues are informational only - no score penalty

	}

	// 3. Incorporate Duplicates
	for _, dup := range duplicates {
		report.Issues = append(report.Issues, ReviewIssue{
			Category:    "CODE_QUALITY",
			Severity:    "WARNING",
			Description: dup,
		})
		// report.Score -= 5  // Commented out - scoring is subjective
	}

	// 4. Parse Complexity Issues from LLM output
	s.parseComplexityIssues(llmOutput, report)

	// 5. Deduplicate complexity issues
	s.deduplicateComplexityIssues(report)

	// 6. Sort issues by priority: SECURITY > BREAKING > ARCHITECTURE > PERFORMANCE > others
	s.sortIssuesByPriority(report)

	// 7. Final adjustments (commented out - scoring is subjective)
	// if report.Score < 0 {
	// 	report.Score = 0
	// }
	// if report.Score < 70 {
	// 	report.Status = "FAIL"
	// }

	return report
}

// parseComplexityIssues extracts complexity issues from LLM output
func (s *Synthesizer) parseComplexityIssues(output string, report *ReviewReport) {
	// Look for "## Unnecessary Complexity" section
	complexityRe := regexp.MustCompile(`(?s)## Unnecessary Complexity\s+(.*?)(##|$)`)
	if match := complexityRe.FindStringSubmatch(output); len(match) > 1 {
		content := match[1]
		// Split by lines but keep track of which issue we're parsing
		lines := strings.Split(content, "\n")
		
		var currentIssue *ComplexityIssue
		var currentField string
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			// Check if this is a new issue (starts with "- [")
			if strings.HasPrefix(line, "- [") {
				// Save previous issue if exists
				if currentIssue != nil {
					report.ComplexityIssues = append(report.ComplexityIssues, *currentIssue)
				}
				
				// Start new issue
				trimmed := strings.TrimPrefix(line, "- ")
				
				// Extract type (COMPLEXITY or ARCHITECTURAL)
				issueType := "code"
				if strings.Contains(trimmed, "[ARCHITECTURAL]") {
					issueType = "architectural"
				}
				
				// Extract location (file:line)
				locationRe := regexp.MustCompile(`(\S+):(\d+)`)
				locationMatch := locationRe.FindStringSubmatch(trimmed)
				if len(locationMatch) < 3 {
					currentIssue = nil
					continue
				}
				filePath := locationMatch[1]
				lineNum := 0
				fmt.Sscanf(locationMatch[2], "%d", &lineNum)
				
				// Extract function name if present
				functionName := ""
				funcRe := regexp.MustCompile(`Function:\s*(\S+)`)
				if funcMatch := funcRe.FindStringSubmatch(trimmed); len(funcMatch) > 1 {
					functionName = funcMatch[1]
				}
				
				// Extract score
				score := 0.0
				scoreRe := regexp.MustCompile(`Score:\s*(\d+)/100`)
				if scoreMatch := scoreRe.FindStringSubmatch(trimmed); len(scoreMatch) > 1 {
					var scoreInt int
					fmt.Sscanf(scoreMatch[1], "%d", &scoreInt)
					score = float64(scoreInt) / 100.0
				}
				
				currentIssue = &ComplexityIssue{
					File:        filePath,
					Line:        lineNum,
					Function:    functionName,
					Score:       score,
					Description: "",
					Reasoning:   "",
					Suggestion:  "",
					Type:        issueType,
				}
				currentField = ""
			} else if currentIssue != nil {
				// This is a continuation line for the current issue
				// Check if it's a labeled field
				if strings.HasPrefix(line, "Description:") {
					currentField = "description"
					currentIssue.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
				} else if strings.HasPrefix(line, "Reasoning:") {
					currentField = "reasoning"
					currentIssue.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "Reasoning:"))
				} else if strings.HasPrefix(line, "Suggestion:") {
					currentField = "suggestion"
					currentIssue.Suggestion = strings.TrimSpace(strings.TrimPrefix(line, "Suggestion:"))
				} else {
					// Continuation of the current field
					if currentField == "description" && currentIssue.Description != "" {
						currentIssue.Description += " " + line
					} else if currentField == "reasoning" && currentIssue.Reasoning != "" {
						currentIssue.Reasoning += " " + line
					} else if currentField == "suggestion" && currentIssue.Suggestion != "" {
						currentIssue.Suggestion += " " + line
					} else if currentField == "" {
						// No field set yet, assume it's description
						if currentIssue.Description == "" {
							currentIssue.Description = line
						}
					}
				}
			}
		}
		
		// Don't forget the last issue
		if currentIssue != nil {
			report.ComplexityIssues = append(report.ComplexityIssues, *currentIssue)
		}
	}
}

// deduplicateComplexityIssues removes duplicate complexity issues
func (s *Synthesizer) deduplicateComplexityIssues(report *ReviewReport) {
	seen := make(map[string]bool)
	unique := make([]ComplexityIssue, 0)
	
	for _, issue := range report.ComplexityIssues {
		// Create a signature based on file, line, and normalized description
		signature := fmt.Sprintf("%s:%d:%s", issue.File, issue.Line, strings.ToLower(issue.Description))
		if !seen[signature] {
			seen[signature] = true
			unique = append(unique, issue)
		}
	}
	
	report.ComplexityIssues = unique
}

func (s *Synthesizer) parseLLMOutput(output string, report *ReviewReport) {
	// Simple regex parsing for the Markdown sections 
	// This assumes the prompt instructions were followed

	// Extract Summary
	summaryRe := regexp.MustCompile(`(?s)## Summary\s+(.*?)\s+##`)
	if match := summaryRe.FindStringSubmatch(output); len(match) > 1 {
		report.Summary = strings.TrimSpace(match[1])
	}

	// Extract Critical Issues
	// Try multiple variations of the section header to be more robust
	// Pattern 1: "## Critical Issues" or "## Critical Issues (Blockers)" with optional whitespace
	// Note: Go regexp doesn't support lookahead, so we use a simpler pattern
	issuesRe := regexp.MustCompile(`(?s)##\s*Critical Issues[^\n]*\n(.*?)(\n##|$)`)
	match := issuesRe.FindStringSubmatch(output)
	
	// Fallback: Try without requiring newline after header (in case LLM doesn't add one)
	if len(match) < 2 {
		issuesRe = regexp.MustCompile(`(?s)##\s*Critical Issues[^\n]*(.*?)(\n##|$)`)
		match = issuesRe.FindStringSubmatch(output)
	}
	
	if len(match) > 1 {
		criticalSection := match[1]
		// Debug: log what we extracted
		if len(strings.TrimSpace(criticalSection)) < 50 {
			fmt.Printf("🔍 Debug: Critical Issues section content: %q\n", strings.TrimSpace(criticalSection))
		}
		
		lines := strings.Split(criticalSection, "\n")
		parsedCount := 0
		skippedCount := 0
		multiLineIssueCount := 0
		
		// State machine to handle multi-line issue descriptions
		var currentIssue *ReviewIssue
		var currentIssueHasContinuation bool = false
		totalLines := len(lines)
		
		for i, line := range lines {
			line = strings.TrimSpace(line)
			
			// Check if this is a new section header (shouldn't happen, but be safe)
			if strings.HasPrefix(line, "##") {
				// Save current issue if exists
				if currentIssue != nil {
					// Finalize the current issue
					location := s.extractLocation(currentIssue.Description)
					if location != "" {
						currentIssue.Location = location
					}
					report.Issues = append(report.Issues, *currentIssue)
					parsedCount++
					currentIssue = nil
				}
				break
			}
			
			// Check if this is a new issue (starts with "-")
			if strings.HasPrefix(line, "-") {
				// Save previous issue if exists
				if currentIssue != nil {
					// Finalize the previous issue
					location := s.extractLocation(currentIssue.Description)
					if location != "" {
						currentIssue.Location = location
					}
					
					// Skip if description is "None" or empty (LLM indicating no issues)
					descLower := strings.ToLower(strings.TrimSpace(currentIssue.Description))
					if currentIssue.Description == "" || descLower == "none" || descLower == "none." || 
					   strings.HasPrefix(descLower, "none.") || strings.HasPrefix(descLower, "no issues") {
						skippedCount++
						currentIssue = nil
						currentIssueHasContinuation = false
					} else {
						// Track if this issue had multi-line description
						if currentIssueHasContinuation {
							multiLineIssueCount++
						}
						report.Issues = append(report.Issues, *currentIssue)
						parsedCount++
						currentIssue = nil
						currentIssueHasContinuation = false
					}
				}
				
				// Parse new issue line: "- [CATEGORY] Description"
				trimmed := strings.TrimPrefix(line, "- ")
				category := "GENERAL"
				desc := trimmed

				if len(trimmed) > 0 && string(trimmed[0]) == "[" {
					endIdx := strings.Index(trimmed, "]")
					if endIdx > 0 {
						category = trimmed[1:endIdx]
						desc = strings.TrimSpace(trimmed[endIdx+1:])
					}
				}

				// Skip if description is "None" or empty (LLM indicating no issues)
				descLower := strings.ToLower(desc)
				if desc == "" || descLower == "none" || descLower == "none." || 
				   strings.HasPrefix(descLower, "none.") || strings.HasPrefix(descLower, "no issues") {
					skippedCount++
					currentIssue = nil
					continue
				}
				
				// Start new issue
				currentIssue = &ReviewIssue{
					Category:    category,
					Severity:    "CRITICAL", // Assuming critical section
					Description: desc,
					Location:    "", // Will be extracted after accumulating all lines
				}
				currentIssueHasContinuation = false
			} else if currentIssue != nil {
				// This is a continuation line for the current issue
				if line != "" {
					// Accumulate the continuation line
					if currentIssue.Description != "" {
						currentIssue.Description += " " + line
					} else {
						currentIssue.Description = line
					}
					currentIssueHasContinuation = true
				} else {
					// Empty line - check if next line is a new issue or section
					// If next line is empty or starts with "-" or "##", finalize current issue
					if i+1 < totalLines {
						nextLine := strings.TrimSpace(lines[i+1])
						if nextLine == "" || strings.HasPrefix(nextLine, "-") || strings.HasPrefix(nextLine, "##") {
							// Finalize current issue
							location := s.extractLocation(currentIssue.Description)
							if location != "" {
								currentIssue.Location = location
							}
							
							descLower := strings.ToLower(strings.TrimSpace(currentIssue.Description))
							if currentIssue.Description == "" || descLower == "none" || descLower == "none." || 
							   strings.HasPrefix(descLower, "none.") || strings.HasPrefix(descLower, "no issues") {
								skippedCount++
							} else {
								// Track if this issue had multi-line description
								if currentIssueHasContinuation {
									multiLineIssueCount++
								}
								report.Issues = append(report.Issues, *currentIssue)
								parsedCount++
							}
							currentIssue = nil
							currentIssueHasContinuation = false
						}
						// Otherwise, continue accumulating (empty line is part of description)
					}
				}
			}
		}
		
		// Don't forget the last issue if we were still building one
		if currentIssue != nil {
			location := s.extractLocation(currentIssue.Description)
			if location != "" {
				currentIssue.Location = location
			}
			
			descLower := strings.ToLower(strings.TrimSpace(currentIssue.Description))
			if currentIssue.Description == "" || descLower == "none" || descLower == "none." || 
			   strings.HasPrefix(descLower, "none.") || strings.HasPrefix(descLower, "no issues") {
				skippedCount++
			} else {
				// Track if this issue had multi-line description
				if currentIssueHasContinuation {
					multiLineIssueCount++
				}
				report.Issues = append(report.Issues, *currentIssue)
				parsedCount++
			}
		}
		
		// Debug: log parsing results
		if parsedCount == 0 && skippedCount > 0 {
			fmt.Printf("⚠️  Warning: Found %d critical issue lines but all were skipped (likely 'None' or empty)\n", skippedCount)
		} else if parsedCount > 0 {
			fmt.Printf("✅ Parsed %d critical issues from LLM output", parsedCount)
			if multiLineIssueCount > 0 {
				fmt.Printf(" (%d issues had multi-line descriptions)", multiLineIssueCount)
			}
			fmt.Printf("\n")
		} else if parsedCount == 0 && totalLines > 5 {
			// Warn if section had content but nothing was parsed (might indicate parsing issue)
			fmt.Printf("⚠️  Warning: Critical Issues section had %d lines but no issues were parsed. This may indicate a parsing issue or LLM output format problem.\n", totalLines)
			// Show a sample of what was found for debugging
			if len(criticalSection) > 0 {
				sampleLen := len(criticalSection)
				if sampleLen > 200 {
					sampleLen = 200
				}
				fmt.Printf("   🔍 Sample content: %q\n", strings.TrimSpace(criticalSection[:sampleLen]))
			}
		}
	} else {
		// Try to find alternative section headers
		altPatterns := []string{
			`##\s*Critical\s+Issues\s*\(Blockers\)`,
			`##\s*Critical\s+Issues`,
			`##\s*Issues`,
			`##\s*Blockers`,
		}
		
		foundAlt := false
		for _, pattern := range altPatterns {
			// Go regexp doesn't support lookahead, use simpler pattern
			altRe := regexp.MustCompile(`(?s)` + pattern + `[^\n]*\n(.*?)(\n##|$)`)
			if altMatch := altRe.FindStringSubmatch(output); len(altMatch) > 1 {
				previewLen := len(altMatch[1])
				if previewLen > 200 {
					previewLen = 200
				}
				fmt.Printf("⚠️  Warning: Found section matching pattern '%s' but it wasn't parsed. Content preview: %q\n", 
					pattern, strings.TrimSpace(altMatch[1][:previewLen]))
				foundAlt = true
				break
			}
		}
		
		if !foundAlt {
			fmt.Printf("⚠️  Warning: Could not find '## Critical Issues' section in LLM output\n")
		}
	}

	// Extract Suggestions
	suggestionsRe := regexp.MustCompile(`(?s)## Suggestions.*?\n(.*?)(##|$)`)
	if match := suggestionsRe.FindStringSubmatch(output); len(match) > 1 {
		lines := strings.Split(match[1], "\n")
		var pending string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "-") {
				// Continuation line: append to the last suggestion if any
				if line != "" && len(report.Suggestions) > 0 {
					report.Suggestions[len(report.Suggestions)-1] += " " + line
				}
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			// Detect heading-only lines like "**Refactor Complex Functions:**"
			// These end with ":" or ":**" and contain mostly bold markers
			stripped := strings.ReplaceAll(item, "*", "")
			stripped = strings.TrimSpace(stripped)
			if strings.HasSuffix(stripped, ":") && len(stripped) < 80 {
				// This is a heading — hold it and merge with the next item
				pending = stripped + " "
				continue
			}
			// Strip bold markers from the item itself
			item = strings.ReplaceAll(item, "**", "")
			report.Suggestions = append(report.Suggestions, pending+item)
			pending = ""
		}
		// If a heading was the last line with no following content
		if pending != "" {
			report.Suggestions = append(report.Suggestions, strings.TrimSuffix(pending, " "))
		}
	}
	
	// Extract Score Override (if LLM provided one)
	// For now, we calculate our own based on findings to be deterministic
}

// extractLocation extracts file path and line number from issue description
func (s *Synthesizer) extractLocation(description string) string {
	// Look for patterns like:
	// - "file.go:123"
	// - "in file.go:123"
	// - "at file.go:123"
	// - "file.go, line 123"
	// - "(file.go:123)"
	
	// Pattern 1: file.go:123 or file.go:123:456
	re1 := regexp.MustCompile(`([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)):(\d+)`)
	if match := re1.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	// Pattern 2: "in file.go, line 123" or "at file.go, line 123"
	re2 := regexp.MustCompile(`(?:in|at)\s+([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)),\s*line\s+(\d+)`)
	if match := re2.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	// Pattern 3: "(file.go:123)"
	re3 := regexp.MustCompile(`\(([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)):(\d+)\)`)
	if match := re3.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	return ""
}

// deduplicateSecurityIssues removes duplicate security findings
// checkSummaryIssuesMismatch detects when Summary mentions issues but Critical Issues section is empty
func (s *Synthesizer) checkSummaryIssuesMismatch(report *ReviewReport, llmOutput string) {
	// Count non-static issues (critical issues)
	criticalCount := 0
	for _, issue := range report.Issues {
		if issue.Category != "STATIC_ANALYSIS" && issue.Severity == "CRITICAL" {
			criticalCount++
		}
	}
	
	// Check if summary mentions critical issues
	summaryLower := strings.ToLower(report.Summary)
	mentionsIssues := false
	issueKeywords := []string{
		"critical", "vulnerability", "vulnerabilities", "security", 
		"architectural violation", "breaking", "severe",
	}
	
	for _, keyword := range issueKeywords {
		if strings.Contains(summaryLower, keyword) {
			mentionsIssues = true
			break
		}
	}
	
	// If summary mentions issues but we have none, try to extract from summary
	if mentionsIssues && criticalCount == 0 {
		// Try to extract issues from the summary itself as a fallback
		// Look for patterns like "X security vulnerabilities" or "critical issues in file.go"
		s.tryExtractIssuesFromSummary(report.Summary, report)
	}
}

// tryExtractIssuesFromSummary attempts to extract critical issues from the summary text
// Returns true if any issues were extracted
func (s *Synthesizer) tryExtractIssuesFromSummary(summary string, report *ReviewReport) bool {
	// Look for file:line patterns in the summary
	locationRe := regexp.MustCompile(`([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)):(\d+)`)
	matches := locationRe.FindAllStringSubmatch(summary, -1)
	
	if len(matches) == 0 {
		return false
	}
	
	// Try to extract sentences that mention issues with file locations
	lines := strings.Split(summary, ".")
	extractedCount := 0
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if this line mentions a critical issue keyword and has a file location
		lineLower := strings.ToLower(line)
		hasKeyword := false
		category := "GENERAL"
		
		if strings.Contains(lineLower, "security") || strings.Contains(lineLower, "vulnerability") {
			hasKeyword = true
			category = "SECURITY"
		} else if strings.Contains(lineLower, "architectural") || strings.Contains(lineLower, "architecture") {
			hasKeyword = true
			category = "ARCHITECTURE"
		} else if strings.Contains(lineLower, "performance") {
			hasKeyword = true
			category = "PERFORMANCE"
		} else if strings.Contains(lineLower, "breaking") {
			hasKeyword = true
			category = "BREAKING"
		} else if strings.Contains(lineLower, "critical") {
			hasKeyword = true
		}
		
		if hasKeyword {
			// Check if this line has a file location
			locationMatch := locationRe.FindStringSubmatch(line)
			if len(locationMatch) > 0 {
				location := fmt.Sprintf("%s:%s", locationMatch[1], locationMatch[3])
				
				// Extract the issue description (limit to reasonable length)
				desc := line
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				
				report.Issues = append(report.Issues, ReviewIssue{
					Category:    category,
					Severity:    "CRITICAL",
					Description: desc,
					Location:    location,
				})
				extractedCount++
			}
		}
	}
	
	if extractedCount > 0 {
		fmt.Printf("📝 Extracted %d critical issue(s) from summary text (issues were mentioned in summary but not in Critical Issues section)\n", extractedCount)
		return true
	}
	
	return false
}

// issueCategoryPriority returns a sort priority for a ReviewIssue category.
// Lower number = higher priority (shown first).
func issueCategoryPriority(category string) int {
	switch strings.ToUpper(category) {
	case "SECURITY":
		return 0
	case "BREAKING":
		return 1
	case "ARCHITECTURE":
		return 2
	case "PERFORMANCE":
		return 3
	case "STATIC_ANALYSIS":
		return 5
	default:
		return 4
	}
}

// sortIssuesByPriority sorts report.Issues so that higher-priority categories come first.
// Within the same category the original LLM ordering is preserved (stable sort).
func (s *Synthesizer) sortIssuesByPriority(report *ReviewReport) {
	sort.SliceStable(report.Issues, func(i, j int) bool {
		return issueCategoryPriority(report.Issues[i].Category) <
			issueCategoryPriority(report.Issues[j].Category)
	})
}

// BuildFixPrompt generates a lean, prescriptive prompt that users can paste
// into Cursor, Antigravity, or any AI coding assistant to fix review findings.
func (s *Synthesizer) BuildFixPrompt(report *ReviewReport) string {
	var sb strings.Builder

	sb.WriteString("Fix the following issues found during code review.\n\n")

	// Collect non-STATIC_ANALYSIS issues grouped by category in priority order
	type catEntry struct {
		key   string
		label string
	}
	categoryOrder := []catEntry{
		{"SECURITY", "Security"},
		{"BREAKING", "Breaking Changes"},
		{"ARCHITECTURE", "Architecture"},
		{"PERFORMANCE", "Performance"},
	}

	grouped := make(map[string][]ReviewIssue)
	for _, issue := range report.Issues {
		cat := strings.ToUpper(issue.Category)
		if cat == "STATIC_ANALYSIS" {
			continue
		}
		if cat == "" {
			cat = "GENERAL"
		}
		grouped[cat] = append(grouped[cat], issue)
	}

	rendered := map[string]bool{}
	issueNum := 1

	for _, entry := range categoryOrder {
		issues := grouped[entry.key]
		if len(issues) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("[%s]\n", entry.label))
		for _, issue := range issues {
			sb.WriteString(s.formatFixInstruction(issueNum, issue))
			issueNum++
		}
		sb.WriteString("\n")
		rendered[entry.key] = true
	}

	for cat, issues := range grouped {
		if rendered[cat] {
			continue
		}
		sb.WriteString(fmt.Sprintf("[%s]\n", cat))
		for _, issue := range issues {
			sb.WriteString(s.formatFixInstruction(issueNum, issue))
			issueNum++
		}
		sb.WriteString("\n")
	}

	// Suggestions as actionable items
	if len(report.Suggestions) > 0 {
		sb.WriteString("[Suggestions]\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", issueNum, sug))
			issueNum++
		}
		sb.WriteString("\n")
	}

	// Safety guardrails — always appended
	sb.WriteString("Constraints:\n")
	sb.WriteString("- Do not break any existing functionality or tests.\n")
	sb.WriteString("- Do not remove or rename public APIs without user approval.\n")
	sb.WriteString("- Preserve existing coding patterns and directory structure.\n")
	sb.WriteString("- If a fix requires a significant refactor, ask the user before proceeding.\n")
	sb.WriteString("- Run the test suite after changes to verify nothing is broken.\n")
	sb.WriteString("- If unsure about a fix, explain trade-offs and let the user decide.\n")

	return sb.String()
}

// formatFixInstruction turns a ReviewIssue into a prescriptive instruction.
func (s *Synthesizer) formatFixInstruction(num int, issue ReviewIssue) string {
	loc := ""
	if issue.Location != "" {
		loc = " at " + issue.Location
	}
	return fmt.Sprintf("%d. Fix%s: %s\n", num, loc, issue.Description)
}

func (s *Synthesizer) deduplicateSecurityIssues(report *ReviewReport) {
	// Separate security issues from others
	securityIssues := make([]ReviewIssue, 0)
	otherIssues := make([]ReviewIssue, 0)
	
	for _, issue := range report.Issues {
		if strings.ToUpper(issue.Category) == "SECURITY" {
			securityIssues = append(securityIssues, issue)
		} else {
			otherIssues = append(otherIssues, issue)
		}
	}
	
	// Deduplicate security issues
	deduplicated := s.deduplicateIssues(securityIssues)
	
	// Rebuild issues list
	report.Issues = append(otherIssues, deduplicated...)
}

// deduplicateIssues removes duplicate issues based on similarity
func (s *Synthesizer) deduplicateIssues(issues []ReviewIssue) []ReviewIssue {
	if len(issues) == 0 {
		return issues
	}
	
	// Create a map to track unique issues
	// Key: normalized issue signature (category + location + normalized description)
	seen := make(map[string]bool)
	deduplicated := make([]ReviewIssue, 0)
	
	for _, issue := range issues {
		// Create a signature for this issue
		signature := s.createIssueSignature(issue)
		
		// Check if we've seen a similar issue
		if seen[signature] {
			continue // Skip duplicate
		}
		
		// Check for semantic similarity with existing issues
		isDuplicate := false
		for _, existingIssue := range deduplicated {
			if s.areIssuesSimilar(issue, existingIssue) {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			seen[signature] = true
			deduplicated = append(deduplicated, issue)
		}
	}
	
	return deduplicated
}

// createIssueSignature creates a unique signature for an issue
func (s *Synthesizer) createIssueSignature(issue ReviewIssue) string {
	// Normalize description (lowercase, remove extra spaces)
	normalizedDesc := strings.ToLower(issue.Description)
	normalizedDesc = regexp.MustCompile(`\s+`).ReplaceAllString(normalizedDesc, " ")
	normalizedDesc = strings.TrimSpace(normalizedDesc)
	
	// Extract key terms (first 50 chars or first sentence)
	keyTerms := normalizedDesc
	if len(keyTerms) > 50 {
		// Try to get first sentence
		if idx := strings.Index(keyTerms, "."); idx > 0 && idx < 50 {
			keyTerms = keyTerms[:idx]
		} else {
			keyTerms = keyTerms[:50]
		}
	}
	
	// Combine category, location, and key terms
	return fmt.Sprintf("%s|%s|%s", strings.ToUpper(issue.Category), issue.Location, keyTerms)
}

// areIssuesSimilar checks if two issues are semantically similar (duplicates)
func (s *Synthesizer) areIssuesSimilar(issue1, issue2 ReviewIssue) bool {
	// Same category required
	if strings.ToUpper(issue1.Category) != strings.ToUpper(issue2.Category) {
		return false
	}
	
	// Normalize descriptions
	normalized1 := strings.ToLower(issue1.Description)
	normalized1 = regexp.MustCompile(`\s+`).ReplaceAllString(normalized1, " ")
	normalized1 = strings.TrimSpace(normalized1)
	
	normalized2 := strings.ToLower(issue2.Description)
	normalized2 = regexp.MustCompile(`\s+`).ReplaceAllString(normalized2, " ")
	normalized2 = strings.TrimSpace(normalized2)
	
	// If same location, check for high similarity
	if issue1.Location != "" && issue2.Location != "" && issue1.Location == issue2.Location {
		// Same location - check if descriptions are very similar
		return s.calculateSimilarity(normalized1, normalized2) > 0.7
	}
	
	// If different locations but very similar descriptions, might be duplicate
	// Only for security issues - be more strict
	if strings.ToUpper(issue1.Category) == "SECURITY" {
		similarity := s.calculateSimilarity(normalized1, normalized2)
		// For security issues, require 85% similarity to consider duplicate
		return similarity > 0.85
	}
	
	return false
}

// calculateSimilarity calculates word-based similarity between two strings
func (s *Synthesizer) calculateSimilarity(text1, text2 string) float64 {
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Create word sets (normalize to lowercase, remove very short words)
	set1 := make(map[string]int)
	set2 := make(map[string]int)
	
	for _, w := range words1 {
		w = strings.ToLower(strings.Trim(w, ".,!?;:()[]{}"))
		if len(w) > 2 {
			set1[w]++
		}
	}
	
	for _, w := range words2 {
		w = strings.ToLower(strings.Trim(w, ".,!?;:()[]{}"))
		if len(w) > 2 {
			set2[w]++
		}
	}
	
	// Calculate Jaccard similarity (intersection over union)
	intersection := 0
	union := len(set1) + len(set2)
	
	for word := range set1 {
		if set2[word] > 0 {
			intersection++
			union-- // Remove from union count since it's in both
		}
	}
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}
