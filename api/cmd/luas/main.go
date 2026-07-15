package main

import (
	"os"
	"strings"

	"github.com/zgiai/luas/api/internal/bootstrap/operatorcommands"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/infra/console/commands"
	"github.com/zgiai/luas/api/internal/infra/plugin"
)

const Version = "1.0.0"

func main() {
	// Parse global flags before anything else
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--env=") {
			os.Setenv("APP_ENV", strings.TrimPrefix(arg, "--env="))
		} else if arg == "--env" && i+1 < len(os.Args) {
			os.Setenv("APP_ENV", os.Args[i+1])
		} else if strings.HasPrefix(arg, "--env-file=") {
			os.Setenv("LUAS_ENV_FILE", strings.TrimPrefix(arg, "--env-file="))
		} else if arg == "--env-file" && i+1 < len(os.Args) {
			os.Setenv("LUAS_ENV_FILE", os.Args[i+1])
		}
	}

	// Initialize Console Application
	cli := console.New("luas", Version)

	// Register command manifests
	manifests := append(commands.DefaultManifests(Version), operatorcommands.Manifest())
	commands.RegisterManifests(cli, manifests...)

	// Check if first argument is a plugin command
	if len(os.Args) > 1 && isPluginCommand(cli, os.Args[1]) {
		pluginName := os.Args[1]
		pluginArgs := os.Args[2:]

		// Execute plugin
		if err := plugin.Execute(pluginName, pluginArgs); err != nil {
			os.Exit(1)
		}
		return
	}

	// Handle Command
	if err := cli.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

// isPluginCommand checks if a command is a plugin command
func isPluginCommand(app *console.Application, cmd string) bool {
	if app != nil && app.HasCommand(cmd) {
		return false
	}

	switch cmd {
	case "help", "-h", "--help", "version", "-v", "--version", "list":
		return false
	}

	// Check if it starts with a known prefix
	if strings.HasPrefix(cmd, "make:") ||
		strings.HasPrefix(cmd, "migrate:") ||
		strings.HasPrefix(cmd, "db:") ||
		strings.HasPrefix(cmd, "route:") ||
		strings.HasPrefix(cmd, "plugin:") {
		return false
	}

	// Check if plugin exists
	return plugin.IsInstalled(cmd)
}
