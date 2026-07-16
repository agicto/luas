package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const (
	routeCatalogKind          = "luas.route_catalog"
	routeCatalogSchemaVersion = 1
)

var routeCatalogStarterPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type routeListFormat string

const (
	routeListFormatTable routeListFormat = "table"
	routeListFormatJSON  routeListFormat = "json"
)

type routeListOptions struct {
	format routeListFormat
}

type registeredRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type registeredRouteCatalog struct {
	Kind           string            `json:"kind"`
	SchemaVersion  int               `json:"schema_version"`
	ActiveStarters []string          `json:"active_starters"`
	Routes         []registeredRoute `json:"routes"`
}

// RouteListCommand lists routes assembled by the same registration seam as the HTTP server.
type RouteListCommand struct {
	output             *console.Output
	suppressCompletion bool
}

func NewRouteListCommand() *RouteListCommand {
	return &RouteListCommand{output: console.NewOutput()}
}

func (c *RouteListCommand) Name() string        { return "route:list" }
func (c *RouteListCommand) Description() string { return "List routes in the active runtime assembly" }
func (c *RouteListCommand) Usage() string       { return "route:list [--format=table|json]" }

func (c *RouteListCommand) SuppressCompletionOutput() bool {
	return c.suppressCompletion
}

func (c *RouteListCommand) Run(args []string) error {
	c.suppressCompletion = false
	options, err := parseRouteListOptions(args)
	if err != nil {
		return err
	}
	c.suppressCompletion = options.format == routeListFormatJSON

	application, err := wiring.InitApplication()
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	previousMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(previousMode)

	engine := gin.New()
	bootstrap.RegisterHTTPRoutes(engine, application)
	catalog, err := buildRegisteredRouteCatalog(
		application.Starters.StarterNames(),
		engine.Routes(),
	)
	if err != nil {
		return fmt.Errorf("build registered route catalog: %w", err)
	}

	if options.format == routeListFormatJSON {
		return writeRegisteredRouteCatalog(os.Stdout, catalog)
	}

	c.output.Title("Registered Routes")
	activeStarters := strings.Join(catalog.ActiveStarters, ", ")
	if activeStarters == "" {
		activeStarters = "(none)"
	}
	c.output.TwoColumn("Active starters", activeStarters)
	rows := make([][]string, 0, len(catalog.Routes))
	for _, route := range catalog.Routes {
		rows = append(rows, []string{route.Method, route.Path})
	}
	c.output.Table([]string{"Method", "Path"}, rows)
	return nil
}

func parseRouteListOptions(args []string) (routeListOptions, error) {
	options := routeListOptions{format: routeListFormatTable}
	formatSet := false

	setFormat := func(raw string) error {
		if formatSet {
			return fmt.Errorf("--format may be provided only once")
		}
		formatSet = true
		switch routeListFormat(raw) {
		case routeListFormatTable, routeListFormatJSON:
			options.format = routeListFormat(raw)
			return nil
		default:
			return fmt.Errorf("--format must be table or json")
		}
	}

	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--format":
			index++
			if index >= len(args) {
				return routeListOptions{}, fmt.Errorf("--format requires a value")
			}
			if err := setFormat(args[index]); err != nil {
				return routeListOptions{}, err
			}
		case strings.HasPrefix(argument, "--format="):
			if err := setFormat(strings.TrimPrefix(argument, "--format=")); err != nil {
				return routeListOptions{}, err
			}
		case argument == "--env" || argument == "--env-file":
			index++
			if index >= len(args) {
				return routeListOptions{}, fmt.Errorf("%s requires a value", argument)
			}
		case strings.HasPrefix(argument, "--env=") || strings.HasPrefix(argument, "--env-file="):
			// Global options are resolved by cmd/luas before command dispatch.
		default:
			return routeListOptions{}, fmt.Errorf("unknown route:list option %q", argument)
		}
	}

	return options, nil
}

func buildRegisteredRouteCatalog(
	activeStarters []string,
	routes gin.RoutesInfo,
) (registeredRouteCatalog, error) {
	starters := append([]string(nil), activeStarters...)
	seenStarters := make(map[string]struct{}, len(starters))
	for _, starterName := range starters {
		if !routeCatalogStarterPattern.MatchString(starterName) {
			return registeredRouteCatalog{}, fmt.Errorf("active starter %q is not canonical", starterName)
		}
		if _, exists := seenStarters[starterName]; exists {
			return registeredRouteCatalog{}, fmt.Errorf("active starter %q is duplicated", starterName)
		}
		seenStarters[starterName] = struct{}{}
	}

	registered := make([]registeredRoute, 0, len(routes))
	seenRoutes := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		path := strings.TrimSpace(route.Path)
		if !isHTTPRouteMethod(method) {
			return registeredRouteCatalog{}, fmt.Errorf("route method %q is unsupported", route.Method)
		}
		if !strings.HasPrefix(path, "/") {
			return registeredRouteCatalog{}, fmt.Errorf("route path %q must be absolute", route.Path)
		}
		key := method + " " + path
		if _, exists := seenRoutes[key]; exists {
			return registeredRouteCatalog{}, fmt.Errorf("registered route %q is duplicated", key)
		}
		seenRoutes[key] = struct{}{}
		registered = append(registered, registeredRoute{Method: method, Path: path})
	}
	if len(registered) == 0 {
		return registeredRouteCatalog{}, fmt.Errorf("registered route catalog is empty")
	}

	sort.Slice(registered, func(i, j int) bool {
		if registered[i].Path == registered[j].Path {
			return registered[i].Method < registered[j].Method
		}
		return registered[i].Path < registered[j].Path
	})

	return registeredRouteCatalog{
		Kind:           routeCatalogKind,
		SchemaVersion:  routeCatalogSchemaVersion,
		ActiveStarters: starters,
		Routes:         registered,
	}, nil
}

func isHTTPRouteMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return true
	default:
		return false
	}
}

func writeRegisteredRouteCatalog(writer io.Writer, catalog registeredRouteCatalog) error {
	if writer == nil {
		return fmt.Errorf("route catalog writer is required")
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(catalog); err != nil {
		return fmt.Errorf("encode registered route catalog: %w", err)
	}
	return nil
}
