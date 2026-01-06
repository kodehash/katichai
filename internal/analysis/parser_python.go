package analysis

import (
	_ "embed"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

//go:embed python_ast_helper.py
var pythonHelperScript string

// PythonParser parses Python source files using Python's AST module
type PythonParser struct {
	scriptPath string // Cached temp file path
}

// pythonASTOutput matches the JSON output from python_ast_helper.py
type pythonASTOutput struct {
	Functions []FunctionInfo `json:"functions"`
	Classes   []ClassInfo    `json:"classes"`
	Imports   []ImportInfo   `json:"imports"`
}

// NewPythonParser creates a new Python parser
func NewPythonParser() *PythonParser {
	return &PythonParser{}
}

// ParseFile parses a Python source file using AST
func (p *PythonParser) ParseFile(filePath string) (*FileAnalysis, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if Python is available
	pythonCmd, err := findPythonCommand()
	if err != nil {
		return nil, fmt.Errorf("python not found: %w", err)
	}

	// Get or create temp script file
	scriptPath, err := p.getOrCreateScriptFile()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp script: %w", err)
	}

	// Call Python helper script
	jsonData, err := p.callPythonHelper(pythonCmd, scriptPath, filePath)
	if err != nil {
		// Provide more detailed error information
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python helper failed (exit code %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("python helper failed: %w", err)
	}
	
	// Validate that we got JSON output
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("python helper returned empty output")
	}

	// Parse JSON output
	fileAnalysis, err := p.parseJSONOutput(jsonData, filePath, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return fileAnalysis, nil
}

// findPythonCommand finds the available Python command (python3 or python)
func findPythonCommand() (string, error) {
	// Try python3 first (preferred)
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3", nil
	}
	
	// Fall back to python
	if _, err := exec.LookPath("python"); err == nil {
		return "python", nil
	}
	
	return "", fmt.Errorf("python not found in PATH")
}

// getOrCreateScriptFile creates or returns cached temp script file
func (p *PythonParser) getOrCreateScriptFile() (string, error) {
	// Return cached path if already created
	if p.scriptPath != "" {
		// Verify it still exists
		if _, err := os.Stat(p.scriptPath); err == nil {
			return p.scriptPath, nil
		}
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "katich_python_ast_*.py")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write embedded script
	if _, err := tmpFile.WriteString(pythonHelperScript); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	p.scriptPath = tmpFile.Name()
	return p.scriptPath, nil
}

// callPythonHelper executes the Python helper script
func (p *PythonParser) callPythonHelper(pythonCmd, scriptPath, filePath string) ([]byte, error) {
	cmd := exec.Command(pythonCmd, scriptPath, filePath)
	
	output, err := cmd.Output()
	if err != nil {
		// Check if it's an exit error with stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python script failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	return output, nil
}

// parseJSONOutput parses the JSON output from Python helper
func (p *PythonParser) parseJSONOutput(jsonData []byte, filePath string, content []byte) (*FileAnalysis, error) {
	var astOutput pythonASTOutput
	if err := json.Unmarshal(jsonData, &astOutput); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(content)
	hashStr := fmt.Sprintf("%x", hash)

	// Create FileAnalysis
	analysis := &FileAnalysis{
		FilePath:  filePath,
		Language:  "Python",
		Hash:      hashStr,
		Functions: astOutput.Functions,
		Classes:   astOutput.Classes,
		Imports:   astOutput.Imports,
		Issues:    make([]Issue, 0),
	}

	// Calculate metrics
	analysis.Metrics = p.calculateMetrics(string(content), analysis)

	// Generate issues
	p.generateIssues(analysis)

	return analysis, nil
}

// calculateMetrics calculates overall file metrics
func (p *PythonParser) calculateMetrics(content string, analysis *FileAnalysis) CodeMetrics {
	metrics := CalculateBasicMetrics(content)
	
	metrics.FunctionCount = len(analysis.Functions)
	metrics.ClassCount = len(analysis.Classes)
	metrics.ImportCount = len(analysis.Imports)

	// Calculate max and average function length
	if len(analysis.Functions) > 0 {
		totalLOC := 0
		for _, fn := range analysis.Functions {
			if fn.LOC > metrics.MaxFunctionLength {
				metrics.MaxFunctionLength = fn.LOC
			}
			totalLOC += fn.LOC
		}
		metrics.AvgFunctionLength = float64(totalLOC) / float64(len(analysis.Functions))
	}

	// Calculate total complexity
	totalComplexity := 0
	for _, fn := range analysis.Functions {
		totalComplexity += fn.Complexity
	}
	
	// Add complexity from class methods
	for _, class := range analysis.Classes {
		for _, method := range class.Methods {
			totalComplexity += method.Complexity
		}
	}
	
	metrics.CyclomaticComplexity = totalComplexity

	return metrics
}

// generateIssues generates issues based on analysis (same thresholds as regex parser)
func (p *PythonParser) generateIssues(analysis *FileAnalysis) {
	// Check function issues
	for _, fn := range analysis.Functions {
		if fn.Complexity > 10 {
			analysis.Issues = append(analysis.Issues, Issue{
				Type:       IssueTypeComplexity,
				Severity:   SeverityWarning,
				Line:       fn.StartLine,
				Message:    fmt.Sprintf("Function '%s' has high complexity: %d", fn.Name, fn.Complexity),
				Suggestion: "Consider breaking down this function into smaller functions",
			})
		}
		if fn.LOC > 50 {
			analysis.Issues = append(analysis.Issues, Issue{
				Type:       IssueTypeFunctionLength,
				Severity:   SeverityWarning,
				Line:       fn.StartLine,
				Message:    fmt.Sprintf("Function '%s' is too long: %d lines", fn.Name, fn.LOC),
				Suggestion: "Consider refactoring into smaller functions",
			})
		}
	}

	// Check method issues
	for _, class := range analysis.Classes {
		for _, method := range class.Methods {
			if method.Complexity > 10 {
				analysis.Issues = append(analysis.Issues, Issue{
					Type:       IssueTypeComplexity,
					Severity:   SeverityWarning,
					Line:       method.StartLine,
					Message:    fmt.Sprintf("Method '%s.%s' has high complexity: %d", class.Name, method.Name, method.Complexity),
					Suggestion: "Consider breaking down this method into smaller methods",
				})
			}
			if method.LOC > 50 {
				analysis.Issues = append(analysis.Issues, Issue{
					Type:       IssueTypeFunctionLength,
					Severity:   SeverityWarning,
					Line:       method.StartLine,
					Message:    fmt.Sprintf("Method '%s.%s' is too long: %d lines", class.Name, method.Name, method.LOC),
					Suggestion: "Consider refactoring into smaller methods",
				})
			}
		}
	}
}

// Cleanup removes the temporary script file
func (p *PythonParser) Cleanup() {
	if p.scriptPath != "" {
		os.Remove(p.scriptPath)
		p.scriptPath = ""
	}
}
