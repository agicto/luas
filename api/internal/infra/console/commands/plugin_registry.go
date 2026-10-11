package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/config"
	"github.com/fastagi/fastbuild/api/internal/infra/console"
	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

// registryDir is the official registry inside the scaffold repository.
const registryDir = "registry"

// scaffoldGitHubRepository is where downstream projects fetch the registry of their release. It is
// assembled from parts so project:init never rewrites it: a renamed project still installs
// official plugins from FastBuild.
const scaffoldGitHubRepository = "fast" + "agi/" + "fast" + "build"

// PluginSearchCommand lists registry plugins that match a request.
type PluginSearchCommand struct{ output *console.Output }

// PluginInfoCommand shows everything an agent needs to decide on and integrate one plugin.
type PluginInfoCommand struct{ output *console.Output }

// PluginIndexCommand regenerates or checks registry/index.json in the scaffold repository.
type PluginIndexCommand struct{ output *console.Output }

func NewPluginSearchCommand() *PluginSearchCommand { return &PluginSearchCommand{console.NewOutput()} }
func NewPluginInfoCommand() *PluginInfoCommand     { return &PluginInfoCommand{console.NewOutput()} }
func NewPluginIndexCommand() *PluginIndexCommand   { return &PluginIndexCommand{console.NewOutput()} }

func (c *PluginSearchCommand) Name() string { return "plugin:search" }
func (c *PluginSearchCommand) Description() string {
	return "Search the plugin registry for a capability"
}
func (c *PluginSearchCommand) Usage() string {
	return "plugin:search [words...] [--source=path] [--format=table|json]"
}

func (c *PluginInfoCommand) Name() string { return "plugin:info" }
func (c *PluginInfoCommand) Description() string {
	return "Show a plugin's requirements, files, configuration, and integration steps"
}
func (c *PluginInfoCommand) Usage() string {
	return "plugin:info <name> [--source=path] [--format=text|json]"
}

func (c *PluginIndexCommand) Name() string { return "plugin:index" }
func (c *PluginIndexCommand) Description() string {
	return "Regenerate registry/index.json from the plugin manifests (scaffold maintainers)"
}
func (c *PluginIndexCommand) Usage() string { return "plugin:index [--check]" }

func (c *PluginSearchCommand) Run(args []string) error {
	words, format, err := positionalAndFormat(args, "table")
	if err != nil {
		return err
	}
	source, err := resolvePluginSource(args)
	if err != nil {
		return err
	}
	defer closeSource(source)
	index, err := source.Index()
	if err != nil {
		return err
	}
	results := index.Search(strings.Join(words, " "))
	if format == "json" {
		type result struct {
			Name     string   `json:"name"`
			Version  string   `json:"version"`
			Summary  string   `json:"summary"`
			UseWhen  []string `json:"use_when"`
			Requires []string `json:"requires_starters"`
			Score    int      `json:"score"`
		}
		out := make([]result, 0, len(results))
		for _, match := range results {
			out = append(out, result{match.Entry.Name, match.Entry.Version, match.Entry.Summary,
				match.Entry.UseWhen, match.Entry.Requires.Starters, match.Score})
		}
		return writeJSON(map[string]any{"source": source.Describe(), "plugins": out})
	}
	if len(results) == 0 {
		c.output.Info("No plugin in %s matches %q.", source.Describe(), strings.Join(words, " "))
		return nil
	}
	rows := make([][]string, 0, len(results))
	for _, match := range results {
		rows = append(rows, []string{match.Entry.Name, match.Entry.Version, match.Entry.Summary})
	}
	c.output.Table([]string{"Name", "Version", "Summary"}, rows)
	c.output.Line("Next: fastbuild plugin:info <name>")
	return nil
}

