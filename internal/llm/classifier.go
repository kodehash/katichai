package llm

import (
	"context"
	"fmt"
	"strings"
)

// ClassificationType represents the type of change
type ClassificationType string

const (
	ClassBoilerplate ClassificationType = "BOILERPLATE"
	ClassLogic       ClassificationType = "LOGIC"
	ClassRefactor    ClassificationType = "REFACTOR"
	ClassDocs        ClassificationType = "DOCS"
	ClassTests       ClassificationType = "TESTS"
	ClassConfig      ClassificationType = "CONFIG"
	ClassUnknown     ClassificationType = "UNKNOWN"
)

// ClassificationResult holds the classification
type ClassificationResult struct {
	Type       ClassificationType
	Confidence float64
	Reasoning  string
}

// Classifier classifies code changes
type Classifier struct {
	client LLMProvider
}

// NewClassifier creates a new classifier
func NewClassifier(client LLMProvider) *Classifier {
	return &Classifier{
		client: client,
	}
}

// ClassifyChanges determines the type of change from a diff
func (c *Classifier) ClassifyChanges(ctx context.Context, diff string) (ClassificationResult, error) {
	prompt := fmt.Sprintf(`You are a code classification system. Analyze the following git diff and classify the primary nature of the changes into exactly ONE of these categories:
- BOILERPLATE: Generated code, getters/setters, simple struct definitions.
- LOGIC: New business logic, algorithms, or feature implementation.
- REFACTOR: Code restructuring without behavior change, renaming, moving files.
- DOCS: Documentation updates, comments only.
- TESTS: Adding or modifying tests.
- CONFIG: Configuration file changes.

Return your response in the following format:
CATEGORY: [One of the above]
CONFIDENCE: [0.0 to 1.0]
REASONING: [One sentence explanation]

Diff:
%s
`, diff)

	req := CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a precise code classifier."},
			{Role: RoleUser, Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   100,
	}

	resp, err := c.client.GenerateCompletion(ctx, req)
	if err != nil {
		return ClassificationResult{Type: ClassUnknown}, err
	}

	return c.parseResponse(resp.Content), nil
}

func (c *Classifier) parseResponse(content string) ClassificationResult {
	lines := strings.Split(content, "\n")
	result := ClassificationResult{Type: ClassUnknown}

	for _, line := range lines {
		if strings.HasPrefix(line, "CATEGORY:") {
			cat := strings.TrimSpace(strings.TrimPrefix(line, "CATEGORY:"))
			result.Type = ClassificationType(cat)
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			fmt.Sscanf(strings.TrimPrefix(line, "CONFIDENCE:"), "%f", &result.Confidence)
		} else if strings.HasPrefix(line, "REASONING:") {
			result.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
		}
	}
	return result
}
