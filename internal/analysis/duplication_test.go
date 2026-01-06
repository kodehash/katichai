package analysis

import (
	"testing"
)

func TestExactDuplication(t *testing.T) {
	detector := NewExactDuplicationDetector()

	// Functions must have at least 4 lines to be considered for duplication
	fn1 := FunctionInfo{
		Name:      "FuncA",
		StartLine: 10,
		EndLine:   20,
		Body: `func A() {
	x := 1
	y := 2
	z := 3
	return x + y + z
}`,
	}

	fn2 := FunctionInfo{
		Name:      "FuncB",
		StartLine: 30,
		EndLine:   40,
		Body: `func B() {
	a := 1
	b := 2
	return a + b
}`,
	}

	// Duplicate of Fn1, different whitespace
	fn3 := FunctionInfo{
		Name:      "FuncCopy",
		StartLine: 50,
		EndLine:   60,
		Body: `func A() {
		x := 1
		y := 2
		z := 3
		return x + y + z
	}`,
	}

	detector.AddFunction("file1.go", fn1)
	detector.AddFunction("file1.go", fn2)
	
	// Check existing
	dups := detector.DetectDuplicates(fn3.Body)
	if len(dups) != 1 {
		t.Fatalf("Expected 1 duplicate, got %d", len(dups))
	}
	
	if dups[0].FuncName != "FuncA" {
		t.Errorf("Expected duplicate of FuncA, got %s", dups[0].FuncName)
	}
}
