package analysis

import (
	"os"
	"testing"

	"github.com/katichai/katich/internal/context"
)

func TestRegexParser_CSharp(t *testing.T) {
	content := `
using System;

namespace Example {
    public class MyService {
        // Comment
        public async Task<string> GetData(int id) {
            if (id > 0) {
                return "data";
            }
            return null;
        }

        private void Process() {
            Console.WriteLine("Processing");
        }
    }
}
`
	tmp, _ := os.CreateTemp("", "*.cs")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	parser := NewRegexParser(string(context.LanguageCSharp))
	analysis, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(analysis.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(analysis.Functions))
	}
	
	if analysis.Functions[0].Name != "GetData" {
		t.Errorf("Expected first func 'GetData', got '%s'", analysis.Functions[0].Name)
	}
}
