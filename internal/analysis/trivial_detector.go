package analysis

import (
	"regexp"
	"strings"
)

// TrivialPatternDetector identifies trivial/boilerplate functions
type TrivialPatternDetector struct {
	language string
}

// NewTrivialPatternDetector creates a detector
func NewTrivialPatternDetector(language string) *TrivialPatternDetector {
	return &TrivialPatternDetector{language: language}
}

// IsTrivial checks if a function is trivial boilerplate
func (d *TrivialPatternDetector) IsTrivial(fn FunctionInfo) bool {
	// 1. Check if it's a getter
	if d.isGetter(fn) {
		return true
	}

	// 2. Check if it's a setter
	if d.isSetter(fn) {
		return true
	}

	// 3. Check if it's simple CRUD
	if d.isSimpleCRUD(fn) {
		return true
	}

	// 4. Check if it's a constructor/initializer
	if d.isConstructor(fn) {
		return true
	}

	return false
}

// isGetter detects getter methods
func (d *TrivialPatternDetector) isGetter(fn FunctionInfo) bool {
	// Common getter patterns:
	// - getName(), getAge(), getId()
	// - name(), age(), id() (Go style)
	// - get_name(), get_age() (Python)

	name := strings.ToLower(fn.Name)

	// Starts with "get"
	if strings.HasPrefix(name, "get") && len(fn.Parameters) == 0 {
		// Check body is simple return
		body := strings.TrimSpace(fn.Body)
		lines := strings.Split(body, "\n")

		// Single line return
		if len(lines) <= 3 {
			return strings.Contains(body, "return")
		}
	}

	// Python style get_*
	if strings.HasPrefix(name, "get_") && len(fn.Parameters) == 0 {
		return true
	}

	return false
}

// isSetter detects setter methods
func (d *TrivialPatternDetector) isSetter(fn FunctionInfo) bool {
	name := strings.ToLower(fn.Name)

	// Starts with "set"
	if strings.HasPrefix(name, "set") && len(fn.Parameters) == 1 {
		body := strings.TrimSpace(fn.Body)
		lines := strings.Split(body, "\n")

		// Simple assignment
		if len(lines) <= 3 {
			return strings.Contains(body, "=") || strings.Contains(body, "this.")
		}
	}

	// Python style set_*
	if strings.HasPrefix(name, "set_") && len(fn.Parameters) == 1 {
		return true
	}

	return false
}

// isSimpleCRUD detects basic CRUD operations
func (d *TrivialPatternDetector) isSimpleCRUD(fn FunctionInfo) bool {
	body := fn.Body
	name := strings.ToLower(fn.Name)

	// Check for single repository/database call patterns
	crudPatterns := []string{
		// Java/Spring
		`repository\.save\(`,
		`repository\.findById\(`,
		`repository\.delete\(`,
		`repository\.findAll\(`,

		// Direct SQL
		`SELECT .* FROM .* WHERE`,
		`INSERT INTO`,
		`UPDATE .* SET`,
		`DELETE FROM`,
	}

	// If name suggests CRUD
	crudNames := []string{"save", "find", "get", "delete", "update", "create", "insert"}
	isCRUDName := false
	for _, crud := range crudNames {
		if strings.Contains(name, crud) {
			isCRUDName = true
			break
		}
	}

	if !isCRUDName {
		return false
	}

	// Check if body is just a simple call
	for _, pattern := range crudPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(body) {
			// Count non-comment lines
			lines := strings.Split(body, "\n")
			codeLines := 0
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
					codeLines++
				}
			}

			// If very few lines, it's trivial
			if codeLines <= 3 {
				return true
			}
		}
	}

	return false
}

// isConstructor detects constructors/initializers
func (d *TrivialPatternDetector) isConstructor(fn FunctionInfo) bool {
	name := strings.ToLower(fn.Name)

	// Common constructor names
	constructorNames := []string{"constructor", "__init__", "init", "initialize"}
	for _, cons := range constructorNames {
		if name == cons {
			return true
		}
	}

	return false
}




