package analysis

import (
	"testing"
)

func TestCheckDrift(t *testing.T) {
	detector := NewDriftDetector()

	code := `
package main

func badStyle() {
	user_id := 123
	if nil != err {
		return
	}
}
`
	report := detector.CheckDrift(code, "Go")

	if len(report.Issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(report.Issues))
	}

	foundNaming := false
	foundError := false

	for _, issue := range report.Issues {
		if issue.Severity == SeverityWarning {
			foundNaming = true
		}
		if issue.Severity == SeverityInfo {
			foundError = true
		}
	}

	if !foundNaming {
		t.Error("Failed to detect snake_case drift")
	}
	if !foundError {
		t.Error("Failed to detect error handling drift")
	}
}