func (c *PluginInfoCommand) Run(args []string) error {
	names, format, err := positionalAndFormat(args, "text")
	if err != nil {
		return err
	}
	if len(names) != 1 {
		return errors.New("usage: " + c.Usage())
	}
	source, err := resolvePluginSource(args)
	if err != nil {
		return err
	}
	defer closeSource(source)
	index, err := source.Index()
	if err != nil {
		return err
	}
	entry, ok := index.Find(names[0])
	if !ok {
		return fmt.Errorf("plugin %q is not in %s; run plugin:search", names[0], source.Describe())
	}
	integration, err := source.ReadDocument(entry.Name, pluginregistry.IntegrationFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if format == "json" {
		return writeJSON(map[string]any{"source": source.Describe(), "plugin": entry, "integration": string(integration)})
	}
	c.output.Title(entry.Name + " " + entry.Version)
	c.output.Line(entry.Summary)
	c.output.NewLine()
	c.output.Line(entry.Description)
	c.output.NewLine()
	c.output.TwoColumn("Source", source.Describe())
	c.output.TwoColumn("Requires FastBuild", ">= "+entry.FastBuild)
	c.output.TwoColumn("Requires starters", listOrDash(entry.Requires.Starters))
	c.output.TwoColumn("Requires plugins", listOrDash(entry.Requires.Plugins))
	if entry.Starter != nil {
		c.output.TwoColumn("Installs starter", entry.Starter.Name)
	}
	c.output.Section("Use when")
	for _, situation := range entry.UseWhen {
		c.output.Line("  - " + situation)
	}
	if len(entry.Env) > 0 {
		c.output.Section("Configuration")
		for _, variable := range entry.Env {
			c.output.Line("  %s — %s", variable.Name, variable.Description)
		}
	}
	c.output.Section("Files")
	for _, file := range sortedKeys(entry.Files) {
		c.output.Line("  " + file)
	}
	if len(integration) > 0 {
		c.output.Section("Integration steps")
		c.output.Line(string(integration))
	}
	c.output.Line("Install: fastbuild plugin:add %s", entry.Name)
	return nil
}

func (c *PluginIndexCommand) Run(args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	dir := filepath.Join(root, registryDir)
	index, err := pluginregistry.BuildIndex(dir)
	if err != nil {
		return err
	}
	raw, err := index.Marshal()
	if err != nil {
		return err
	}
	target := filepath.Join(dir, pluginregistry.IndexFile)
	if hasFlag(args, "check") {
		current, readErr := os.ReadFile(target)
		if readErr != nil || string(current) != string(raw) {
			return errors.New("registry/index.json is stale; run: cd api && go run ./cmd/fastbuild plugin:index")
		}
		c.output.Success("registry/index.json is current (%d plugins)", len(index.Plugins))
		return nil
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return err
	}
	c.output.Success("Wrote registry/index.json (%d plugins)", len(index.Plugins))
	return nil
}

// resolvePluginSource picks the registry: --source when given, the scaffold's own registry/ when
// present, and otherwise the FastBuild release the project started from.
func resolvePluginSource(args []string) (pluginregistry.Source, error) {
	if explicit, ok := flagValue(args, "source"); ok && explicit != "" {
		return sourceFromFlag(explicit)
	}
	root, err := findProjectRoot()
	if err != nil {
		return nil, err
	}
	if stat, statErr := os.Stat(filepath.Join(root, registryDir, pluginregistry.IndexFile)); statErr == nil && !stat.IsDir() {
		return pluginregistry.NewLocalSource(filepath.Join(root, registryDir), "local:registry"), nil
	}
	return projectReleaseSource(root)
}

func sourceFromFlag(raw string) (pluginregistry.Source, error) {
	if strings.HasPrefix(raw, "github:") {
		return pluginregistry.ParseRemoteSource(raw)
	}
	dir, err := filepath.Abs(raw)
	if err != nil {
		return nil, err
	}
	return pluginregistry.NewLocalSource(dir, "path:"+raw), nil
}

// projectReleaseSource reads the registry of the FastBuild release recorded by project:init.
func projectReleaseSource(root string) (pluginregistry.Source, error) {
	raw, err := os.ReadFile(filepath.Join(root, projectMarkerFile))
	if err != nil {
		return nil, fmt.Errorf("no registry/ here and no %s to find the FastBuild release; pass --source", projectMarkerFile)
	}
	var marker ProjectMarker
	if decodeErr := json.Unmarshal(raw, &marker); decodeErr != nil {
		return nil, fmt.Errorf("%s: %w", projectMarkerFile, decodeErr)
	}
	version, err := pluginregistry.ParseVersion(marker.ScaffoldVersion)
	if err != nil {
		return nil, fmt.Errorf("%s records scaffold version %q, which is not a release; pass --source", projectMarkerFile, marker.ScaffoldVersion)
	}
	return pluginregistry.ParseRemoteSource("github:" + scaffoldGitHubRepository + "@v" + version.String())
}

// findProjectRoot walks up from the working directory to the checkout that holds api/go.mod and
// contracts/openapi.yaml, so plugin commands work from the root or from api/.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "api", "go.mod")) && fileExists(filepath.Join(dir, "contracts", "openapi.yaml")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not inside a FastBuild checkout (api/go.mod and contracts/openapi.yaml not found)")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}

func positionalAndFormat(args []string, defaultFormat string) ([]string, string, error) {
	format := defaultFormat
	if value, ok := flagValue(args, "format"); ok {
		format = value
	}
	if format != defaultFormat && format != "json" {
		return nil, "", fmt.Errorf("--format must be %s or json", defaultFormat)
	}
	var positional []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			positional = append(positional, arg)
		}
	}
	return positional, format, nil
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name {
			return true
		}
	}
	return false
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func listOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// closeSource removes a downloaded registry once a command is done with it.
func closeSource(source pluginregistry.Source) {
	if closer, ok := source.(io.Closer); ok {
		_ = closer.Close()
	}
}

