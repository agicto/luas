package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ProjectMapCommand prints an AI-readable map of the Luas workspace.
type ProjectMapCommand struct{}

type projectMap struct {
	Root      string            `json:"root"`
	API       apiMap            `json:"api"`
	Web       webMap            `json:"web"`
	Contracts []contractMapItem `json:"contracts"`
}

type apiMap struct {
	Modules    []apiModuleMapItem `json:"modules"`
	Migrations []string           `json:"migrations"`
	Routes     []routeMapItem     `json:"routes"`
}

type webMap struct {
	Features []string `json:"features"`
}

type apiModuleMapItem struct {
	Name     string             `json:"name"`
	Path     string             `json:"path"`
	Files    []string           `json:"files"`
	Manifest starterManifestMap `json:"manifest"`
}

type starterManifestMap struct {
	Migrations []string `json:"migrations"`
	Seeders    []string `json:"seeders"`
}

type contractMapItem struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	BasePath  string   `json:"base_path"`
	Endpoints []string `json:"endpoints"`
}

type routeMapItem struct {
	Module string `json:"module"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

var (
	starterMigrationNamePattern = regexp.MustCompile(`StarterMigration\s*\{\s*Name:\s*"([^"]+)"`)
	starterSeederNamePattern    = regexp.MustCompile(`StarterSeeder\s*\{\s*Name:\s*"([^"]+)"`)
	routeCallPattern            = regexp.MustCompile(`\.((?:GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|Group))\("([^"]*)"`)
	routeNamePattern            = regexp.MustCompile(`\.Name\("([^"]+)"\)`)
	contractBasePathPattern     = regexp.MustCompile("Base path:\\s*`([^`]+)`")
	contractEndpointPattern     = regexp.MustCompile("`(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\\s+([^`]+)`")
)

// NewProjectMapCommand creates the project map command.
func NewProjectMapCommand() *ProjectMapCommand {
	return &ProjectMapCommand{}
}

func (c *ProjectMapCommand) Name() string {
	return "map"
}

func (c *ProjectMapCommand) Description() string {
	return "Print an AI-readable project map"
}

func (c *ProjectMapCommand) Usage() string {
	return "luas map [--json]"
}

func (c *ProjectMapCommand) SuppressSuccessOutput() bool {
	return true
}

func (c *ProjectMapCommand) Run(args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	project, err := collectProjectMap(root)
	if err != nil {
		return err
	}

	if hasFlag(args, "--json") {
		encoded, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	fmt.Print(renderProjectMapMarkdown(project))
	return nil
}

func findWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if exists(filepath.Join(dir, "api", "go.mod")) && exists(filepath.Join(dir, "contracts")) {
			return dir, nil
		}
		if filepath.Base(dir) == "api" && exists(filepath.Join(dir, "go.mod")) {
			parent := filepath.Dir(dir)
			if exists(filepath.Join(parent, "contracts")) {
				return parent, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("could not locate Luas workspace root")
}

func collectProjectMap(root string) (*projectMap, error) {
	modules, err := listAPIModules(root)
	if err != nil {
		return nil, err
	}
	features, err := listDirs(filepath.Join(root, "web", "src", "features"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	migrations, err := listFiles(filepath.Join(root, "api", "database", "migrations"), ".go")
	if err != nil {
		return nil, err
	}
	routes, err := listRoutes(root)
	if err != nil {
		return nil, err
	}
	contracts, err := listContracts(root)
	if err != nil {
		return nil, err
	}

	return &projectMap{
		Root: root,
		API: apiMap{
			Modules:    modules,
			Migrations: migrations,
			Routes:     routes,
		},
		Web: webMap{
			Features: features,
		},
		Contracts: contracts,
	}, nil
}

func listDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names, nil
}

func listAPIModules(root string) ([]apiModuleMapItem, error) {
	dir := filepath.Join(root, "api", "internal", "modules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	modules := make([]apiModuleMapItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		moduleDir := filepath.Join(dir, entry.Name())
		files, err := listGoFilenames(moduleDir)
		if err != nil {
			return nil, err
		}
		modules = append(modules, apiModuleMapItem{
			Name:  entry.Name(),
			Path:  filepath.ToSlash(filepath.Join("api", "internal", "modules", entry.Name())),
			Files: files,
			Manifest: starterManifestMap{
				Migrations: starterManifestNames(filepath.Join(moduleDir, "provider.go"), starterMigrationNamePattern),
				Seeders:    starterManifestNames(filepath.Join(moduleDir, "provider.go"), starterSeederNamePattern),
			},
		})
	}

	slices.SortFunc(modules, func(a, b apiModuleMapItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return modules, nil
}

func listGoFilenames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		files = append(files, entry.Name())
	}
	slices.Sort(files)
	return files, nil
}

func listFiles(dir string, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) || entry.Name() == "migrations.go" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), suffix))
	}
	slices.Sort(names)
	return names, nil
}

