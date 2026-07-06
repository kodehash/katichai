//go:build integration

package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var katichBin string

func TestMain(m *testing.M) {
	modRoot := findModuleRoot()
	if modRoot == "" {
		fmt.Fprintln(os.Stderr, "cannot find module root (go.mod)")
		os.Exit(1)
	}

	katichBin = filepath.Join(modRoot, ".test-katich-bin")
	cmd := exec.Command("go", "build", "-o", katichBin, "./cmd/katich/main.go")
	cmd.Dir = modRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build katich: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Remove(katichBin)
	os.Exit(code)
}

func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// mcpExchange sends JSON-RPC lines to the built katich binary over stdio.
func mcpExchange(t *testing.T, workDir string, lines ...string) []json.RawMessage {
	t.Helper()
	input := strings.Join(lines, "\n") + "\n"

	cmd := exec.Command(katichBin, "mcp")
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("katich mcp failed: %v\nstderr: %s", err, stderr.String())
	}

	var responses []json.RawMessage
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		responses = append(responses, json.RawMessage(line))
	}
	return responses
}

func toolCallRequest(id int, name string, args map[string]interface{}) string {
	if args == nil {
		args = map[string]interface{}{}
	}
	params := map[string]interface{}{"name": name, "arguments": args}
	p, _ := json.Marshal(params)
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":%s}`, id, string(p))
}

func intParseResponse(t *testing.T, raw json.RawMessage) (result map[string]interface{}, rpcErr map[string]interface{}) {
	t.Helper()
	var resp struct {
		Result json.RawMessage        `json:"result"`
		Error  map[string]interface{} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, string(raw))
	}
	rpcErr = resp.Error
	if resp.Result != nil {
		result = make(map[string]interface{})
		json.Unmarshal(resp.Result, &result)
	}
	return
}

func intExtractToolText(t *testing.T, result map[string]interface{}) string {
	t.Helper()
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("result has no content blocks")
	}
	block, _ := content[0].(map[string]interface{})
	text, _ := block["text"].(string)
	return text
}

// createTestRepo creates a temp git repo with real source files, two commits,
// and katich initialized. Returns path and cleanup func.
func createTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "katich-integ-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	cleanup := func() { os.RemoveAll(dir) }

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			cleanup()
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	// Go source file
	os.MkdirAll(filepath.Join(dir, "pkg"), 0755)
	os.WriteFile(filepath.Join(dir, "pkg", "handler.go"), []byte(`package pkg

import "fmt"

func HandleRequest(method string, path string) (int, error) {
	if method == "" {
		return 400, fmt.Errorf("method required")
	}
	if path == "" {
		return 400, fmt.Errorf("path required")
	}
	switch method {
	case "GET":
		return 200, nil
	case "POST":
		return 201, nil
	default:
		return 405, fmt.Errorf("unsupported method: %s", method)
	}
}
`), 0644)

	// Python file with a deliberate issue (password in env without validation)
	os.WriteFile(filepath.Join(dir, "app.py"), []byte(`import os

def connect_db():
    host = os.getenv("DB_HOST", "localhost")
    password = os.getenv("DB_PASSWORD")
    return f"postgres://{host}:{password}@db/app"

def process_data(items):
    result = []
    for item in items:
        if item.get("status") == "active":
            result.append(item)
    return result
`), 0644)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit")

	// Second commit
	os.WriteFile(filepath.Join(dir, "pkg", "utils.go"), []byte(`package pkg

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
`), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "add utils")

	// Initialize katich
	katichDir := filepath.Join(dir, ".katich")
	os.MkdirAll(katichDir, 0755)
	os.WriteFile(filepath.Join(katichDir, "config.yaml"), []byte(`llm:
  provider: openai
  model: gpt-4
  api_key: ""
  max_input_tokens: 20000
  tokens_per_minute: 90000

embeddings:
  provider: local
  model: jina-code-v2

review:
  generate_html: false
  generate_gfm: false
  detect_ai_code: false
  generate_fix_prompt: true
`), 0644)

	return dir, cleanup
}

// writeAPIKeyConfig writes a config with the detected API key.
func writeAPIKeyConfig(t *testing.T, dir string) {
	t.Helper()
	apiKey := os.Getenv("OPENAI_API_KEY")
	provider, model := "openai", "gpt-4o"
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		provider, model = "anthropic", "claude-3-5-sonnet"
	}
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY or ANTHROPIC_API_KEY not set — skipping live test")
	}

	cfg := fmt.Sprintf(`llm:
  provider: %s
  model: %s
  api_key: "%s"
  max_input_tokens: 20000
  tokens_per_minute: 90000

embeddings:
  provider: local
  model: jina-code-v2

review:
  generate_html: false
  generate_gfm: false
  detect_ai_code: false
  generate_fix_prompt: true
`, provider, model, apiKey)

	os.WriteFile(filepath.Join(dir, ".katich", "config.yaml"), []byte(cfg), 0644)
}

// ═══════════════════════════════════════════════════════════════════════════
//  INTEGRATION TESTS — real git repo, real katich binary, real tool calls
//
//  Run with:  go test ./internal/mcp/ -tags integration -v -timeout 120s
// ═══════════════════════════════════════════════════════════════════════════