// PluginAddCommand installs a registry plugin as source and records it in the lock file.
type PluginAddCommand struct{ output *console.Output }

// PluginStatusCommand compares installed plugin files with the lock file.
type PluginStatusCommand struct{ output *console.Output }

func NewPluginAddCommand() *PluginAddCommand       { return &PluginAddCommand{console.NewOutput()} }
func NewPluginStatusCommand() *PluginStatusCommand { return &PluginStatusCommand{console.NewOutput()} }

func (c *PluginAddCommand) Name() string { return "plugin:add" }
func (c *PluginAddCommand) Description() string {
	return "Install a registry plugin into this project as source"
}
func (c *PluginAddCommand) Usage() string {
	return "plugin:add <name> [--source=path|github:owner/repo@tag] [--dry-run]"
}

func (c *PluginStatusCommand) Name() string { return "plugin:status" }
func (c *PluginStatusCommand) Description() string {
	return "Show installed plugins and which of their files changed since installation"
}
func (c *PluginStatusCommand) Usage() string { return "plugin:status [--format=table|json]" }

func (c *PluginAddCommand) Run(args []string) error {
	names, _, err := positionalAndFormat(args, "text")
	if err != nil {
		return err
	}
	if len(names) != 1 {
		return errors.New("usage: " + c.Usage())
	}
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	source, err := resolvePluginSource(args)
	if err != nil {
		return err
	}
	defer closeSource(source)
	installer, err := newProjectInstaller(root, source)
	if err != nil {
		return err
	}
	installer.DryRun = hasFlag(args, "dry-run")

	result, err := installer.Install(names[0])
	if err != nil {
		return err
	}
	entry := result.Entry
	if installer.DryRun {
		c.output.Info("Dry run: %s %s from %s would write:", entry.Name, entry.Version, source.Describe())
		for _, file := range result.Written {
			c.output.Line("  + " + file)
		}
		for _, applied := range result.Wiring {
			c.output.Line("  ~ %s (%s)", applied.File, applied.Slot)
		}
		return nil
	}
	c.output.Success("Installed %s %s from %s", entry.Name, entry.Version, source.Describe())
	for _, file := range result.Written {
		c.output.Line("  + " + file)
	}
	for _, file := range result.Edited() {
		c.output.Line("  ~ " + file)
	}
	if entry.Fragment != "" && !result.FragmentMerged {
		c.output.Warning("Node.js or contract tooling is unavailable, so %s was not merged yet.", entry.Fragment)
		c.output.Line("  cd contracts && corepack pnpm install && corepack pnpm merge-fragment ../%s", entry.Fragment)
	}
	if len(entry.Env) > 0 {
		c.output.Section("Configuration")
		for _, variable := range entry.Env {
			c.output.Line("  %s — %s", variable.Name, variable.Description)
		}
	}
	if integration, docErr := source.ReadDocument(entry.Name, pluginregistry.IntegrationFile); docErr == nil {
		c.output.Section("Integration steps (complete these, then run the checks)")
		c.output.Line(string(integration))
	}
	c.output.Line("Recorded in %s. Review the diff before committing.", pluginregistry.LockFile)
	return nil
}

