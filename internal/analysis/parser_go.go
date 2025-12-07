package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// GoParser parses Go source files
type GoParser struct{}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// ParseFile parses a Go source file
func (p *GoParser) ParseFile(filePath string) (*FileAnalysis, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	analysis := &FileAnalysis{
		FilePath:  filePath,
		Language:  "Go",
		Functions: make([]FunctionInfo, 0),
		Classes:   make([]ClassInfo, 0),
		Imports:   make([]ImportInfo, 0),
		Issues:    make([]Issue, 0),
	}

	// Extract imports
	for _, imp := range file.Imports {
		importInfo := ImportInfo{
			Path: imp.Path.Value,
		}
		if imp.Name != nil {
			importInfo.Alias = imp.Name.Name
		}
		analysis.Imports = append(analysis.Imports, importInfo)
	}

	// Walk AST
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			funcInfo := p.extractFunction(node, fset)
			analysis.Functions = append(analysis.Functions, funcInfo)
			
			// Check for issues
			if funcInfo.Complexity > 10 {
				analysis.Issues = append(analysis.Issues, Issue{
					Type:     IssueTypeComplexity,
					Severity: SeverityWarning,
					Line:     funcInfo.StartLine,
					Message:  fmt.Sprintf("Function '%s' has high complexity: %d", funcInfo.Name, funcInfo.Complexity),
					Suggestion: "Consider breaking down this function into smaller functions",
				})
			}
			
			if funcInfo.LOC > 50 {
				analysis.Issues = append(analysis.Issues, Issue{
					Type:     IssueTypeFunctionLength,
					Severity: SeverityWarning,
					Line:     funcInfo.StartLine,
					Message:  fmt.Sprintf("Function '%s' is too long: %d lines", funcInfo.Name, funcInfo.LOC),
					Suggestion: "Consider refactoring into smaller functions",
				})
			}

		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				classInfo := p.extractStruct(node, structType, fset)
				analysis.Classes = append(analysis.Classes, classInfo)
			}
		}
		return true
	})

	// Calculate metrics
	analysis.Metrics = p.calculateMetrics(string(content), analysis)

	return analysis, nil
}

// extractFunction extracts function information
func (p *GoParser) extractFunction(funcDecl *ast.FuncDecl, fset *token.FileSet) FunctionInfo {
	startPos := fset.Position(funcDecl.Pos())
	endPos := fset.Position(funcDecl.End())

	funcInfo := FunctionInfo{
		Name:       funcDecl.Name.Name,
		StartLine:  startPos.Line,
		EndLine:    endPos.Line,
		LOC:        endPos.Line - startPos.Line + 1,
		Parameters: make([]string, 0),
		IsExported: funcDecl.Name.IsExported(),
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			for _, name := range param.Names {
				funcInfo.Parameters = append(funcInfo.Parameters, name.Name)
			}
		}
	}

	// Calculate complexity
	funcInfo.Complexity = p.calculateComplexity(funcDecl)

	// Extract comments
	if funcDecl.Doc != nil {
		funcInfo.Comments = funcDecl.Doc.Text()
	}

	return funcInfo
}

// extractStruct extracts struct information
func (p *GoParser) extractStruct(typeSpec *ast.TypeSpec, structType *ast.StructType, fset *token.FileSet) ClassInfo {
	startPos := fset.Position(structType.Pos())
	endPos := fset.Position(structType.End())

	classInfo := ClassInfo{
		Name:       typeSpec.Name.Name,
		StartLine:  startPos.Line,
		EndLine:    endPos.Line,
		Methods:    make([]FunctionInfo, 0),
		Fields:     make([]FieldInfo, 0),
		IsExported: typeSpec.Name.IsExported(),
	}

	// Extract fields
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			for _, name := range field.Names {
				fieldInfo := FieldInfo{
					Name: name.Name,
					Type: fmt.Sprintf("%v", field.Type),
				}
				classInfo.Fields = append(classInfo.Fields, fieldInfo)
			}
		}
	}

	return classInfo
}

// calculateComplexity calculates cyclomatic complexity
func (p *GoParser) calculateComplexity(funcDecl *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			// Count logical operators
			if binExpr, ok := n.(*ast.BinaryExpr); ok {
				if binExpr.Op == token.LAND || binExpr.Op == token.LOR {
					complexity++
				}
			}
		}
		return true
	})

	return complexity
}

// calculateMetrics calculates overall file metrics
func (p *GoParser) calculateMetrics(content string, analysis *FileAnalysis) CodeMetrics {
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
	metrics.CyclomaticComplexity = totalComplexity

	return metrics
}
