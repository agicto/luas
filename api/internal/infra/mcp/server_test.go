package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, server *Server, lines ...string) []map[string]any {
	t.Helper()
	var out strings.Builder
	require.NoError(t, server.Serve(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n"), &out))
	var responses []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &decoded))
		responses = append(responses, decoded)
	}
	return responses
}

func echoServer() *Server {
	return NewServer("test", "1.0.0", "Use echo.", Tool{
		Name: "echo", Description: "Echo a word.", ReadOnly: true,
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"word": map[string]any{"type": "string"}}},
		Handle: func(_ context.Context, raw json.RawMessage) (any, error) {
			var args struct {
				Word string `json:"word"`
			}
			if err := DecodeArguments(raw, &args); err != nil {
				return nil, err
			}
			if args.Word == "" {
				return nil, errors.New("word is required")
			}
			return map[string]any{"word": args.Word}, nil
		},
	})
}

func TestServerSpeaksTheToolProtocol(t *testing.T) {
	responses := serve(t, echoServer(),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"word":"hi"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"echo","arguments":{"other":1}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"resources/list"}`,
		`not json`,
	)
	require.Len(t, responses, 7, "the notification gets no answer")

	initialize := responses[0]["result"].(map[string]any)
	assert.Equal(t, ProtocolVersion, initialize["protocolVersion"])
	assert.Equal(t, "Use echo.", initialize["instructions"])

	tools := responses[1]["result"].(map[string]any)["tools"].([]any)
	require.Len(t, tools, 1)
	assert.Equal(t, true, tools[0].(map[string]any)["annotations"].(map[string]any)["readOnlyHint"])

	ok := responses[2]["result"].(map[string]any)
	assert.Equal(t, map[string]any{"word": "hi"}, ok["structuredContent"])
	assert.Nil(t, ok["isError"])

	failed := responses[3]["result"].(map[string]any)
	assert.Equal(t, true, failed["isError"])
	assert.Contains(t, failed["content"].([]any)[0].(map[string]any)["text"], "word is required")
	assert.Equal(t, true, responses[4]["result"].(map[string]any)["isError"], "unknown arguments are rejected")

	assert.InDelta(t, codeMethodNotFound, responses[5]["error"].(map[string]any)["code"], 0)
	assert.InDelta(t, codeParseError, responses[6]["error"].(map[string]any)["code"], 0)
}
