// Package mcp serves tools to coding agents over the Model Context Protocol: JSON-RPC 2.0 messages,
// one per line, on stdin and stdout. It implements the tool subset of the protocol, which is all a
// command-line project tool needs.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ProtocolVersion is the MCP revision this server speaks.
const ProtocolVersion = "2025-06-18"

// maxMessageBytes bounds one request line.
const maxMessageBytes = 4 << 20

// Tool is one callable tool. Handle returns a JSON object (a map or struct), sent both as text and
// as structured content; an error becomes a tool result with isError set, so the agent sees the
// message and can recover.
type Tool struct {
	Name        string
	Title       string
	Description string
	// InputSchema is a JSON Schema object describing the arguments.
	InputSchema map[string]any
	// ReadOnly marks tools that never change the project.
	ReadOnly bool
	Handle   func(ctx context.Context, arguments json.RawMessage) (any, error)
}

// Server dispatches requests to tools.
type Server struct {
	Name         string
	Version      string
	Instructions string
	tools        []Tool
}

// NewServer returns a server exposing the given tools.
func NewServer(name, version, instructions string, tools ...Tool) *Server {
	return &Server{Name: name, Version: version, Instructions: instructions, tools: tools}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Serve reads requests until in closes or ctx ends, answering each request in order.
// Notifications, which carry no id, get no answer.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64<<10), maxMessageBytes)
	var writeMu sync.Mutex
	encoder := json.NewEncoder(out)
	write := func(message response) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return encoder.Encode(message)
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			if writeErr := write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParseError, "invalid JSON"}}); writeErr != nil {
				return writeErr
			}
			continue
		}
		if len(req.ID) == 0 {
			continue // notification, such as notifications/initialized
		}
		result, rpcErr := s.dispatch(ctx, req)
		if err := write(response{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) dispatch(ctx context.Context, req request) (any, *rpcError) {
	if req.JSONRPC != "2.0" {
		return nil, &rpcError{codeInvalidRequest, "jsonrpc must be 2.0"}
	}
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
			"instructions":    s.Instructions,
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.tools))
		for _, tool := range s.tools {
			tools = append(tools, map[string]any{
				"name":        tool.Name,
				"title":       tool.Title,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
				"annotations": map[string]any{"readOnlyHint": tool.ReadOnly, "openWorldHint": false},
			})
		}
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		return s.call(ctx, req.Params)
	default:
		return nil, &rpcError{codeMethodNotFound, "method not found: " + req.Method}
	}
}

func (s *Server) call(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil || call.Name == "" {
		return nil, &rpcError{codeInvalidParams, "tools/call needs a tool name"}
	}
	for _, tool := range s.tools {
		if tool.Name != call.Name {
			continue
		}
		if len(call.Arguments) == 0 {
			call.Arguments = json.RawMessage("{}")
		}
		value, err := tool.Handle(ctx, call.Arguments)
		if err != nil {
			return map[string]any{"content": []map[string]any{{"type": "text", "text": err.Error()}}, "isError": true}, nil
		}
		text, marshalErr := json.MarshalIndent(value, "", "  ")
		if marshalErr != nil {
			return nil, &rpcError{codeInvalidParams, marshalErr.Error()}
		}
		return map[string]any{
			"content":           []map[string]any{{"type": "text", "text": string(text)}},
			"structuredContent": value,
		}, nil
	}
	return nil, &rpcError{codeInvalidParams, fmt.Sprintf("unknown tool %q", call.Name)}
}

// DecodeArguments decodes tool arguments strictly, rejecting unknown fields.
func DecodeArguments(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}
