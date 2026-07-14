package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/starter"
	"github.com/zgiai/luas/api/internal/wiring"
	"github.com/zgiai/luas/api/routes"
)

// ServeCommand starts the HTTP server
type ServeCommand struct {
	output *console.Output
}

func NewServeCommand() *ServeCommand {
	return &ServeCommand{output: console.NewOutput()}
}

func (c *ServeCommand) Name() string        { return "serve" }
func (c *ServeCommand) Description() string { return "Start the HTTP server" }
func (c *ServeCommand) Usage() string       { return "serve [--port=8080] [--migrate]" }

func (c *ServeCommand) Run(args []string) error {
	options, err := parseServeOptions(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if options.port != 0 {
		cfg.Server.Port = options.port
	}
	if validationErr := starter.ValidateConfig(cfg); validationErr != nil {
		return fmt.Errorf("resolve starter configuration: %w", validationErr)
	}
	if runtimeErr := validateServeRuntime(options, cfg); runtimeErr != nil {
		return runtimeErr
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return fmt.Errorf("initialize logger: %w", loggerErr)
	}

	// Initialize application via Wire DI with the same typed snapshot.
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}
	if options.migrate {
		if err := bootstrap.RunRegisteredMigrations(application.Migrator); err != nil {
			return fmt.Errorf("run startup migrations: %w", err)
		}
	}

	kernel := bootstrap.NewHttpKernel(application)
	kernel.Handle()
	return nil
}

type serveOptions struct {
	port    int
	migrate bool
}

func validateServeRuntime(options serveOptions, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if options.migrate && cfg.IsProduction() {
		return fmt.Errorf("startup migrations are disabled in production; run `luas migrate --force` as a pre-deploy job")
	}
	return nil
}

func parseServeOptions(args []string) (serveOptions, error) {
	var options serveOptions
	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--migrate":
			options.migrate = true
		case argument == "--port":
			i++
			if i >= len(args) {
				return serveOptions{}, fmt.Errorf("--port requires a value")
			}
			port, err := parseServePort(args[i])
			if err != nil {
				return serveOptions{}, err
			}
			options.port = port
		case strings.HasPrefix(argument, "--port="):
			port, err := parseServePort(strings.TrimPrefix(argument, "--port="))
			if err != nil {
				return serveOptions{}, err
			}
			options.port = port
		case argument == "--env" || argument == "--env-file":
			i++
			if i >= len(args) {
				return serveOptions{}, fmt.Errorf("%s requires a value", argument)
			}
		case strings.HasPrefix(argument, "--env=") || strings.HasPrefix(argument, "--env-file="):
			// Global options are applied by cmd/luas before command dispatch.
		default:
			return serveOptions{}, fmt.Errorf("unknown serve option %q", argument)
		}
	}
	return options, nil
}

func parseServePort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("--port must be an integer between 1 and 65535")
	}
	return port, nil
}

// EnvCommand shows environment information
type EnvCommand struct {
	output *console.Output
}

func NewEnvCommand() *EnvCommand {
	return &EnvCommand{output: console.NewOutput()}
}

func (c *EnvCommand) Name() string        { return "env" }
func (c *EnvCommand) Description() string { return "Display the current environment" }
func (c *EnvCommand) Usage() string       { return "env" }

func (c *EnvCommand) Run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	c.output.Title("Environment Information")

	c.output.TwoColumn("Environment", cfg.Server.Mode)
	optionalStarters := strings.Join(cfg.Starters.Optional, ", ")
	if optionalStarters == "" {
		optionalStarters = "(none)"
	}
	c.output.TwoColumn("Optional Starters", optionalStarters)
	c.output.TwoColumn("Server Port", fmt.Sprintf("%d", cfg.Server.Port))
	c.output.TwoColumn("Database Enabled", fmt.Sprintf("%v", cfg.Database.Enabled))
	if cfg.Database.Enabled {
		c.output.TwoColumn("Database Host", cfg.Database.Host)
		c.output.TwoColumn("Database Name", cfg.Database.DBName())
	}

	return nil
}

// VersionCommand shows version information
type VersionCommand struct {
	output  *console.Output
	version string
}

func NewVersionCommand(version string) *VersionCommand {
	return &VersionCommand{output: console.NewOutput(), version: version}
}

func (c *VersionCommand) Name() string        { return "version" }
func (c *VersionCommand) Description() string { return "Display application version" }
func (c *VersionCommand) Usage() string       { return "version" }

func (c *VersionCommand) Run(args []string) error {
	c.output.Info("Luas v%s", c.version)
	return nil
}

// RouteListCommand lists all registered routes
type RouteListCommand struct {
	output *console.Output
}

func NewRouteListCommand() *RouteListCommand {
	return &RouteListCommand{output: console.NewOutput()}
}

func (c *RouteListCommand) Name() string        { return "route:list" }
func (c *RouteListCommand) Description() string { return "List all registered routes" }
func (c *RouteListCommand) Usage() string       { return "route:list" }

func (c *RouteListCommand) Run(args []string) error {
	gin.SetMode(gin.ReleaseMode)

	// Initialize application via Wire DI
	application, err := wiring.InitApplication()
	if err != nil {
		return fmt.Errorf("failed to init application: %w", err)
	}

	r := gin.New()
	routes.Setup(r, application.Starters)

	c.output.Title("Registered Routes")

	headers := []string{"Method", "Path", "Handler"}
	rows := make([][]string, 0)

	for _, route := range r.Routes() {
		rows = append(rows, []string{route.Method, route.Path, route.Handler})
	}

	c.output.Table(headers, rows)
	return nil
}