func listContracts(root string) ([]contractMapItem, error) {
	dir := filepath.Join(root, "contracts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	items := make([]contractMapItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(dir, entry.Name())
		detail := inspectContract(path)
		items = append(items, contractMapItem{
			Name:      name,
			Path:      filepath.ToSlash(filepath.Join("contracts", entry.Name())),
			Title:     detail.Title,
			BasePath:  detail.BasePath,
			Endpoints: detail.Endpoints,
		})
	}
	slices.SortFunc(items, func(a, b contractMapItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}

type contractDetail struct {
	Title     string
	BasePath  string
	Endpoints []string
}

func inspectContract(path string) contractDetail {
	content, err := os.ReadFile(path)
	if err != nil {
		return contractDetail{}
	}

	text := string(content)
	detail := contractDetail{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			detail.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			break
		}
	}
	if match := contractBasePathPattern.FindStringSubmatch(text); len(match) == 2 {
		detail.BasePath = match[1]
	}
	for _, match := range contractEndpointPattern.FindAllStringSubmatch(text, -1) {
		if len(match) == 3 {
			detail.Endpoints = append(detail.Endpoints, match[1]+" "+match[2])
		}
	}
	return detail
}

func listRoutes(root string) ([]routeMapItem, error) {
	dir := filepath.Join(root, "api", "internal", "modules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	routes := make([]routeMapItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "routes.go")
		content, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		routes = append(routes, extractRoutes(entry.Name(), string(content))...)
	}

	slices.SortFunc(routes, func(a, b routeMapItem) int {
		if a.Module != b.Module {
			return strings.Compare(a.Module, b.Module)
		}
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		return strings.Compare(a.Method, b.Method)
	})
	return routes, nil
}

func extractRoutes(module string, content string) []routeMapItem {
	lines := strings.Split(content, "\n")
	routes := make([]routeMapItem, 0)
	prefixStack := []string{""}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if match := routeCallPattern.FindStringSubmatch(trimmed); len(match) == 3 {
			method := match[1]
			path := match[2]
			if method == "Group" {
				prefixStack = append(prefixStack, joinRoutePath(prefixStack[len(prefixStack)-1], path))
				continue
			}

			route := routeMapItem{
				Module: module,
				Method: method,
				Path:   joinRoutePath(prefixStack[len(prefixStack)-1], path),
			}
			if nameMatch := routeNamePattern.FindStringSubmatch(trimmed); len(nameMatch) == 2 {
				route.Name = nameMatch[1]
			}
			routes = append(routes, route)
		}
		if strings.HasPrefix(trimmed, "})") && len(prefixStack) > 1 {
			prefixStack = prefixStack[:len(prefixStack)-1]
		}
	}

	return routes
}

func joinRoutePath(prefix string, path string) string {
	joined := strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
	joined = strings.TrimRight(joined, "/")
	if joined == "" {
		return "/"
	}
	return joined
}

func starterManifestNames(path string, pattern *regexp.Regexp) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	matches := pattern.FindAllStringSubmatch(string(content), -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		names = append(names, match[1])
	}
	slices.Sort(names)
	return names
}

func renderProjectMapMarkdown(project *projectMap) string {
	var b strings.Builder
	b.WriteString("# Luas Project Map\n\n")
	b.WriteString("Root: `")
	b.WriteString(project.Root)
	b.WriteString("`\n\n")

	writeAPIModulesMarkdown(&b, project.API.Modules)
	writeMarkdownList(&b, "API Migrations", project.API.Migrations)
	writeRoutesMarkdown(&b, project.API.Routes)
	writeMarkdownList(&b, "Web Features", project.Web.Features)

	b.WriteString("## Contracts\n\n")
	if len(project.Contracts) == 0 {
		b.WriteString("- none\n\n")
	} else {
		for _, item := range project.Contracts {
			b.WriteString("- `")
			b.WriteString(item.Name)
			b.WriteString("` -> `")
			b.WriteString(item.Path)
			b.WriteString("`\n")
			if item.BasePath != "" {
				b.WriteString("  - base: `")
				b.WriteString(item.BasePath)
				b.WriteString("`\n")
			}
			for _, endpoint := range item.Endpoints {
				b.WriteString("  - `")
				b.WriteString(endpoint)
				b.WriteString("`\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func writeAPIModulesMarkdown(b *strings.Builder, modules []apiModuleMapItem) {
	b.WriteString("## API Modules\n\n")
	if len(modules) == 0 {
		b.WriteString("- none\n\n")
		return
	}
	for _, module := range modules {
		b.WriteString("- `")
		b.WriteString(module.Name)
		b.WriteString("` -> `")
		b.WriteString(module.Path)
		b.WriteString("`")
		if len(module.Manifest.Migrations) > 0 || len(module.Manifest.Seeders) > 0 {
			b.WriteString(" (")
			if len(module.Manifest.Migrations) > 0 {
				b.WriteString("migrations: ")
				b.WriteString(strings.Join(module.Manifest.Migrations, ", "))
			}
			if len(module.Manifest.Migrations) > 0 && len(module.Manifest.Seeders) > 0 {
				b.WriteString("; ")
			}
			if len(module.Manifest.Seeders) > 0 {
				b.WriteString("seeders: ")
				b.WriteString(strings.Join(module.Manifest.Seeders, ", "))
			}
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func writeRoutesMarkdown(b *strings.Builder, routes []routeMapItem) {
	b.WriteString("## API Routes\n\n")
	if len(routes) == 0 {
		b.WriteString("- none\n\n")
		return
	}
	for _, route := range routes {
		b.WriteString("- `")
		b.WriteString(route.Method)
		b.WriteString(" ")
		b.WriteString(route.Path)
		b.WriteString("`")
		if route.Name != "" {
			b.WriteString(" -> `")
			b.WriteString(route.Name)
			b.WriteString("`")
		}
		b.WriteString(" (")
		b.WriteString(route.Module)
		b.WriteString(")\n")
	}
	b.WriteString("\n")
}

func writeMarkdownList(b *strings.Builder, title string, items []string) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	if len(items) == 0 {
		b.WriteString("- none\n\n")
		return
	}
	for _, item := range items {
		b.WriteString("- `")
		b.WriteString(item)
		b.WriteString("`\n")
	}
	b.WriteString("\n")
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
