package commands

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/mcp"
	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

// MCPCommand serves the plugin registry to coding agents over the Model Context Protocol. Every
// tool is read-only: agents discover, decide, and preview here, and change the project with the
// plugin commands, where the builder sees each step.
type MCPCommand struct{}

func NewMCPCommand() *MCPCommand { return &MCPCommand{} }

func (c *MCPCommand) Name() string { return "mcp" }
func (c *MCPCommand) Description() string {
	return "Serve the plugin registry to coding agents over MCP (stdio)"
}
func (c *MCPCommand) Usage() string { return "mcp" }

// SuppressCompletionOutput keeps stdout pure protocol messages.
func (c *MCPCommand) SuppressCompletionOutput() bool { return true }

const mcpInstructions = `FastBuild project tools. Before building a capability, theme, or page section, call ` +
	`plugin_search with the builder's words; when a plugin's use_when matches, call plugin_info and ` +
	`plugin_preview, then install it with the CLI: cd api && go run ./cmd/fastbuild plugin:add <name>, ` +
	`and complete its INTEGRATION.md steps. Call plugin_doctor before declaring integration done.`

func (c *MCPCommand) Run([]string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := mcp.NewServer("fastbuild", currentScaffoldVersion(), mcpInstructions, pluginTools()...)
	return server.Serve(ctx, os.Stdin, os.Stdout)
}

func currentScaffoldVersion() string {
	if root, err := findProjectRoot(); err == nil {
		if marker, readErr := readProjectMarker(root); readErr == nil {
			return marker.ScaffoldVersion
		}
		return scaffoldVersion(root)
	}
	return "unknown"
}

func readProjectMarker(root string) (*ProjectMarker, error) {
	raw, err := os.ReadFile(filepath.Join(root, projectMarkerFile))
	if err != nil {
		return nil, err
	}
	var marker ProjectMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return nil, err
	}
	return &marker, nil
}

func object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

var (
	sourceProperty = map[string]any{"type": "string", "description": "Registry source: a path or github:owner/repo@tag. Defaults to the project's registry."}
	nameProperty   = map[string]any{"type": "string", "description": "Plugin name from plugin_search."}
)

func pluginTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "project_info", Title: "Project", ReadOnly: true,
			Description: "The project's Go module, FastBuild version, available starters, and installed plugins.",
			InputSchema: object(map[string]any{}),
			Handle: func(context.Context, json.RawMessage) (any, error) {
				root, err := findProjectRoot()
				if err != nil {
					return nil, err
				}
				module, err := projectModulePath(root)
				if err != nil {
					return nil, err
				}
				lock, err := pluginregistry.ReadLock(root)
				if err != nil {
					return nil, err
				}
				installed := map[string]string{}
				for name, entry := range lock.Plugins {
					installed[name] = entry.Version
				}
				return map[string]any{
					"module": module, "fastbuild_version": currentScaffoldVersion(),
					"starters": config.StarterNames(), "installed_plugins": installed,
				}, nil
			},
		},
		{
			Name: "plugin_search", Title: "Search plugins", ReadOnly: true,
			Description: "Rank registry plugins (features, themes, blocks) against a request in any language.",
			InputSchema: object(map[string]any{
				"query":  map[string]any{"type": "string", "description": "The builder's request or keywords."},
				"kind":   map[string]any{"type": "string", "enum": []string{"feature", "theme", "block"}},
				"source": sourceProperty,
			}),
			Handle: func(_ context.Context, raw json.RawMessage) (any, error) {
				var args struct{ Query, Kind, Source string }
				if err := mcp.DecodeArguments(raw, &args); err != nil {
					return nil, err
				}
				source, index, err := mcpIndex(args.Source)
				if err != nil {
					return nil, err
				}
				defer closeSource(source)
				type match struct {
					Name     string   `json:"name"`
					Kind     string   `json:"kind"`
					Version  string   `json:"version"`
					Summary  string   `json:"summary"`
					UseWhen  []string `json:"use_when"`
					Requires []string `json:"requires_starters"`
					Score    int      `json:"score"`
				}
				matches := []match{}
				for _, result := range index.Search(args.Query) {
					if args.Kind != "" && result.Entry.PluginKind() != args.Kind {
						continue
					}
					matches = append(matches, match{result.Entry.Name, result.Entry.PluginKind(), result.Entry.Version,
						result.Entry.Summary, result.Entry.UseWhen, result.Entry.Requires.Starters, result.Score})
				}
				return map[string]any{"source": source.Describe(), "plugins": matches}, nil
			},
		},
		{
			Name: "plugin_info", Title: "Plugin details", ReadOnly: true,
			Description: "A plugin's manifest, files, configuration, and integration steps.",
			InputSchema: object(map[string]any{"name": nameProperty, "source": sourceProperty}, "name"),
			Handle: func(_ context.Context, raw json.RawMessage) (any, error) {
				var args struct{ Name, Source string }
				if err := mcp.DecodeArguments(raw, &args); err != nil {
					return nil, err
				}
				source, index, err := mcpIndex(args.Source)
				if err != nil {
					return nil, err
				}
				defer closeSource(source)
				entry, ok := index.Find(args.Name)
				if !ok {
					return nil, errors.New("no plugin named " + args.Name + "; call plugin_search")
				}
				integration, docErr := source.ReadDocument(entry.Name, pluginregistry.IntegrationFile)
				if docErr != nil && !errors.Is(docErr, os.ErrNotExist) {
					return nil, docErr
				}
				return map[string]any{"source": source.Describe(), "plugin": entry, "integration": string(integration)}, nil
			},
		},
		{
			Name: "plugin_preview", Title: "Preview an install", ReadOnly: true,
			Description: "What plugin:add would write and which core slots it would edit, without changing anything.",
			InputSchema: object(map[string]any{"name": nameProperty, "source": sourceProperty}, "name"),
			Handle: func(_ context.Context, raw json.RawMessage) (any, error) {
				var args struct{ Name, Source string }
				if err := mcp.DecodeArguments(raw, &args); err != nil {
					return nil, err
				}
				root, err := findProjectRoot()
				if err != nil {
					return nil, err
				}
				source, err := resolvePluginSource(sourceArgs(args.Source))
				if err != nil {
					return nil, err
				}
				defer closeSource(source)
				installer, err := newProjectInstaller(root, source)
				if err != nil {
					return nil, err
				}
				installer.DryRun = true
				result, err := installer.Install(args.Name)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"plugin": result.Entry.Name, "version": result.Entry.Version, "writes": result.Written,
					"wiring": result.Wiring, "merges_contract": result.Entry.Fragment != "" && installer.Contract != nil,
					"install_command": "cd api && go run ./cmd/fastbuild plugin:add " + result.Entry.Name + sourceSuffix(args.Source),
				}, nil
			},
		},
		{
			Name: "plugin_status", Title: "Installed plugins", ReadOnly: true,
			Description: "Installed plugins with each file's state since installation: unchanged, modified, or missing.",
			InputSchema: object(map[string]any{}),
			Handle: func(context.Context, json.RawMessage) (any, error) {
				root, err := findProjectRoot()
				if err != nil {
					return nil, err
				}
				lock, err := pluginregistry.ReadLock(root)
				if err != nil {
					return nil, err
				}
				plugins := map[string]any{}
				for name, entry := range lock.Plugins {
					plugins[name] = map[string]any{"version": entry.Version, "kind": entry.Kind, "source": entry.Source,
						"files": pluginregistry.Status(root, entry)}
				}
				return map[string]any{"plugins": plugins}, nil
			},
		},
		{
			Name: "plugin_doctor", Title: "Integration check", ReadOnly: true,
			Description: "Unfinished integration steps of installed plugins, each with its fix. Empty means done.",
			InputSchema: object(map[string]any{}),
			Handle: func(context.Context, json.RawMessage) (any, error) {
				root, err := findProjectRoot()
				if err != nil {
					return nil, err
				}
				lock, err := pluginregistry.ReadLock(root)
				if err != nil {
					return nil, err
				}
				problems := pluginregistry.Doctor(root, lock)
				if problems == nil {
					problems = []pluginregistry.Problem{}
				}
				return map[string]any{"problems": problems}, nil
			},
		},
		{
			Name: "plugin_slots", Title: "Extension points", ReadOnly: true,
			Description: "Named slots where plugins insert lines into core files, for writing a plugin's wiring.",
			InputSchema: object(map[string]any{}),
			Handle: func(context.Context, json.RawMessage) (any, error) {
				return map[string]any{"marker": pluginregistry.SlotMarker, "slots": pluginregistry.ProjectSlots}, nil
			},
		},
	}
}

func sourceArgs(source string) []string {
	if source == "" {
		return nil
	}
	return []string{"--source=" + source}
}

func sourceSuffix(source string) string {
	if source == "" {
		return ""
	}
	return " --source=" + source
}

func mcpIndex(source string) (pluginregistry.Source, *pluginregistry.Index, error) {
	resolved, err := resolvePluginSource(sourceArgs(strings.TrimSpace(source)))
	if err != nil {
		return nil, nil, err
	}
	index, err := resolved.Index()
	if err != nil {
		closeSource(resolved)
		return nil, nil, err
	}
	return resolved, index, nil
}
