package analysis

import (
	"fmt"
	"regexp"
	"strings"
)

// DBQueryInfo represents information about a database query
type DBQueryInfo struct {
	File            string        `json:"file"`
	Line            int           `json:"line"`
	Function        string        `json:"function"`
	QueryType       string        `json:"query_type"` // "ORM" or "RAW"
	Query           string        `json:"query"`      // Extracted query or ORM method chain
	EfficiencyScore float64       `json:"efficiency_score"` // 0.0-1.0, higher = more efficient
	Issues          []DBQueryIssue `json:"issues"`
	Recommendation  string        `json:"recommendation"` // e.g., "Consider raw query for better performance"
}

// DBQueryIssue represents a specific issue with a query
type DBQueryIssue struct {
	Type        string `json:"type"` // "N+1", "MISSING_INDEX", "INEFFICIENT_JOIN", "RAW_PREFERRED", "SELECT_ALL", "NO_LIMIT", etc.
	Severity    string `json:"severity"` // "CRITICAL", "WARNING", "INFO"
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// AnalyzeDBQueries analyzes database queries in file analysis
func AnalyzeDBQueries(fileAnalysis *FileAnalysis, filePath string) []DBQueryInfo {
	queries := make([]DBQueryInfo, 0)

	// Check if file has database/ORM imports
	hasDBImports := false
	for _, imp := range fileAnalysis.Imports {
		importPath := strings.ToLower(imp.Path)
		if isDatabaseImport(importPath) {
			hasDBImports = true
			break
		}
	}

	if !hasDBImports {
		return queries
	}

	// Analyze each function for database queries
	for _, fn := range fileAnalysis.Functions {
		if fn.Body == "" {
			continue
		}

		// Detect queries in function body
		detectedQueries := detectQueriesInFunction(fn, filePath, fileAnalysis.Language)
		queries = append(queries, detectedQueries...)
	}

	return queries
}

// isDatabaseImport checks if an import is a database/ORM library
func isDatabaseImport(importPath string) bool {
	// Python
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

	// JavaScript/TypeScript
	if strings.Contains(importPath, "sequelize") ||
		strings.Contains(importPath, "prisma") ||
		strings.Contains(importPath, "typeorm") ||
		strings.Contains(importPath, "mongoose") ||
		strings.Contains(importPath, "knex") ||
		strings.Contains(importPath, "bookshelf") {
		return true
	}

	// Go
	if strings.Contains(importPath, "database/sql") ||
		strings.Contains(importPath, "gorm.io/gorm") ||
		strings.Contains(importPath, "github.com/jmoiron/sqlx") ||
		strings.Contains(importPath, "gorm.io/driver") {
		return true
	}

	// Java
	if strings.Contains(importPath, "javax.persistence") ||
		strings.Contains(importPath, "org.hibernate") ||
		strings.Contains(importPath, "org.springframework.data") ||
		strings.Contains(importPath, "jakarta.persistence") {
		return true
	}

	return false
}

// detectQueriesInFunction detects and analyzes queries in a function
func detectQueriesInFunction(fn FunctionInfo, filePath string, language string) []DBQueryInfo {
	queries := make([]DBQueryInfo, 0)
	body := fn.Body

	// Detect ORM queries
	ormPatterns := []struct {
		pattern string
		queryType string
	}{
		{`\.query\(`, "ORM"},
		{`\.filter\(`, "ORM"},
		{`\.get\(`, "ORM"},
		{`\.findOne\(`, "ORM"},
		{`\.findAll\(`, "ORM"},
		{`\.find\(`, "ORM"},
		{`\.findById\(`, "ORM"},
		{`\.where\(`, "ORM"},
		{`\.select\(`, "ORM"},
		{`\.First\(`, "ORM"},
		{`\.Find\(`, "ORM"},
		{`\.Query\(`, "ORM"},
		{`\.createQuery\(`, "ORM"},
	}

	// Detect raw SQL queries
	rawSQLPatterns := []string{
		`SELECT\s+.*?\s+FROM`,
		`INSERT\s+INTO`,
		`UPDATE\s+.*?\s+SET`,
		`DELETE\s+FROM`,
		`EXEC\s+`,
		`EXECUTE\s+`,
		`db\.query\(`,
		`db\.execute\(`,
		`connection\.query\(`,
		`\.raw\(`,
		`\.execute\(`,
	}

	// Check for ORM queries
	for _, pattern := range ormPatterns {
		re := regexp.MustCompile("(?i)" + pattern.pattern)
		if re.MatchString(body) {
			query := DBQueryInfo{
				File:            filePath,
				Line:            fn.StartLine,
				Function:        fn.Name,
				QueryType:       pattern.queryType,
				Query:           extractQuerySnippet(body, pattern.pattern),
				EfficiencyScore: 1.0, // Start with perfect score
				Issues:          make([]DBQueryIssue, 0),
			}

			// Analyze query efficiency
			analyzeQueryEfficiency(&query, body, language)
			queries = append(queries, query)
			break // Only create one query info per function for now
		}
	}

	// Check for raw SQL queries
	for _, pattern := range rawSQLPatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if re.MatchString(body) {
			query := DBQueryInfo{
				File:            filePath,
				Line:            fn.StartLine,
				Function:        fn.Name,
				QueryType:       "RAW",
				Query:           extractQuerySnippet(body, pattern),
				EfficiencyScore: 1.0,
				Issues:          make([]DBQueryIssue, 0),
			}

			// Analyze raw query efficiency
			analyzeRawQueryEfficiency(&query, body, language)
			queries = append(queries, query)
			break
		}
	}

	return queries
}

