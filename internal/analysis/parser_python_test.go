package analysis

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// isPythonAvailable checks if Python is available on the system
func isPythonAvailable() bool {
	_, err := exec.LookPath("python3")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("python")
	return err == nil
}

func TestPythonParser_BasicFunctions(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
def calculate_sum(a, b):
    """Add two numbers"""
    if a > 0 and b > 0:
        return a + b
    return 0

def process_data(items):
    for item in items:
        if item > 10:
            print(item)
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(analysis.Functions))
	}

	// Check first function
	if len(analysis.Functions) > 0 {
		fn := analysis.Functions[0]
		if fn.Name != "calculate_sum" {
			t.Errorf("Expected function name 'calculate_sum', got '%s'", fn.Name)
		}
		if len(fn.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(fn.Parameters))
		}
		if fn.Complexity < 2 {
			t.Errorf("Expected complexity >= 2, got %d", fn.Complexity)
		}
	}

	// Check metrics
	if analysis.Metrics.FunctionCount != 2 {
		t.Errorf("Expected FunctionCount 2, got %d", analysis.Metrics.FunctionCount)
	}
}

func TestPythonParser_Classes(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
class DataProcessor:
    """Process data efficiently"""
    
    def __init__(self, name):
        self.name = name
        self.count = 0
    
    def process(self, data):
        """Process the data"""
        for item in data:
            self.count += 1
            if item > 0:
                print(item)
    
    def _internal_helper(self):
        return self.count

class SimpleClass:
    pass
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Classes) != 2 {
		t.Errorf("Expected 2 classes, got %d", len(analysis.Classes))
	}

	// Check first class
	if len(analysis.Classes) > 0 {
		class := analysis.Classes[0]
		if class.Name != "DataProcessor" {
			t.Errorf("Expected class name 'DataProcessor', got '%s'", class.Name)
		}
		if len(class.Methods) != 3 {
			t.Errorf("Expected 3 methods, got %d", len(class.Methods))
		}
		if !class.IsExported {
			t.Error("Expected class to be exported")
		}
	}

	// Check metrics
	if analysis.Metrics.ClassCount != 2 {
		t.Errorf("Expected ClassCount 2, got %d", analysis.Metrics.ClassCount)
	}
}

func TestPythonParser_Imports(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
import os
import sys
from pathlib import Path
from typing import List, Dict
import json as j

def process():
    pass
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Imports) < 3 {
		t.Errorf("Expected at least 3 imports, got %d", len(analysis.Imports))
	}

	// Check for json alias
	hasJsonAlias := false
	for _, imp := range analysis.Imports {
		if strings.Contains(imp.Path, "json") && imp.Alias == "j" {
			hasJsonAlias = true
			break
		}
	}
	if !hasJsonAlias {
		t.Error("Expected to find json import with alias 'j'")
	}

	// Check metrics
	if analysis.Metrics.ImportCount < 3 {
		t.Errorf("Expected ImportCount >= 3, got %d", analysis.Metrics.ImportCount)
	}
}

func TestPythonParser_ComplexityCalculation(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
def complex_function(x, y):
    if x > 0:
        if y > 0:
            for i in range(10):
                while i < 5:
                    if i % 2 == 0 or i % 3 == 0:
                        try:
                            return i
                        except:
                            pass
    return 0
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(analysis.Functions))
	}

	fn := analysis.Functions[0]
	// Should have high complexity due to nested if, for, while, or, try
	if fn.Complexity < 8 {
		t.Errorf("Expected complexity >= 8 for complex function, got %d", fn.Complexity)
	}

	// Complexity of 9 is below the threshold of 10, so no issue should be generated
	// This test just verifies the complexity is calculated correctly
}

func TestPythonParser_AsyncFunctions(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
async def fetch_data(url):
    """Fetch data asynchronously"""
    async with aiohttp.ClientSession() as session:
        async for response in session.get(url):
            return await response.text()
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 1 {
		t.Fatalf("Expected 1 async function, got %d", len(analysis.Functions))
	}

	fn := analysis.Functions[0]
	if fn.Name != "fetch_data" {
		t.Errorf("Expected function name 'fetch_data', got '%s'", fn.Name)
	}
}

func TestPythonParser_SyntaxError(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
def broken syntax here:
    this is not valid python
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	_, err = parser.ParseFile(tmp.Name())
	
	// Should return error (allowing fallback to regex parser)
	if err == nil {
		t.Error("Expected error for syntax error")
	}
}

func TestPythonParser_ExportedVsPrivate(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
def public_function():
    pass

def _private_function():
    pass

class PublicClass:
    pass

class _PrivateClass:
    pass
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check functions
	if len(analysis.Functions) != 2 {
		t.Fatalf("Expected 2 functions, got %d", len(analysis.Functions))
	}

	publicFunc := analysis.Functions[0]
	privateFunc := analysis.Functions[1]

	if !publicFunc.IsExported {
		t.Error("Expected public_function to be exported")
	}
	if privateFunc.IsExported {
		t.Error("Expected _private_function to NOT be exported")
	}

	// Check classes
	if len(analysis.Classes) != 2 {
		t.Fatalf("Expected 2 classes, got %d", len(analysis.Classes))
	}

	publicClass := analysis.Classes[0]
	privateClass := analysis.Classes[1]

	if !publicClass.IsExported {
		t.Error("Expected PublicClass to be exported")
	}
	if privateClass.IsExported {
		t.Error("Expected _PrivateClass to NOT be exported")
	}
}

func TestPythonParser_LongFunction(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	// Create a function with > 50 lines
	lines := []string{"def very_long_function():"}
	for i := 0; i < 55; i++ {
		lines = append(lines, "    print('line')")
	}
	content := strings.Join(lines, "\n")

	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(analysis.Functions))
	}

	// Should trigger function length warning
	foundLengthIssue := false
	for _, issue := range analysis.Issues {
		if issue.Type == IssueTypeFunctionLength {
			foundLengthIssue = true
			break
		}
	}
	if !foundLengthIssue {
		t.Error("Expected function length issue to be generated")
	}
}

func TestPythonParser_Decorators(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("Python not available, skipping AST test")
	}

	content := `
@property
def get_value(self):
    return self._value

@staticmethod
@cache
def cached_function():
    return "result"
`
	tmp, err := os.CreateTemp("", "*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	
	tmp.WriteString(content)
	tmp.Close()

	parser := NewPythonParser()
	defer parser.Cleanup()
	
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(analysis.Functions))
	}

	// Verify decorators are preserved in body
	if len(analysis.Functions) > 0 {
		body := analysis.Functions[0].Body
		if !strings.Contains(body, "@property") {
			t.Error("Expected decorator to be preserved in function body")
		}
	}
}
