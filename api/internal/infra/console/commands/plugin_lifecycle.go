package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/console"
	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
)

// PluginUpdateCommand moves an installed plugin to a newer version, keeping local edits.
type PluginUpdateCommand struct{ output *console.Output }

// PluginRemoveCommand uninstalls a plugin: files, slot edits, contract, and lock entry.
type PluginRemoveCommand struct{ output *console.Output }

// PluginDoctorCommand reports unfinished plugin integration steps.
type PluginDoctorCommand struct{ output *console.Output }

// PluginSlotsCommand lists the extension points plugins wire into.
type PluginSlotsCommand struct{ output *console.Output }

// PluginPackCommand copies an installed plugin's files back into the scaffold registry.
type PluginPackCommand struct{ output *console.Output }

func NewPluginUpdateCommand() *PluginUpdateCommand { return &PluginUpdateCommand{console.NewOutput()} }
func NewPluginRemoveCommand() *PluginRemoveCommand { return &PluginRemoveCommand{console.NewOutput()} }
func NewPluginDoctorCommand() *PluginDoctorCommand { return &PluginDoctorCommand{console.NewOutput()} }
func NewPluginSlotsCommand() *PluginSlotsCommand   { return &PluginSlotsCommand{console.NewOutput()} }
func NewPluginPackCommand() *PluginPackCommand     { return &PluginPackCommand{console.NewOutput()} }

func (c *PluginUpdateCommand) Name() string { return "plugin:update" }
func (c *PluginUpdateCommand) Description() string {
	return "Update an installed plugin, merging the project's edits"
}
func (c *PluginUpdateCommand) Usage() string {
	return "plugin:update <name> [--source=path|github:owner/repo@tag] [--trust] [--dry-run]"
}

func (c *PluginRemoveCommand) Name() string { return "plugin:remove" }
func (c *PluginRemoveCommand) Description() string {
	return "Remove an installed plugin's files, wiring, and contract"
}
func (c *PluginRemoveCommand) Usage() string { return "plugin:remove <name> [--force] [--dry-run]" }

func (c *PluginDoctorCommand) Name() string { return "plugin:doctor" }
func (c *PluginDoctorCommand) Description() string {
	return "Check that installed plugins are completely integrated"
}
func (c *PluginDoctorCommand) Usage() string { return "plugin:doctor [--format=text|json]" }

func (c *PluginSlotsCommand) Name() string { return "plugin:slots" }
func (c *PluginSlotsCommand) Description() string {
	return "List the slots where plugins insert lines into core files"
}
func (c *PluginSlotsCommand) Usage() string { return "plugin:slots [--format=table|json]" }

func (c *PluginPackCommand) Name() string { return "plugin:pack" }
func (c *PluginPackCommand) Description() string {
	return "Copy an installed plugin's edited files back into registry/ (plugin authors)"
}
func (c *PluginPackCommand) Usage() string { return "plugin:pack <name>" }

func singleName(args []string, usage string) (string, error) {
	names, _, err := positionalAndFormat(args, "text")
	if err != nil {
		return "", err
	}
	if len(names) != 1 {
		return "", errors.New("usage: " + usage)
	}
	return names[0], nil
}

func (c *PluginUpdateCommand) Run(args []string) error {
	name, err := singleName(args, c.Usage())
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
	locked, ok := lock.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q is not installed; run plugin:add %s", name, name)
	}
	source, err := resolvePluginSource(args)
	if err != nil {
		return err
	}
	defer closeSource(source)
	if trustErr := requireTrust(source, args); trustErr != nil {
		return trustErr
	}
	installer, err := newProjectInstaller(root, source)
	if err != nil {
		return err
	}
	installer.DryRun = hasFlag(args, "dry-run")
	// The source the plugin was installed from supplies the base of three-way merges.
	var base pluginregistry.Source
	if locked.Source != source.Describe() {
		if base, err = sourceFromLabel(root, locked.Source); err == nil {
			defer closeSource(base)
		}
	}

	result, err := installer.Update(name, base)
	if err != nil {
		return err
	}
	verb := "Updated"
	if installer.DryRun {
		verb = "Dry run: would update"
	}
	c.output.Success("%s %s %s → %s from %s", verb, name, result.FromVersion, result.Entry.Version, source.Describe())
	lines := []struct {
		mark  string
		files []string
	}{
		{"+ replaced", result.Written}, {"~ merged", result.Merged}, {"! conflict", result.Conflicts},
		{"? incoming", result.Incoming}, {"- deleted", result.Deleted}, {"= kept", result.Kept},
	}
	for _, group := range lines {
		for _, file := range group.files {
			c.output.Line("  %s %s", group.mark, file)
		}
	}
	for _, applied := range result.Wired {
		c.output.Line("  + wired %s (%s)", applied.File, applied.Slot)
	}
	for _, applied := range result.Unwired {
		c.output.Line("  - unwired %s (%s)", applied.File, applied.Slot)
	}
	if result.ContractMerged {
		c.output.Line("  ~ %s", pluginregistry.ContractFile)
	}
	c.printManual(result.Manual)
	if len(result.Conflicts) > 0 || len(result.Incoming) > 0 {
		c.output.Warning("Resolve the conflicts and incoming files, then run plugin:doctor.")
	}
	if integration, docErr := source.ReadDocument(name, pluginregistry.IntegrationFile); docErr == nil && !installer.DryRun {
		c.output.Section("Integration steps (rerun the ones this update affects)")
		c.output.Line(string(integration))
	}
	return nil
}

func (c *PluginUpdateCommand) printManual(manual []string) {
	for _, step := range manual {
		c.output.Warning("Manual: %s", step)
	}
}