// analyzeQueryEfficiency analyzes ORM query efficiency
func analyzeQueryEfficiency(query *DBQueryInfo, body string, language string) {
	issues := make([]DBQueryIssue, 0)
	bodyLower := strings.ToLower(body)

	// Check for N+1 query problem (query inside loop)
	if strings.Contains(bodyLower, "for ") || strings.Contains(bodyLower, "while ") {
		// Simple heuristic: if there's a loop and query methods inside, likely N+1
		queryMethods := []string{"query", "filter", "get", "findone", "findall", "find", "where", "first", "find"}
		for _, method := range queryMethods {
			if strings.Contains(bodyLower, method+"(") {
				issues = append(issues, DBQueryIssue{
					Type:        "N+1",
					Severity:    "WARNING",
					Description: "Query executed inside a loop - potential N+1 query problem",
					Suggestion:  "Consider using eager loading, batch queries, or join operations to fetch related data in a single query",
				})
				query.EfficiencyScore -= 0.3
				break
			}
		}
	}

	// Check for SELECT * (select all fields)
	if strings.Contains(bodyLower, "select *") || strings.Contains(bodyLower, "select_all") {
		issues = append(issues, DBQueryIssue{
			Type:        "SELECT_ALL",
			Severity:    "INFO",
			Description: "Query selects all fields instead of specific columns",
			Suggestion:  "Specify only the columns you need to reduce data transfer and improve performance",
		})
		query.EfficiencyScore -= 0.1
	}

	// Check for missing LIMIT on potentially large queries
	if !strings.Contains(bodyLower, "limit") && !strings.Contains(bodyLower, ".limit(") && !strings.Contains(bodyLower, ".take(") {
		// If it's a SELECT or find operation without limit, flag it
		if strings.Contains(bodyLower, "select") || strings.Contains(bodyLower, "find") || strings.Contains(bodyLower, "query") {
			issues = append(issues, DBQueryIssue{
				Type:        "NO_LIMIT",
				Severity:    "WARNING",
				Description: "Query may return large result set without pagination",
				Suggestion:  "Add LIMIT/OFFSET or pagination to prevent loading excessive data",
			})
			query.EfficiencyScore -= 0.1
		}
	}

	// Check if raw query would be preferred (complex operations)
	if isComplexQuery(body) {
		issues = append(issues, DBQueryIssue{
			Type:        "RAW_PREFERRED",
			Severity:    "INFO",
			Description: "Complex query that might be more efficient as raw SQL",
			Suggestion:  "Consider using raw SQL for better performance and control over query execution",
		})
		query.EfficiencyScore -= 0.2
	}

	// Clamp efficiency score between 0.0 and 1.0
	if query.EfficiencyScore < 0.0 {
		query.EfficiencyScore = 0.0
	}
	if query.EfficiencyScore > 1.0 {
		query.EfficiencyScore = 1.0
	}

	query.Issues = issues

	// Set recommendation based on issues
	if len(issues) > 0 {
		rawPreferred := false
		for _, issue := range issues {
			if issue.Type == "RAW_PREFERRED" {
				rawPreferred = true
				break
			}
		}
		if rawPreferred {
			query.Recommendation = "Consider using raw SQL query for better performance and control"
		} else if query.EfficiencyScore < 0.5 {
			query.Recommendation = "Query has significant efficiency issues - review and optimize"
		} else if query.EfficiencyScore < 0.8 {
			query.Recommendation = "Query could be optimized for better performance"
		}
	}
}

