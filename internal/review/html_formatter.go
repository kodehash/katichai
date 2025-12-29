package review

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/katichai/katich/internal/analysis"
)

// FormatHTML generates a JaCoCo-style HTML report
func (f *Formatter) FormatHTML(report *ReviewReport, fileContents map[string]string, diffInfo *DiffInfo) string {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Katich AI Code Review Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism-tomorrow.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js"></script>
    <style>
        .line-highlight-ai { background-color: rgba(255, 255, 0, 0.2); }
        .line-highlight-issue { border-left: 3px solid #ef4444; padding-left: 8px; }
        .line-highlight-duplicate { border-left: 3px solid #3b82f6; padding-left: 8px; }
        code[class*="language-"] { font-size: 14px; }
        .file-tree-item:hover { background-color: #f3f4f6; cursor: pointer; }
    </style>
</head>
<body class="bg-gray-50">
    <div class="flex h-screen overflow-hidden">
        <!-- Sidebar Navigation -->
        <div class="w-64 bg-white border-r border-gray-200 overflow-y-auto">
            <div class="p-4 border-b border-gray-200">
                <h2 class="text-lg font-bold text-gray-800">Navigation</h2>
            </div>
            <nav class="p-2">
                <a href="#dashboard" class="block px-3 py-2 rounded hover:bg-gray-100 mb-2 text-sm font-medium text-gray-700">📊 Dashboard</a>
                <a href="#critical-issues" class="block px-3 py-2 rounded hover:bg-gray-100 mb-2 text-sm font-medium text-gray-700">⚠️ Critical Issues</a>
                {{if .HasDuplicates}}
                <a href="#duplicates" class="block px-3 py-2 rounded hover:bg-gray-100 mt-2 text-sm font-medium text-gray-700">🔄 Duplicate Code</a>
                {{end}}
                {{if .HasAI}}
                <a href="#ai-analysis" class="block px-3 py-2 rounded hover:bg-gray-100 mt-2 text-sm font-medium text-gray-700">🤖 AI Analysis</a>
                {{end}}
                <a href="#static-analysis" class="block px-3 py-2 rounded hover:bg-gray-100 mt-2 text-sm font-medium text-gray-700">📊 Static Analysis</a>
            </nav>
        </div>

        <!-- Main Content -->
        <div class="flex-1 overflow-y-auto">
            <!-- Header -->
            <div class="bg-white border-b border-gray-200 p-6">
                <div class="flex items-center justify-between">
                    <div>
                        <h1 class="text-3xl font-bold text-gray-900">Katich AI Code Review Report</h1>
                        <p class="text-gray-600 mt-1">Generated on {{.Timestamp}}</p>
                        {{if .DiffInfo}}
                        <p class="text-gray-500 text-sm mt-1">
                            {{if .DiffInfo.Range}}
                            <span class="font-medium">Range:</span> {{.DiffInfo.Range}}
                            {{if and .DiffInfo.FromCommit .DiffInfo.ToCommit}}
                            <span class="ml-4">({{.DiffInfo.FromCommit}}..{{.DiffInfo.ToCommit}})</span>
                            {{end}}
                            {{else if .DiffInfo.ToCommit}}
                            <span class="font-medium">Commits:</span>
                            {{if .DiffInfo.FromCommit}}
                            <span>{{.DiffInfo.FromCommit}}..{{.DiffInfo.ToCommit}}</span>
                            {{else}}
                            <span>{{.DiffInfo.ToCommit}}</span>
                            {{end}}
                            {{end}}
                        </p>
                        {{end}}
                    </div>
                </div>
                {{if .TokensUsed}}
                <div class="mt-4 text-sm text-gray-600">
                    <span class="font-medium">Tokens:</span> {{.TokensUsed.InputTokens}} input + {{.TokensUsed.OutputTokens}} output = <strong>{{.TokensUsed.TotalTokens}} total</strong>
                </div>
                {{end}}
            </div>

            <!-- Dashboard -->
            <section id="dashboard" class="p-6">
                <h2 class="text-2xl font-bold text-gray-900 mb-4">📊 Dashboard</h2>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white p-4 rounded-lg shadow">
                        <div class="text-sm text-gray-600">Total Files</div>
                        <div class="text-2xl font-bold text-gray-900">{{.FileCount}}</div>
                    </div>
                    <div class="bg-white p-4 rounded-lg shadow">
                        <div class="text-sm text-gray-600">Critical Issues</div>
                        <div class="text-2xl font-bold text-red-600">{{.CriticalIssueCount}}</div>
                    </div>
                    <div class="bg-white p-4 rounded-lg shadow">
                        <div class="text-sm text-gray-600">AI-Generated Files</div>
                        <div class="text-2xl font-bold text-yellow-600">{{.AIFileCount}}</div>
                    </div>
                    <div class="bg-white p-4 rounded-lg shadow">
                        <div class="text-sm text-gray-600">Duplicates</div>
                        <div class="text-2xl font-bold text-blue-600">{{.DuplicateCount}}</div>
                    </div>
                </div>
                {{if .Summary}}
                <div class="bg-white p-4 rounded-lg shadow">
                    <h3 class="font-semibold text-gray-900 mb-2">Summary</h3>
                    <p class="text-gray-700 whitespace-pre-wrap">{{.Summary}}</p>
                </div>
                {{end}}
            </section>

            <!-- Suggestions -->
            {{if .Suggestions}}
            <section id="suggestions" class="p-6 border-t border-gray-200">
                <button onclick="toggleSection('suggestions-content')" class="flex items-center justify-between w-full text-left">
                    <div>
                        <h2 class="text-2xl font-bold text-gray-900">💡 Suggestions</h2>
                        <p class="text-sm text-gray-500 mt-1">{{len .Suggestions}} suggestion{{if ne (len .Suggestions) 1}}s{{end}}</p>
                    </div>
                    <span class="text-gray-500" id="suggestions-toggle">▼</span>
                </button>
                <div id="suggestions-content" class="mt-4" style="display: block;">
                    <div class="space-y-3">
                        {{range .Suggestions}}
                        <div class="bg-white p-4 rounded-lg shadow border-l-4 border-green-500">
                            {{if .Heading}}
                            <div class="font-semibold text-gray-900 mb-2">{{.Heading}}</div>
                            {{end}}
                            {{if .Description}}
                            <div class="text-gray-700">{{.Description}}</div>
                            {{end}}
                        </div>
                        {{end}}
                    </div>
                </div>
            </section>
            {{end}}

            <!-- Critical Issues -->
            {{if .CriticalIssues}}
            <section id="critical-issues" class="p-6 border-t border-gray-200">
                <button onclick="toggleSection('critical-issues-content')" class="flex items-center justify-between w-full text-left">
                    <div>
                        <h2 class="text-2xl font-bold text-gray-900">⚠️ Critical Issues</h2>
                        <p class="text-sm text-gray-500 mt-1">{{len .CriticalIssues}} issue{{if ne (len .CriticalIssues) 1}}s{{end}}</p>
                    </div>
                    <span class="text-gray-500" id="critical-issues-toggle">▼</span>
                </button>
                <div id="critical-issues-content" class="mt-4" style="display: block;">
                <div class="space-y-3">
                    {{range .CriticalIssues}}
                    <div class="bg-white p-4 rounded-lg shadow border-l-4 {{.BorderClass}}">
                        <div class="flex items-start justify-between">
                            <div class="flex-1">
                                <div class="flex items-center gap-2 mb-2">
                                    <span class="px-2 py-1 text-xs font-semibold rounded {{.CategoryClass}}">{{.Category}}</span>
                                    <span class="px-2 py-1 text-xs font-semibold rounded {{.SeverityClass}}">{{.Severity}}</span>
                                </div>
                                <p class="text-gray-800">{{.Description}}</p>
                                {{if .Location}}
                                <p class="text-sm text-gray-600 mt-2">
                                    <a href="#file-{{.LocationFileID}}" class="text-blue-600 hover:underline">{{.Location}}</a>
                                </p>
                                {{end}}
                            </div>
                        </div>
                    </div>
                    {{end}}
                </div>
            </section>
            {{end}}


            <!-- Duplicate Code -->
            {{if .DuplicateBlocks}}
            <section id="duplicates" class="p-6 border-t border-gray-200">
                <button onclick="toggleSection('duplicates-content')" class="flex items-center justify-between w-full text-left">
                    <div>
                        <h2 class="text-2xl font-bold text-gray-900">🔄 Duplicate Code</h2>
                        <p class="text-sm text-gray-500 mt-1">{{len .DuplicateBlocks}} duplicate block{{if ne (len .DuplicateBlocks) 1}}s{{end}}</p>
                    </div>
                    <span class="text-gray-500" id="duplicates-toggle">▶</span>
                </button>
                <div id="duplicates-content" class="mt-4" style="display: none;">
                {{range .DuplicateBlocks}}
                <div class="bg-white rounded-lg shadow mb-6 p-4">
                    <div class="mb-4">
                        <div class="text-sm text-gray-600 mb-2">
                            Similarity: <span class="font-semibold">{{printf "%.1f" (multiply .Similarity 100)}}%</span>
                            | Lines: <span class="font-semibold">{{.Lines}}</span>
                        </div>
                    </div>
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <div class="text-sm font-semibold text-gray-700 mb-2">Original: {{.OriginalFile}}</div>
                            <div class="bg-gray-50 rounded p-2 text-xs text-gray-600">Lines {{.OriginalStart}}-{{.OriginalEnd}}</div>
                            {{if .OriginalCode}}
                            <pre class="bg-gray-900 rounded-lg p-3 mt-2 overflow-x-auto text-sm"><code class="language-go">{{.OriginalCode}}</code></pre>
                            {{end}}
                        </div>
                        <div>
                            <div class="text-sm font-semibold text-gray-700 mb-2">Duplicate: {{.DuplicateFile}}</div>
                            <div class="bg-gray-50 rounded p-2 text-xs text-gray-600">Lines {{.DuplicateStart}}-{{.DuplicateEnd}}</div>
                            {{if .DuplicateCode}}
                            <pre class="bg-gray-900 rounded-lg p-3 mt-2 overflow-x-auto text-sm"><code class="language-go">{{.DuplicateCode}}</code></pre>
                            {{end}}
                        </div>
                    </div>
                </div>
                {{end}}
                </div>
            </section>
            {{end}}

            <!-- AI-Generated Code Analysis -->
            {{if .AIFiles}}
            <section id="ai-analysis" class="p-6 border-t border-gray-200">
                <button onclick="toggleSection('ai-analysis-content')" class="flex items-center justify-between w-full text-left">
                    <div>
                        <h2 class="text-2xl font-bold text-gray-900">🤖 AI-Generated Code Analysis</h2>
                        <p class="text-sm text-gray-500 mt-1">{{len .AIFiles}} file{{if ne (len .AIFiles) 1}}s{{end}} with AI-generated code</p>
                    </div>
                    <span class="text-gray-500" id="ai-analysis-toggle">▶</span>
                </button>
                <div id="ai-analysis-content" class="mt-4" style="display: none;">
                {{range .AIFiles}}
                <div class="bg-white rounded-lg shadow mb-6">
                    <div class="p-4 border-b border-gray-200">
                        <h3 class="text-lg font-semibold text-gray-900">{{.FilePath}}</h3>
                        <div class="mt-2 text-sm text-gray-600">
                            <span>AI-Generated: <strong class="text-yellow-600">{{printf "%.1f" .AIScore.AIPercentage}}%</strong></span>
                            <span class="ml-4">({{.AIScore.AIGeneratedLOC}}/{{.AIScore.TotalLOC}} lines)</span>
                            <span class="ml-4">Overall Confidence: <strong>{{printf "%.0f" (multiply .AIScore.OverallConfidence 100)}}%</strong></span>
                        </div>
                    </div>
                    {{if .AIScore.FunctionScores}}
                    <div class="p-4">
                        <h4 class="font-semibold text-gray-900 mb-3">AI-Generated Functions</h4>
                        <div class="space-y-3">
                            {{range .AIScore.FunctionScores}}
                            <div class="bg-gray-50 p-4 rounded border-l-4 border-yellow-400">
                                <div class="flex items-start justify-between">
                                    <div class="flex-1">
                                        <div class="font-medium text-gray-900 text-lg">{{.FunctionName}}</div>
                                        <div class="text-sm text-gray-600 mt-1">
                                            Lines {{.StartLine}}-{{.EndLine}} | 
                                            LOC: {{.LOC}} | 
                                            Confidence: <strong class="text-yellow-600">{{printf "%.0f" (multiply .AIConfidence 100)}}%</strong>
                                        </div>
                                        {{if .Indicators}}
                                        <div class="mt-3">
                                            <div class="text-xs font-semibold text-gray-500 mb-2 uppercase">Indicators:</div>
                                            <div class="flex flex-wrap gap-2">
                                                {{range .Indicators}}
                                                <span class="px-2 py-1 bg-yellow-100 text-yellow-800 text-xs rounded font-medium">{{.}}</span>
                                                {{end}}
                                            </div>
                                        </div>
                                        {{end}}
                                    </div>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
                {{end}}
                </div>
            </section>
            {{end}}

            <!-- Static Analysis -->
            {{if .StaticAnalysisIssues}}
            <section id="static-analysis" class="p-6 border-t border-gray-200">
                <button onclick="toggleSection('static-analysis-content')" class="flex items-center justify-between w-full text-left">
                    <div>
                        <h2 class="text-2xl font-bold text-gray-900">📊 Static Analysis</h2>
                        <p class="text-sm text-gray-500 mt-1">{{len .StaticAnalysisIssues}} issue{{if ne (len .StaticAnalysisIssues) 1}}s{{end}} across {{len .StaticAnalysisByCategory}} categor{{if eq (len .StaticAnalysisByCategory) 1}}y{{else}}ies{{end}}</p>
                    </div>
                    <span class="text-gray-500" id="static-analysis-toggle">▶</span>
                </button>
                <div id="static-analysis-content" class="mt-4" style="display: none;">
                    {{range .StaticAnalysisByCategory}}
                    <div class="mb-6">
                        <h3 class="text-lg font-semibold text-gray-900 mb-3">{{.Category}} ({{.Count}} issue{{if ne .Count 1}}s{{end}})</h3>
                        <div class="space-y-3">
                            {{range .Issues}}
                            <div class="bg-white rounded-lg shadow p-4 border-l-4 {{.BorderClass}}">
                                <div class="flex items-start justify-between">
                                    <div class="flex-1">
                                        <div class="flex items-center gap-2 mb-2">
                                            <span class="px-2 py-1 text-xs font-semibold rounded {{.SeverityClass}}">{{.Severity}}</span>
                                        </div>
                                        <p class="text-gray-800">{{.Description}}</p>
                                        {{if .Location}}
                                        <p class="text-sm text-gray-600 mt-2">
                                            <span class="font-medium">Location:</span> {{.Location}}
                                            {{if .Line}}
                                            <span class="ml-2">(Line {{.Line}})</span>
                                            {{end}}
                                        </p>
                                        {{end}}
                                    </div>
                                </div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </section>
            {{end}}
        </div>
    </div>

    <script>
        // Smooth scrolling for anchor links
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                const target = document.querySelector(this.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
            });
        });

        // Toggle section collapse/expand
        function toggleSection(sectionId) {
            const content = document.getElementById(sectionId);
            const toggle = document.getElementById(sectionId.replace('-content', '-toggle'));
            
            if (content.style.display === 'none') {
                content.style.display = 'block';
                toggle.textContent = '▼';
            } else {
                content.style.display = 'none';
                toggle.textContent = '▶';
            }
        }
    </script>
</body>
</html>`

	// Prepare template data
	data := f.prepareHTMLData(report, fileContents, diffInfo)
	
	// Parse and execute template
	t, err := template.New("html").Funcs(template.FuncMap{
		"multiply": func(a, b float64) float64 { return a * b },
		"formatSuggestion": func(text string) template.HTML {
			// Replace **text** with <strong>text</strong>
			// Pattern: **text** -> <strong>text</strong>
			result := text
			// Find all **text** patterns and replace them
			for {
				start := strings.Index(result, "**")
				if start == -1 {
					break
				}
				end := strings.Index(result[start+2:], "**")
				if end == -1 {
					break
				}
				end = start + 2 + end
				// Extract text between **
				boldText := result[start+2 : end]
				// Replace **text** with <strong>text</strong>
				result = result[:start] + "<strong>" + boldText + "</strong>" + result[end+2:]
			}
			return template.HTML(result)
		},
	}).Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("Error generating HTML: %v", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error executing HTML template: %v", err)
	}

	return buf.String()
}

// DiffInfo contains information about the diff being reviewed
type DiffInfo struct {
	Range      string // e.g., "main..feature-branch" or "HEAD~3..HEAD"
	FromCommit string // Starting commit SHA
	ToCommit   string // Ending commit SHA
}

// htmlData holds all data for HTML template rendering
type suggestionHTML struct {
	Heading     string
	Description string
}

type htmlData struct {
	Timestamp          string
	Status             string
	StatusClass        string
	StatusTextClass    string
	Score              int
	ScoreClass         string
	Summary            string
	TokensUsed         *tokenUsageHTML
	FileCount          int
	CriticalIssueCount int
	AIFileCount        int
	DuplicateCount     int
	HasDuplicates      bool
	HasAI              bool
	CriticalIssues     []criticalIssueHTML
	DuplicateBlocks    []duplicateBlockHTML
	AIFiles            []aiFileHTML
	StaticAnalysisIssues []staticAnalysisIssueHTML
	StaticAnalysisByCategory []staticAnalysisCategoryHTML
	Suggestions        []suggestionHTML
	DiffInfo           *DiffInfo
}

type tokenUsageHTML struct {
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
}

type criticalIssueHTML struct {
	Category      string
	Severity      string
	Description   string
	Location      string
	LocationFileID string
	BorderClass   string
	CategoryClass string
	SeverityClass string
}

type duplicateBlockHTML struct {
	OriginalFile   string
	OriginalStart  int
	OriginalEnd    int
	DuplicateFile  string
	DuplicateStart int
	DuplicateEnd   int
	Similarity     float64
	Lines          int
	OriginalCode   string
	DuplicateCode  string
}

type aiFileHTML struct {
	FilePath    string
	AIScore     *analysis.AIFileScore
}

type staticAnalysisIssueHTML struct {
	Category      string
	Severity      string
	Description   string
	Location      string
	Line          int
	BorderClass   string
	CategoryClass string
	SeverityClass string
}

type staticAnalysisCategoryHTML struct {
	Category string
	Count    int
	Issues   []staticAnalysisIssueHTML
}

func (f *Formatter) prepareHTMLData(report *ReviewReport, fileContents map[string]string, diffInfo *DiffInfo) *htmlData {
	data := &htmlData{
		Timestamp:          time.Now().Format("2006-01-02 15:04:05"),
		// Status:             report.Status,  // Commented out - scoring is subjective
		// Score:              report.Score,   // Commented out - scoring is subjective
		Summary:            report.Summary,
		Suggestions:        f.parseSuggestions(report.Suggestions),
		DiffInfo:           diffInfo,
		FileCount:          len(report.FileAnalysis),
		CriticalIssueCount: 0,
		AIFileCount:        0,
		DuplicateCount:     len(report.DuplicateBlocks),
		HasDuplicates:      len(report.DuplicateBlocks) > 0,
		HasAI:              false,
	}

	// Status styling (commented out - scoring is subjective)
	// if report.Status == "PASS" {
	// 	data.StatusClass = "bg-green-50"
	// 	data.StatusTextClass = "text-green-600"
	// } else {
	// 	data.StatusClass = "bg-red-50"
	// 	data.StatusTextClass = "text-red-600"
	// }

	// Score styling (commented out - scoring is subjective)
	// if report.Score >= 80 {
	// 	data.ScoreClass = "text-green-600"
	// } else if report.Score >= 60 {
	// 	data.ScoreClass = "text-yellow-600"
	// } else {
	// 	data.ScoreClass = "text-red-600"
	// }

	// Token usage
	if report.TokensUsed.TotalTokens > 0 {
		data.TokensUsed = &tokenUsageHTML{
			InputTokens:  report.TokensUsed.InputTokens,
			OutputTokens: report.TokensUsed.OutputTokens,
			TotalTokens:  report.TokensUsed.TotalTokens,
		}
	}

	// Critical issues (exclude static analysis and duplicate-related issues)
	for _, issue := range report.Issues {
		// Skip static analysis and duplicate-related issues (they're shown in their own sections)
		if issue.Category == "STATIC_ANALYSIS" {
			continue
		}
		if strings.Contains(strings.ToLower(issue.Description), "duplicate") {
			continue
		}
		
		data.CriticalIssueCount++
		issueHTML := criticalIssueHTML{
			Category:      issue.Category,
			Severity:      issue.Severity,
			Description:   issue.Description,
			Location:      issue.Location,
			LocationFileID: f.fileIDFromPath(issue.Location),
		}

			// Styling
			switch issue.Category {
			case "SECURITY":
				issueHTML.BorderClass = "border-red-500"
				issueHTML.CategoryClass = "bg-red-100 text-red-800"
			case "ARCHITECTURE":
				issueHTML.BorderClass = "border-orange-500"
				issueHTML.CategoryClass = "bg-orange-100 text-orange-800"
			case "PERFORMANCE":
				issueHTML.BorderClass = "border-yellow-500"
				issueHTML.CategoryClass = "bg-yellow-100 text-yellow-800"
			default:
				issueHTML.BorderClass = "border-gray-500"
				issueHTML.CategoryClass = "bg-gray-100 text-gray-800"
			}

			switch issue.Severity {
			case "CRITICAL":
				issueHTML.SeverityClass = "bg-red-100 text-red-800"
			case "WARNING":
				issueHTML.SeverityClass = "bg-yellow-100 text-yellow-800"
			default:
				issueHTML.SeverityClass = "bg-blue-100 text-blue-800"
			}

			data.CriticalIssues = append(data.CriticalIssues, issueHTML)
	}

	// Duplicate blocks
	for _, dup := range report.DuplicateBlocks {
		dupHTML := duplicateBlockHTML{
			OriginalFile:    dup.OriginalFile,
			OriginalStart:   dup.OriginalStart,
			OriginalEnd:     dup.OriginalEnd,
			DuplicateFile:   dup.DuplicateFile,
			DuplicateStart:  dup.DuplicateStart,
			DuplicateEnd:    dup.DuplicateEnd,
			Similarity:      dup.Similarity,
			Lines:           dup.Lines,
		}

		// Extract code snippets
		if originalCode := fileContents[dup.OriginalFile]; originalCode != "" {
			dupHTML.OriginalCode = f.extractLines(originalCode, dup.OriginalStart, dup.OriginalEnd)
		}
		if duplicateCode := fileContents[dup.DuplicateFile]; duplicateCode != "" {
			dupHTML.DuplicateCode = f.extractLines(duplicateCode, dup.DuplicateStart, dup.DuplicateEnd)
		}

		data.DuplicateBlocks = append(data.DuplicateBlocks, dupHTML)
	}

	// AI-generated files
	for filePath, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 {
			data.HasAI = true
			data.AIFileCount++

			aiFileHTML := aiFileHTML{
				FilePath: filePath,
				AIScore:  analysis.AIScore,
			}

			data.AIFiles = append(data.AIFiles, aiFileHTML)
		}
	}

	// Static analysis issues (full details, grouped by category)
	// Group issues by category first
	staticByCategory := make(map[string][]staticAnalysisIssueHTML)
	
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			issueHTML := staticAnalysisIssueHTML{
				Category:    issue.Subcategory,
				Severity:    issue.Severity,
				Description: issue.Description,
				Location:    issue.Location,
			}

			// Extract line number from location if available
			// Location format might be "file.go:123" or just "file.go"
			if issue.Location != "" {
				parts := strings.Split(issue.Location, ":")
				if len(parts) > 1 {
					// Try to parse the last part as line number
					if lineNum, err := fmt.Sscanf(parts[len(parts)-1], "%d", &issueHTML.Line); err == nil && lineNum > 0 {
						// Successfully parsed line number
						issueHTML.Location = strings.Join(parts[:len(parts)-1], ":")
					} else {
						// No line number in location, keep as is
						issueHTML.Location = issue.Location
					}
				}
			}

			// Category styling
			categoryNames := map[string]string{
				"complexity":      "High Complexity",
				"function_length": "Long Functions",
				"naming":          "Naming Issues",
				"duplication":     "Code Duplication",
				"unused_code":     "Unused Code",
				"style_violation": "Style Violations",
				"ai_generated":    "AI-Generated Patterns",
			}
			categoryKey := issue.Subcategory
			if categoryKey == "" {
				categoryKey = "other"
			}
			if name, ok := categoryNames[categoryKey]; ok {
				issueHTML.Category = name
			} else {
				issueHTML.Category = categoryKey
			}

			// Border and category class based on severity
			switch issue.Severity {
			case "ERROR":
				issueHTML.BorderClass = "border-red-500"
				issueHTML.SeverityClass = "bg-red-100 text-red-800"
			case "WARNING":
				issueHTML.BorderClass = "border-yellow-500"
				issueHTML.SeverityClass = "bg-yellow-100 text-yellow-800"
			default:
				issueHTML.BorderClass = "border-blue-500"
				issueHTML.SeverityClass = "bg-blue-100 text-blue-800"
			}

			issueHTML.CategoryClass = "bg-gray-100 text-gray-800"

			// Group by category
			if staticByCategory[issueHTML.Category] == nil {
				staticByCategory[issueHTML.Category] = make([]staticAnalysisIssueHTML, 0)
			}
			staticByCategory[issueHTML.Category] = append(staticByCategory[issueHTML.Category], issueHTML)
			data.StaticAnalysisIssues = append(data.StaticAnalysisIssues, issueHTML)
		}
	}

	// Convert grouped map to slice for template
	for category, issues := range staticByCategory {
		data.StaticAnalysisByCategory = append(data.StaticAnalysisByCategory, staticAnalysisCategoryHTML{
			Category: category,
			Count:    len(issues),
			Issues:   issues,
		})
	}

	return data
}

// Helper functions
func (f *Formatter) fileIDFromPath(path string) string {
	// Create a simple file ID from path
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ReplaceAll(path, "\\", "-")
	path = strings.ReplaceAll(path, ".", "-")
	return "file-" + path
}

func (f *Formatter) detectLanguage(filePath string) string {
	if strings.HasSuffix(filePath, ".go") {
		return "go"
	} else if strings.HasSuffix(filePath, ".js") {
		return "javascript"
	} else if strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") {
		return "typescript"
	} else if strings.HasSuffix(filePath, ".py") {
		return "python"
	} else if strings.HasSuffix(filePath, ".java") {
		return "java"
	} else if strings.HasSuffix(filePath, ".rs") {
		return "rust"
	} else if strings.HasSuffix(filePath, ".cpp") || strings.HasSuffix(filePath, ".cc") || strings.HasSuffix(filePath, ".cxx") {
		return "cpp"
	} else if strings.HasSuffix(filePath, ".c") {
		return "c"
	}
	return "text"
}

func (f *Formatter) extractLines(content string, startLine, endLine int) string {
	lines := strings.Split(content, "\n")
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return ""
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

// parseSuggestions parses suggestion strings to extract headings and descriptions
func (f *Formatter) parseSuggestions(suggestions []string) []suggestionHTML {
	parsed := make([]suggestionHTML, 0)
	
	for _, sug := range suggestions {
		sug = strings.TrimSpace(sug)
		if sug == "" {
			continue
		}
		
		// Remove leading "- " if present (from markdown list)
		sug = strings.TrimPrefix(sug, "- ")
		sug = strings.TrimSpace(sug)
		
		// Check for pattern: **Heading**: Description
		if strings.Contains(sug, "**:") {
			parts := strings.SplitN(sug, "**:", 2)
			if len(parts) == 2 {
				heading := strings.TrimSpace(strings.TrimPrefix(parts[0], "**"))
				description := strings.TrimSpace(parts[1])
				if heading != "" && description != "" {
					parsed = append(parsed, suggestionHTML{
						Heading:     heading,
						Description: description,
					})
					continue
				}
			}
		}
		
		// Check for pattern: **Heading**: Description (without closing **)
		if strings.HasPrefix(sug, "**") && strings.Contains(sug, ":") {
			// Find the first colon after **
			colonIdx := strings.Index(sug[2:], ":")
			if colonIdx > 0 {
				heading := strings.TrimSpace(sug[2 : 2+colonIdx])
				description := strings.TrimSpace(sug[2+colonIdx+1:])
				if heading != "" && description != "" {
					parsed = append(parsed, suggestionHTML{
						Heading:     heading,
						Description: description,
					})
					continue
				}
			}
		}
		
		// Check for pattern: Heading: Description (no asterisks)
		// Only treat as heading if it's short, doesn't contain a period, and has description
		if strings.Contains(sug, ":") {
			parts := strings.SplitN(sug, ":", 2)
			if len(parts) == 2 {
				heading := strings.TrimSpace(parts[0])
				description := strings.TrimSpace(parts[1])
				// Check if it looks like a heading (short, no period, has description)
				if len(heading) > 0 && len(heading) < 50 && !strings.Contains(heading, ".") && description != "" {
					parsed = append(parsed, suggestionHTML{
						Heading:     heading,
						Description: description,
					})
					continue
				}
			}
		}
		
		// If no heading pattern found, treat entire string as description
		// Skip if it looks like just a heading without description (ends with colon, short)
		if strings.HasSuffix(sug, ":") && len(sug) < 50 {
			// This might be a category header, skip it or use as heading only
			continue
		}
		
		parsed = append(parsed, suggestionHTML{
			Description: sug,
		})
	}
	
	return parsed
}

