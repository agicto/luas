package commands

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/bootstrap"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/console"
	"github.com/zgiai/zgo/internal/wiring"
	"github.com/zgiai/zgo/routes"
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
func (c *ServeCommand) Usage() string       { return "serve [--port=8080]" }

func (c *ServeCommand) Run(args []string) error {
	// Initialize logger
	bootstrap.InitLogger()

	// Initialize application via Wire DI
	application, err := wiring.InitApplication()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	cfg := application.Config

	// Parse port override from args
	for i, arg := range args {
		if arg == "--port" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &cfg.Server.Port)
		}
	}
	kernel := bootstrap.NewHttpKernel(application)
	kernel.Handle()
	return nil
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
	c.output.Info("ZGO v%s", c.version)
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