// analyzeRawQueryEfficiency analyzes raw SQL query efficiency
func analyzeRawQueryEfficiency(query *DBQueryInfo, body string, language string) {
	bodyLower := strings.ToLower(body)
	issues := make([]DBQueryIssue, 0)

	// Check for SQL injection risk (string concatenation)
	if strings.Contains(body, "+") || strings.Contains(body, "fmt.Sprintf") || strings.Contains(body, "format") {
		// Check if it's likely string concatenation with variables
		if strings.Contains(body, "$") || strings.Contains(body, "%") || strings.Contains(body, "{") {
			issues = append(issues, DBQueryIssue{
				Type:        "SQL_INJECTION_RISK",
				Severity:    "CRITICAL",
				Description: "Raw SQL query uses string concatenation - potential SQL injection vulnerability",
				Suggestion:  "Use parameterized queries or prepared statements to prevent SQL injection",
			})
			query.EfficiencyScore -= 0.5
		}
	}

	// Check for SELECT *
	if strings.Contains(bodyLower, "select *") {
		issues = append(issues, DBQueryIssue{
			Type:        "SELECT_ALL",
			Severity:    "INFO",
			Description: "Query selects all fields instead of specific columns",
			Suggestion:  "Specify only the columns you need to reduce data transfer and improve performance",
		})
		query.EfficiencyScore -= 0.1
	}

	// Check for missing LIMIT
	if !strings.Contains(bodyLower, "limit") {
		if strings.Contains(bodyLower, "select") {
			issues = append(issues, DBQueryIssue{
				Type:        "NO_LIMIT",
				Severity:    "WARNING",
				Description: "Query may return large result set without pagination",
				Suggestion:  "Add LIMIT clause to prevent loading excessive data",
			})
			query.EfficiencyScore -= 0.1
		}
	}

	// Check for inefficient joins (cartesian product risk)
	if strings.Contains(bodyLower, "join") {
		joinCount := strings.Count(bodyLower, "join")
		if joinCount >= 3 {
			issues = append(issues, DBQueryIssue{
				Type:        "INEFFICIENT_JOIN",
				Severity:    "WARNING",
				Description: fmt.Sprintf("Query has %d joins - verify join conditions to avoid cartesian products", joinCount),
				Suggestion:  "Ensure all joins have proper ON conditions and consider query optimization",
			})
			query.EfficiencyScore -= 0.2
		}
	}

	// Clamp efficiency score
	if query.EfficiencyScore < 0.0 {
		query.EfficiencyScore = 0.0
	}
	if query.EfficiencyScore > 1.0 {
		query.EfficiencyScore = 1.0
	}

	query.Issues = issues
}

// isComplexQuery checks if a query is complex enough to warrant raw SQL
func isComplexQuery(body string) bool {
	bodyLower := strings.ToLower(body)
	
	// Check for complex SQL operations
	complexPatterns := []string{
		"group by",
		"having",
		"window",
		"over (",
		"partition by",
		"union",
		"subquery",
		"exists (",
		"in (select",
	}

	complexCount := 0
	for _, pattern := range complexPatterns {
		if strings.Contains(bodyLower, pattern) {
			complexCount++
		}
	}

	// Check for multiple joins
	joinCount := strings.Count(bodyLower, "join")
	if joinCount >= 3 {
		complexCount++
	}

	return complexCount >= 2
}

// extractQuerySnippet extracts a snippet of the query from function body
func extractQuerySnippet(body string, pattern string) string {
	// Try to extract a reasonable snippet around the query
	lines := strings.Split(body, "\n")
	
	// Find line with the pattern
	for i, line := range lines {
		re := regexp.MustCompile("(?i)" + pattern)
		if re.MatchString(line) {
			// Extract this line and next few lines (up to 5 lines)
			start := max(0, i)
			end := min(len(lines), i+5)
			snippet := strings.Join(lines[start:end], "\n")
			
			// Truncate if too long
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			return snippet
		}
	}
	
	// Fallback: return first 200 chars
	if len(body) > 200 {
		return body[:200] + "..."
	}
	return body
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

