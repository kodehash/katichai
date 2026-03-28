package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

// inProcessExchange sends JSON-RPC lines to an in-process Server.
func inProcessExchange(t *testing.T, lines ...string) []json.RawMessage {
	t.Helper()
	input := strings.Join(lines, "\n") + "\n"
	var out bytes.Buffer
	s := NewServer(strings.NewReader(input), &out)
	if err := s.Run(); err != nil {
		t.Fatalf("server.Run: %v", err)
	}

	var responses []json.RawMessage
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		responses = append(responses, json.RawMessage(line))
	}
	return responses
}

func parseResponse(t *testing.T, raw json.RawMessage) (id json.RawMessage, result map[string]interface{}, rpcErr map[string]interface{}) {
	t.Helper()
	var resp struct {
		ID     json.RawMessage        `json:"id"`
		Result json.RawMessage        `json:"result"`
		Error  map[string]interface{} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response: %v\n%s", err, string(raw))
	}
	id = resp.ID
	rpcErr = resp.Error
	if resp.Result != nil {
		result = make(map[string]interface{})
		json.Unmarshal(resp.Result, &result)
	}
	return
}

func extractToolText(t *testing.T, result map[string]interface{}) string {
	t.Helper()
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("result has no content blocks")
	}
	block, _ := content[0].(map[string]interface{})
	text, _ := block["text"].(string)
	return text
}

const (
	initLine   = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}`
	notifyLine = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
)

// ═══════════════════════════════════════════════════════════════════════════
//  PROTOCOL TESTS — in-process, no binary, no git repo needed
// ═══════════════════════════════════════════════════════════════════════════

func TestProtocol_InitializeHandshake(t *testing.T) {
	responses := inProcessExchange(t, initLine)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	_, result, rpcErr := parseResponse(t, responses[0])
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	if result["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], protocolVersion)
	}
	si, _ := result["serverInfo"].(map[string]interface{})
	if si["name"] != serverName {
		t.Errorf("serverInfo.name = %v, want %v", si["name"], serverName)
	}
	caps, _ := result["capabilities"].(map[string]interface{})
	tools, _ := caps["tools"].(map[string]interface{})
	if tools == nil {
		t.Error("capabilities.tools missing")
	}
}

func TestProtocol_ToolsListReturnsAllTools(t *testing.T) {
	responses := inProcessExchange(t, initLine, notifyLine,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	)
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	_, result, _ := parseResponse(t, responses[1])
	toolsList, _ := result["tools"].([]interface{})

	expected := map[string]bool{
		"katich_review_latest": false, "katich_review_diff": false,
		"katich_review_full": false, "katich_context_build": false,
		"katich_doctor": false, "katich_version": false,
	}
	for _, tool := range toolsList {
		tm, _ := tool.(map[string]interface{})
		name, _ := tm["name"].(string)
		expected[name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if len(toolsList) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(toolsList))
	}
}

func TestProtocol_Ping(t *testing.T) {
	responses := inProcessExchange(t, initLine,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	)
	_, result, rpcErr := parseResponse(t, responses[1])
	if rpcErr != nil {
		t.Fatalf("unexpected error: %v", rpcErr)
	}
	if len(result) != 0 {
		t.Errorf("ping result should be empty, got %v", result)
	}
}

func TestProtocol_UnknownMethodReturnsError(t *testing.T) {
	responses := inProcessExchange(t, initLine,
		`{"jsonrpc":"2.0","id":2,"method":"bogus/method"}`,
	)
	_, _, rpcErr := parseResponse(t, responses[1])
	if rpcErr == nil {
		t.Fatal("expected error for unknown method")
	}
	code, _ := rpcErr["code"].(float64)
	if int(code) != -32601 {
		t.Errorf("error code = %v, want -32601", code)
	}
}

func TestProtocol_MalformedJSON(t *testing.T) {
	responses := inProcessExchange(t, `{not valid json`)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	_, _, rpcErr := parseResponse(t, responses[0])
	code, _ := rpcErr["code"].(float64)
	if int(code) != -32700 {
		t.Errorf("error code = %v, want -32700", code)
	}
}

func TestProtocol_NotificationProducesNoResponse(t *testing.T) {
	responses := inProcessExchange(t, notifyLine)
	if len(responses) != 0 {
		t.Errorf("expected 0 responses for notification, got %d", len(responses))
	}
}

func TestProtocol_EmptyLinesIgnored(t *testing.T) {
	responses := inProcessExchange(t, "", initLine, "",
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`, "",
	)
	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}
}

func TestProtocol_SequentialIDs(t *testing.T) {
	responses := inProcessExchange(t, initLine, notifyLine,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	)
	if len(responses) != 4 {
		t.Fatalf("expected 4 responses, got %d", len(responses))
	}
	for i, want := range []string{"1", "2", "3", "4"} {
		id, _, _ := parseResponse(t, responses[i])
		if strings.TrimSpace(string(id)) != want {
			t.Errorf("response %d: id = %s, want %s", i, id, want)
		}
	}
}

func TestProtocol_UnknownToolReturnsIsError(t *testing.T) {
	responses := inProcessExchange(t, initLine,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"does_not_exist","arguments":{}}}`,
	)
	_, result, _ := parseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	if !isErr {
		t.Error("expected isError=true for unknown tool")
	}
	text := extractToolText(t, result)
	if !strings.Contains(text, "unknown tool") {
		t.Errorf("expected 'unknown tool' in error, got %q", text)
	}
}

func TestProtocol_ReviewDiffMissingRange(t *testing.T) {
	responses := inProcessExchange(t, initLine,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"katich_review_diff","arguments":{}}}`,
	)
	_, result, _ := parseResponse(t, responses[1])
	isErr, _ := result["isError"].(bool)
	if !isErr {
		t.Error("expected isError=true when range is missing")
	}
	text := extractToolText(t, result)
	if !strings.Contains(text, "range") {
		t.Errorf("expected 'range' in error, got %q", text)
	}
}

func TestProtocol_ToolDefinitionsHaveValidSchemas(t *testing.T) {
	registry := NewToolRegistry()
	for _, def := range registry.Definitions() {
		if def.Name == "" {
			t.Error("tool has empty name")
		}
		if def.Description == "" {
			t.Errorf("tool %q has empty description", def.Name)
		}
		if def.InputSchema.Type != "object" {
			t.Errorf("tool %q schema type = %q, want 'object'", def.Name, def.InputSchema.Type)
		}
		for _, req := range def.InputSchema.Required {
			if _, ok := def.InputSchema.Properties[req]; !ok {
				t.Errorf("tool %q required field %q not in properties", def.Name, req)
			}
		}
	}
}

func TestProtocol_InvalidToolCallParams(t *testing.T) {
	responses := inProcessExchange(t, initLine,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":"not an object"}`,
	)
	_, _, rpcErr := parseResponse(t, responses[1])
	if rpcErr == nil {
		t.Fatal("expected error for invalid tool call params")
	}
	code, _ := rpcErr["code"].(float64)
	if int(code) != -32602 {
		t.Errorf("error code = %v, want -32602 (invalid params)", code)
	}
}
