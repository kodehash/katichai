package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "katich"
	serverVersion   = "1.0.0"
)

// JSON-RPC types

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol types

type initializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    struct{}   `json:"capabilities"`
	ClientInfo      clientInfo `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResult struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    serverCapability `json:"capabilities"`
	ServerInfo      serverInfo       `json:"serverInfo"`
}

type serverCapability struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsListResult struct {
	Tools []toolDefinition `json:"tools"`
}

type toolDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	InputSchema toolInputSchema     `json:"inputSchema"`
}

type toolInputSchema struct {
	Type       string                        `json:"type"`
	Properties map[string]propertyDefinition `json:"properties,omitempty"`
	Required   []string                      `json:"required,omitempty"`
}

type propertyDefinition struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type toolCallResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server is the MCP stdio server.
type Server struct {
	reader  io.Reader
	writer  io.Writer
	mu      sync.Mutex
	tools   *ToolRegistry
}

// NewServer creates an MCP server reading from r and writing to w.
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		reader: r,
		writer: w,
		tools:  NewToolRegistry(),
	}
}

// Serve runs the MCP server on stdin/stdout until EOF.
func Serve() error {
	s := NewServer(os.Stdin, os.Stdout)
	return s.Run()
}

// Run processes JSON-RPC messages line by line until the input closes.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.reader)
	// Allow large messages (16 MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, -32700, fmt.Sprintf("parse error: %v", err))
			continue
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *jsonrpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// Acknowledgement, no response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolCall(req)
	case "ping":
		s.writeResult(req.ID, map[string]interface{}{})
	default:
		if req.ID != nil {
			s.writeError(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
	}
}

func (s *Server) handleInitialize(req *jsonrpcRequest) {
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: serverCapability{
			Tools: &toolsCapability{ListChanged: false},
		},
		ServerInfo: serverInfo{
			Name:    serverName,
			Version: serverVersion,
		},
	}
	s.writeResult(req.ID, result)
}

func (s *Server) handleToolsList(req *jsonrpcRequest) {
	defs := s.tools.Definitions()
	result := toolsListResult{Tools: defs}
	s.writeResult(req.ID, result)
}

func (s *Server) handleToolCall(req *jsonrpcRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
		return
	}

	output, toolErr := s.tools.Call(params.Name, params.Arguments)

	result := toolCallResult{
		Content: []contentBlock{{Type: "text", Text: output}},
		IsError: toolErr != nil,
	}
	if toolErr != nil {
		result.Content[0].Text = toolErr.Error()
	}

	s.writeResult(req.ID, result)
}

func (s *Server) writeResult(id json.RawMessage, result interface{}) {
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) writeError(id json.RawMessage, code int, message string) {
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	s.send(resp)
}

func (s *Server) send(resp jsonrpcResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	data = append(data, '\n')
	s.writer.Write(data)
}