func TestIntegration_VersionTool(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_version", nil),
	)
	// responses: [0]=initialize, [1]=tool call (notification produces no response)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	if isErr {
		t.Fatalf("katich_version returned error: %s", intExtractToolText(t, result))
	}

	text := intExtractToolText(t, result)
	if !strings.Contains(text, "katich version") {
		t.Errorf("expected 'katich version' in output, got: %s", text)
	}
}

func TestIntegration_DoctorInRealRepo(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_doctor", nil),
	)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	if isErr {
		t.Fatalf("katich_doctor returned error: %s", intExtractToolText(t, result))
	}

	text := intExtractToolText(t, result)

	// Doctor should find the git repo and config we created
	for _, check := range []string{"Git installation", "Git repository", "Configuration file"} {
		if !strings.Contains(text, check) {
			t.Errorf("doctor output missing %q\nFull output:\n%s", check, text)
		}
	}
	if !strings.Contains(text, "Found") {
		t.Errorf("expected at least one 'Found' in doctor output\nFull output:\n%s", text)
	}
}

func TestIntegration_ContextBuildCreatesFiles(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_context_build", nil),
	)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	text := intExtractToolText(t, result)

	if isErr && strings.Contains(text, "not in a git repository") {
		t.Fatalf("context build could not find git repo: %s", text)
	}

	// context.json must exist and have detection + analysis sections
	contextPath := filepath.Join(dir, ".katich", "context.json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("context.json not created: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("context.json is empty")
	}

	var ctx map[string]json.RawMessage
	if err := json.Unmarshal(data, &ctx); err != nil {
		t.Fatalf("context.json not valid JSON: %v", err)
	}
	if _, ok := ctx["detection"]; !ok {
		t.Error("context.json missing 'detection' key")
	}
	if _, ok := ctx["analysis"]; !ok {
		t.Error("context.json missing 'analysis' key")
	}

	// Verify detection found Go and Python
	var detection struct {
		Languages map[string]int `json:"languages"`
	}
	json.Unmarshal(ctx["detection"], &detection)
	if detection.Languages["Go"] == 0 {
		t.Error("detection did not find Go files")
	}
	if detection.Languages["Python"] == 0 {
		t.Error("detection did not find Python files")
	}
}

func TestIntegration_ContextBuildForceRebuild(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()

	// First build
	mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_context_build", nil),
	)

	// Second build with force
	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_context_build", map[string]interface{}{"force": true}),
	)

	result, _ := intParseResponse(t, responses[1])
	text := intExtractToolText(t, result)

	// Force should NOT skip with "No changes found"
	if strings.Contains(text, "No changes found") {
		t.Error("force rebuild should not skip with 'No changes found'")
	}
}

func TestIntegration_ReviewLatestWithoutAPIKey(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_review_latest", nil),
	)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)

	if !isErr {
		t.Skip("review succeeded — API key was found (skipping negative test)")
	}

	text := intExtractToolText(t, result)
	lower := strings.ToLower(text)
	keyRelated := strings.Contains(lower, "api key") ||
		strings.Contains(lower, "api_key") ||
		strings.Contains(lower, "llm") ||
		strings.Contains(lower, "provider")
	if !keyRelated {
		t.Errorf("expected error about API key or LLM, got: %s", text)
	}
}

func TestIntegration_ReviewLatestWithAPIKey(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()
	writeAPIKeyConfig(t, dir)

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_review_latest", map[string]interface{}{"format": "json"}),
	)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	text := intExtractToolText(t, result)

	if isErr {
		t.Fatalf("review with API key failed: %s", text)
	}

	var report map[string]interface{}
	if err := json.Unmarshal([]byte(text), &report); err != nil {
		t.Fatalf("review output is not valid JSON: %v\nOutput:\n%.500s", err, text)
	}
	for _, field := range []string{"summary", "issues", "score"} {
		if _, ok := report[field]; !ok {
			t.Errorf("review report missing '%s' field", field)
		}
	}
}

func TestIntegration_ReviewDiffWithRange(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()
	writeAPIKeyConfig(t, dir)

	responses := mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_review_diff", map[string]interface{}{
			"range":  "HEAD~1..HEAD",
			"format": "json",
		}),
	)

	result, _ := intParseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	text := intExtractToolText(t, result)

	if isErr {
		t.Fatalf("review diff failed: %s", text)
	}

	var report map[string]interface{}
	if err := json.Unmarshal([]byte(text), &report); err != nil {
		t.Fatalf("review diff output is not valid JSON: %v", err)
	}
	if _, ok := report["summary"]; !ok {
		t.Error("review diff report missing 'summary'")
	}
}

func TestIntegration_ReviewGFMCreatesReportFile(t *testing.T) {
	dir, cleanup := createTestRepo(t)
	defer cleanup()
	writeAPIKeyConfig(t, dir)

	mcpExchange(t, dir, initLine, notifyLine,
		toolCallRequest(2, "katich_review_latest", map[string]interface{}{"gfm": true}),
	)

	entries, err := os.ReadDir(filepath.Join(dir, ".katich", "reports"))
	if err != nil {
		t.Fatalf("could not read reports dir: %v", err)
	}

	foundMD := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			foundMD = true
			info, _ := e.Info()
			if info.Size() == 0 {
				t.Error("GFM report file is empty")
			}
		}
	}
	if !foundMD {
		t.Error("no .md report file created in .katich/reports/")
	}
}
