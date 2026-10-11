package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callTool(t *testing.T, name string, arguments map[string]any) map[string]any {
	t.Helper()
	for _, tool := range pluginTools() {
		if tool.Name != name {
			continue
		}
		assert.True(t, tool.ReadOnly, "%s must be read-only", name)
		raw, err := json.Marshal(arguments)
		require.NoError(t, err)
		value, err := tool.Handle(context.Background(), raw)
		require.NoError(t, err)
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		return decoded
	}
	t.Fatalf("no tool %s", name)
	return nil
}

func TestMCPToolsSearchPreviewAndCheckAProject(t *testing.T) {
	registry := fixtureRegistry(t)
	root := downstreamProject(t)
	t.Chdir(root)

	found := callTool(t, "plugin_search", map[string]any{"query": "我们需要笔记", "source": registry})
	plugins := found["plugins"].([]any)
	require.Len(t, plugins, 1)
	assert.Equal(t, "demo-notes", plugins[0].(map[string]any)["name"])

	preview := callTool(t, "plugin_preview", map[string]any{"name": "demo-notes", "source": registry})
	assert.Len(t, preview["writes"], 2)
	assert.Len(t, preview["wiring"], 6)
	assert.NoFileExists(t, root+"/api/internal/modules/demonotes/doc.go", "a preview writes nothing")

	info := callTool(t, "project_info", nil)
	assert.Equal(t, "github.com/example-demo/platform/api", info["module"])
	assert.Empty(t, callTool(t, "plugin_doctor", nil)["problems"])
	assert.NotEmpty(t, callTool(t, "plugin_slots", nil)["slots"])
}
