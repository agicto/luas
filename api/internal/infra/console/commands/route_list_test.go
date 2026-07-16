package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseRouteListOptions(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantFormat routeListFormat
		wantError  bool
	}{
		{name: "default table", wantFormat: routeListFormatTable},
		{name: "json equals", args: []string{"--format=json"}, wantFormat: routeListFormatJSON},
		{name: "json separate", args: []string{"--format", "json"}, wantFormat: routeListFormatJSON},
		{name: "missing value", args: []string{"--format"}, wantError: true},
		{name: "unknown format", args: []string{"--format=yaml"}, wantError: true},
		{name: "unknown option", args: []string{"--output=routes.json"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseRouteListOptions(tt.args)
			if tt.wantError {
				if err == nil {
					t.Fatal("parseRouteListOptions() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRouteListOptions() error = %v", err)
			}
			if options.format != tt.wantFormat {
				t.Fatalf("format = %q, want %q", options.format, tt.wantFormat)
			}
		})
	}
}

func TestBuildRegisteredRouteCatalogIsDeterministicAndOwned(t *testing.T) {
	starters := []string{"user", "apikey", "audit"}
	routes := gin.RoutesInfo{
		{Method: "POST", Path: "/v1/login", Handler: "wrapped"},
		{Method: "GET", Path: "/health/ready", Handler: "health"},
		{Method: "GET", Path: "/", Handler: "welcome"},
	}

	catalog, err := buildRegisteredRouteCatalog(starters, routes)
	if err != nil {
		t.Fatalf("buildRegisteredRouteCatalog() error = %v", err)
	}

	starters[0] = "mutated"
	routes[0].Path = "/mutated"
	if catalog.Kind != routeCatalogKind || catalog.SchemaVersion != routeCatalogSchemaVersion {
		t.Fatalf("catalog identity = %q/%d", catalog.Kind, catalog.SchemaVersion)
	}
	if got := catalog.ActiveStarters[0]; got != "user" {
		t.Fatalf("active_starters[0] = %q, want user", got)
	}
	wantRoutes := []registeredRoute{
		{Method: "GET", Path: "/"},
		{Method: "GET", Path: "/health/ready"},
		{Method: "POST", Path: "/v1/login"},
	}
	if len(catalog.Routes) != len(wantRoutes) {
		t.Fatalf("route count = %d, want %d", len(catalog.Routes), len(wantRoutes))
	}
	for index, want := range wantRoutes {
		if catalog.Routes[index] != want {
			t.Fatalf("routes[%d] = %+v, want %+v", index, catalog.Routes[index], want)
		}
	}

	var output bytes.Buffer
	if err := writeRegisteredRouteCatalog(&output, catalog); err != nil {
		t.Fatalf("writeRegisteredRouteCatalog() error = %v", err)
	}
	var decoded registeredRouteCatalog
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, output.String())
	}
	if decoded.Kind != routeCatalogKind || len(decoded.Routes) != 3 {
		t.Fatalf("decoded catalog = %+v", decoded)
	}
}

func TestBuildRegisteredRouteCatalogRejectsAmbiguity(t *testing.T) {
	tests := []struct {
		name     string
		starters []string
		routes   gin.RoutesInfo
	}{
		{
			name:     "duplicate starter",
			starters: []string{"user", "user"},
			routes:   gin.RoutesInfo{{Method: "GET", Path: "/"}},
		},
		{
			name:     "noncanonical starter",
			starters: []string{"User"},
			routes:   gin.RoutesInfo{{Method: "GET", Path: "/"}},
		},
		{
			name:   "duplicate route",
			routes: gin.RoutesInfo{{Method: "GET", Path: "/"}, {Method: "GET", Path: "/"}},
		},
		{
			name:   "relative path",
			routes: gin.RoutesInfo{{Method: "GET", Path: "health"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := buildRegisteredRouteCatalog(tt.starters, tt.routes); err == nil {
				t.Fatal("buildRegisteredRouteCatalog() error = nil, want error")
			}
		})
	}
}