// sourceFromLabel reopens the source recorded in the lock file, such as a pinned release.
func sourceFromLabel(root, label string) (pluginregistry.Source, error) {
	switch {
	case strings.HasPrefix(label, "github:"):
		return pluginregistry.ParseRemoteSource(label)
	case label == "local:registry":
		return pluginregistry.NewLocalSource(filepath.Join(root, registryDir), label), nil
	case strings.HasPrefix(label, "path:"):
		return pluginregistry.NewLocalSource(strings.TrimPrefix(label, "path:"), label), nil
	}
	return nil, fmt.Errorf("unknown plugin source %q", label)
}

func (c *PluginRemoveCommand) Run(args []string) error {
	name, err := singleName(args, c.Usage())
	if err != nil {
		return err
	}
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	installer, err := newProjectInstaller(root, nil)
	if err != nil {
		return err
	}
	installer.DryRun = hasFlag(args, "dry-run")
	result, err := installer.Remove(name, pluginregistry.RemoveOptions{Force: hasFlag(args, "force")})
	if err != nil {
		return err
	}
	verb := "Removed"
	if installer.DryRun {
		verb = "Dry run: would remove"
	}
	c.output.Success("%s %s %s", verb, name, result.Entry.Version)
	for _, file := range result.Deleted {
		c.output.Line("  - " + file)
	}
	for _, applied := range result.Unwired {
		c.output.Line("  ~ %s (%s)", applied.File, applied.Slot)
	}
	if result.Entry.Contract != nil && installer.Contract != nil {
		c.output.Line("  ~ %s", pluginregistry.ContractFile)
	}
	for _, step := range result.Manual {
		c.output.Warning("Manual: %s", step)
	}
	if installer.DryRun {
		return nil
	}
	c.output.Section("Next")
	if result.Entry.Starter != "" {
		c.output.Line("  - Remove %s from OPTIONAL_STARTERS in every environment.", result.Entry.Starter)
		c.output.Line("  - cd api && make wire && make starter-catalog")
		c.output.Line("  - Tables created by the plugin's migrations stay in existing databases; drop them with a new migration if the data should go.")
	}
	for _, applied := range result.Unwired {
		if applied.Slot == "web.features" {
			c.output.Line("  - Remove the feature from NEXT_PUBLIC_OPTIONAL_FEATURES in every Web environment.")
			break
		}
	}
	if result.Entry.Contract != nil {
		c.output.Line("  - cd contracts && corepack pnpm generate && corepack pnpm check")
	}
	c.output.Line("  - Run the checks, then review the diff before committing.")
	return nil
}

func (c *PluginDoctorCommand) Run(args []string) error {
	_, format, err := positionalAndFormat(args, "text")
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
	problems := pluginregistry.Doctor(root, lock)
	if format == "json" {
		if writeErr := writeJSON(map[string]any{"plugins": len(lock.Plugins), "problems": problems}); writeErr != nil {
			return writeErr
		}
	} else {
		for _, problem := range problems {
			c.output.Error("%s: %s", problem.Plugin, problem.Problem)
			c.output.Line("    fix: %s", problem.Fix)
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d plugin integration problems", len(problems))
	}
	if format != "json" {
		c.output.Success("%d installed plugins are completely integrated", len(lock.Plugins))
	}
	return nil
}

func (c *PluginSlotsCommand) Run(args []string) error {
	_, format, err := positionalAndFormat(args, "table")
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(map[string]any{"marker": pluginregistry.SlotMarker, "slots": pluginregistry.ProjectSlots})
	}
	rows := make([][]string, 0, len(pluginregistry.ProjectSlots))
	for _, slot := range pluginregistry.ProjectSlots {
		rows = append(rows, []string{slot.Name, slot.File, slot.Description})
	}
	c.output.Table([]string{"Slot", "File", "Inserts"}, rows)
	c.output.Line("A plugin wires a slot with {\"slot\": \"<name>\", \"lines\": [...]} in plugin.json.")
	return nil
}

func (c *PluginPackCommand) Run(args []string) error {
	name, err := singleName(args, c.Usage())
	if err != nil {
		return err
	}
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	pluginDir := filepath.Join(root, registryDir, pluginregistry.PluginsDir, name)
	if _, statErr := os.Stat(filepath.Join(pluginDir, pluginregistry.ManifestFile)); statErr != nil {
		return fmt.Errorf("registry/plugins/%s does not exist; plugin:pack runs in the scaffold repository", name)
	}
	lock, err := pluginregistry.ReadLock(root)
	if err != nil {
		return err
	}
	locked, ok := lock.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q is not installed here; install it, edit it, then pack it", name)
	}
	packed := 0
	for _, status := range pluginregistry.Status(root, locked) {
		if status.State != "modified" {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(status.Path)))
		if readErr != nil {
			return readErr
		}
		if writeErr := os.WriteFile(pluginregistry.PluginFilePath(filepath.Join(root, registryDir), name, status.Path), content, 0o644); writeErr != nil {
			return writeErr
		}
		locked.Files[status.Path] = pluginregistry.Checksum(content)
		c.output.Line("  ~ %s", status.Path)
		packed++
	}
	lock.Plugins[name] = locked
	if writeErr := lock.Write(root); writeErr != nil {
		return writeErr
	}
	index, err := pluginregistry.BuildIndex(filepath.Join(root, registryDir))
	if err != nil {
		return err
	}
	raw, err := index.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, registryDir, pluginregistry.IndexFile), raw, 0o644); err != nil {
		return err
	}
	c.output.Success("Packed %d edited files into registry/plugins/%s and regenerated the index", packed, name)
	c.output.Line("New files and contract changes are not packed: add them under registry/plugins/%s/files by hand.", name)
	c.output.Line("Bump the version in plugin.json when the change is user-visible.")
	return nil
}