func (c *PluginStatusCommand) Run(args []string) error {
	_, format, err := positionalAndFormat(args, "table")
	if err != nil {
		return err
	}
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	lock, err := pluginregistry.ReadLock(root)
	if err != nil {
		return err
	}
	type report struct {
		Name    string                      `json:"name"`
		Version string                      `json:"version"`
		Source  string                      `json:"source"`
		Starter string                      `json:"starter,omitempty"`
		Files   []pluginregistry.FileStatus `json:"files"`
	}
	reports := make([]report, 0, len(lock.Plugins))
	for _, name := range sortedLockNames(lock) {
		entry := lock.Plugins[name]
		reports = append(reports, report{name, entry.Version, entry.Source, entry.Starter, pluginregistry.Status(root, entry)})
	}
	if format == "json" {
		return writeJSON(map[string]any{"plugins": reports})
	}
	if len(reports) == 0 {
		c.output.Info("No plugins installed. Find one with: fastbuild plugin:search <words>")
		return nil
	}
	for _, item := range reports {
		counts := map[string]int{}
		for _, file := range item.Files {
			counts[file.State]++
		}
		c.output.TwoColumn(item.Name+" "+item.Version, fmt.Sprintf("%d unchanged, %d modified, %d missing (%s)",
			counts["unchanged"], counts["modified"], counts["missing"], item.Source))
		for _, file := range item.Files {
			if file.State != "unchanged" {
				c.output.Line("    %s %s", file.State, file.Path)
			}
		}
	}
	return nil
}

// newProjectInstaller configures an installer for the checkout at root: the project's scaffold
// version, its starters, its Go module path, and the contract tooling when it is installed.
func newProjectInstaller(root string, source pluginregistry.Source) (*pluginregistry.Installer, error) {
	module, err := projectModulePath(root)
	if err != nil {
		return nil, err
	}
	scaffoldVersion := ""
	if raw, readErr := os.ReadFile(filepath.Join(root, projectMarkerFile)); readErr == nil {
		var marker ProjectMarker
		if json.Unmarshal(raw, &marker) == nil {
			scaffoldVersion = marker.ScaffoldVersion
		}
	}
	installer := &pluginregistry.Installer{
		Root:            root,
		Source:          source,
		ScaffoldVersion: scaffoldVersion,
		ScaffoldModule:  scaffoldModulePath,
		KnownStarters:   config.StarterNames(),
		Transform: func(_ string, content []byte) []byte {
			if module == scaffoldModulePath {
				return content
			}
			return []byte(strings.ReplaceAll(string(content), scaffoldModulePath, module))
		},
	}
	if node, lookErr := exec.LookPath("node"); lookErr == nil && fileExists(filepath.Join(root, "contracts", "node_modules", "yaml", "package.json")) {
		installer.Contract = nodeContract{root: root, node: node}
	}
	return installer, nil
}

// nodeContract runs the contract tooling in contracts/scripts, which owns the openapi.yaml layout.
type nodeContract struct {
	root string
	node string
}

func (n nodeContract) run(args ...string) ([]byte, error) {
	command := exec.Command(n.node, args...)
	command.Dir = filepath.Join(n.root, "contracts")
	var stderr strings.Builder
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

func (n nodeContract) Merge(fragment string) (*pluginregistry.ContractAdditions, error) {
	output, err := n.run("scripts/merge-fragment.mjs", "--json", filepath.Join(n.root, filepath.FromSlash(fragment)))
	if err != nil {
		return nil, err
	}
	var additions pluginregistry.ContractAdditions
	if decodeErr := json.Unmarshal(output, &additions); decodeErr != nil {
		return nil, fmt.Errorf("merge-fragment output: %w", decodeErr)
	}
	return &additions, nil
}

func (n nodeContract) Remove(additions pluginregistry.ContractAdditions) error {
	raw, err := json.Marshal(additions)
	if err != nil {
		return err
	}
	_, err = n.run("scripts/remove-contract.mjs", string(raw))
	return err
}

func projectModulePath(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "api", "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if module, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(module), nil
		}
	}
	return "", errors.New("api/go.mod declares no module")
}

func sortedLockNames(lock *pluginregistry.Lock) []string {
	names := make([]string, 0, len(lock.Plugins))
	for name := range lock.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
