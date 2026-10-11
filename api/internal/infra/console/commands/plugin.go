package commands

import (
	"fmt"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/console"
	"github.com/fastagi/fastbuild/api/internal/infra/plugin"
)

// PluginListCommand lists external CLI commands: executables named fastbuild-<name> on PATH.
// Registry plugins, which add source to the project, are managed by the plugin:* commands.
type PluginListCommand struct {
	output *console.Output
}

func NewPluginListCommand() *PluginListCommand {
	return &PluginListCommand{
		output: console.NewOutput(),
	}
}

func (c *PluginListCommand) Name() string {
	return "command:list"
}

func (c *PluginListCommand) Description() string {
	return "List external CLI commands (fastbuild-<name> executables on PATH)"
}

func (c *PluginListCommand) Usage() string {
	return "fastbuild command:list"
}

func (c *PluginListCommand) Run(args []string) error {
	registry := plugin.Global()
	plugins := registry.List()

	if len(plugins) == 0 {
		c.output.Info("No external commands installed")
		c.output.Info("")
		c.output.Info("An external command is an executable named fastbuild-<name> on PATH that answers --version.")
		return nil
	}

	c.output.Success(fmt.Sprintf("Found %d external command(s):", len(plugins)))
	c.output.Info("")

	// Print table header
	c.output.Info(fmt.Sprintf("%-15s %-15s %s", "NAME", "VERSION", "BINARY"))
	c.output.Info(strings.Repeat("-", 70))

	// Print plugins
	for _, p := range plugins {
		version := p.Version
		if version == "" {
			version = "unknown"
		}
		c.output.Info(fmt.Sprintf("%-15s %-15s %s", p.Name, version, p.Binary))
	}

	c.output.Info("")
	c.output.Info("Usage: fastbuild <plugin-name> <command> [args...]")
	c.output.Info("Example: fastbuild ai docs user")

	return nil
}
