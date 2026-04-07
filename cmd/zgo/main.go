package main

import (
	"os"
	"strings"

	"github.com/zgiai/zgo/internal/infra/console"
	"github.com/zgiai/zgo/internal/infra/console/commands"
	"github.com/zgiai/zgo/internal/infra/plugin"
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
			os.Setenv("ZGO_ENV_FILE", strings.TrimPrefix(arg, "--env-file="))
		} else if arg == "--env-file" && i+1 < len(os.Args) {
			os.Setenv("ZGO_ENV_FILE", os.Args[i+1])
		}
	}

	// Initialize Console Application
	cli := console.New("zgo", Version)

	// Register Commands
	registerCommands(cli)

	// Check if first argument is a plugin command
	if len(os.Args) > 1 && isPluginCommand(os.Args[1]) {
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

func registerCommands(app *console.Application) {
	// Register make commands
	app.Register(commands.NewMakeModelCommand())
	app.Register(commands.NewMakeServiceCommand())
	app.Register(commands.NewMakeHandlerCommand())
	app.Register(commands.NewMakeRepositoryCommand())
	app.Register(commands.NewMakeSeederCommand())
	app.Register(commands.NewMakeMigrationCommand())
	app.Register(commands.NewMakeModuleCommand())

	// Register database migration commands (new Migrator-based)
	dbMigrate := commands.NewMigrateCommand()
	app.Register(dbMigrate)
	app.RegisterAs("migrate", dbMigrate)

	dbFresh := commands.NewFreshCommand()
	app.Register(dbFresh)
	app.RegisterAs("migrate:fresh", dbFresh)

	dbRollback := commands.NewRollbackCommand()
	app.Register(dbRollback)
	app.RegisterAs("migrate:rollback", dbRollback)

	dbReset := commands.NewResetCommand()
	app.Register(dbReset)
	app.RegisterAs("migrate:reset", dbReset)

	dbStatus := commands.NewStatusCommand()
	app.Register(dbStatus)
	app.RegisterAs("migrate:status", dbStatus)

	// Register seed command
	dbSeed := commands.NewDBSeedCommand()
	app.Register(dbSeed)
	app.RegisterAs("seed", dbSeed)

	// Register other commands
	app.Register(commands.NewAIChatCommand())
	app.Register(commands.NewServeCommand())
	app.Register(commands.NewEnvCommand())
	app.Register(commands.NewVersionCommand(Version))
	app.Register(commands.NewRouteListCommand())

	// Register plugin commands
	app.Register(commands.NewPluginListCommand())
}

// isPluginCommand checks if a command is a plugin command
func isPluginCommand(cmd string) bool {
	// Skip if it's a known core command
	coreCommands := map[string]bool{
		"make:model":       true,
		"make:service":     true,
		"make:handler":     true,
		"make:repository":  true,
		"make:seeder":      true,
		"make:migration":   true,
		"make:module":      true,
		"migrate":          true,
		"migrate:fresh":    true,
		"migrate:rollback": true,
		"migrate:reset":    true,
		"migrate:status":   true,
		"db:migrate":       true,
		"db:fresh":         true,
		"db:rollback":      true,
		"db:reset":         true,
		"db:status":        true,
		"db:seed":          true,
		"seed":             true,
		"ai:chat":          true,
		"serve":            true,
		"env":              true,
		"version":          true,
		"route:list":       true,
		"plugin:list":      true,
		"help":             true,
	}

	if coreCommands[cmd] {
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
