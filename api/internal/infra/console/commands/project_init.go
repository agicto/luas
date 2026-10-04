package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/starter"
)

// The scaffold identity is assembled from parts so project:init never rewrites its own search
// strings, which keeps this command and its tests valid in an initialized project.
const (
	scaffoldSlug       = "lu" + "as"
	scaffoldModulePath = "github.com/zgiai/" + scaffoldSlug + "/api"
	scaffoldRepository = "https://github.com/zgiai/" + scaffoldSlug
	projectMarkerFile  = ".luas-project.json"
)

var (
	projectSlugPattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}[a-z0-9]$`)
	projectModulePattern = regexp.MustCompile(`^[a-z0-9.-]+\.[a-z]{2,}(/[A-Za-z0-9._~-]+)+$`)
)

// ProjectOptions describes the downstream identity written by project:init.
type ProjectOptions struct {
	Name       string
	Slug       string
	Module     string
	Repository string
	Starters   []string
}

// ProjectMarker records where a downstream project came from, so upgrades can be compared against
// the scaffold version it started from.
type ProjectMarker struct {
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Module          string    `json:"module"`
	Repository      string    `json:"repository"`
	Starters        []string  `json:"starters"`
	ScaffoldVersion string    `json:"scaffold_version"`
	InitializedAt   time.Time `json:"initialized_at"`
}

// ProjectInitCommand turns a fresh copy of the scaffold into a named downstream project.
type ProjectInitCommand struct {
	output *console.Output
}

func NewProjectInitCommand() *ProjectInitCommand {
	return &ProjectInitCommand{output: console.NewOutput()}
}

func (c *ProjectInitCommand) Name() string { return "project:init" }
func (c *ProjectInitCommand) Description() string {
	return "Rename a fresh scaffold copy into a downstream project and select its starters"
}
func (c *ProjectInitCommand) Usage() string {
	return `project:init --name="Acme Platform" --slug=acme --module=github.com/acme/platform/api ` +
		`[--repository=https://github.com/acme/platform] [--starters=organization,permission]`
}

func (c *ProjectInitCommand) Run(args []string) error {
	options := ProjectOptions{}
	options.Name, _ = flagValue(args, "name")
	options.Slug, _ = flagValue(args, "slug")
	options.Module, _ = flagValue(args, "module")
	options.Repository, _ = flagValue(args, "repository")
	if raw, ok := flagValue(args, "starters"); ok {
		options.Starters = splitList(raw)
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	marker, changed, err := InitProject(root, options, scaffoldVersion(root), time.Now().UTC())
	if err != nil {
		return err
	}
	c.output.Success("Initialized %s (%s) from Luas %s", marker.Name, marker.Module, marker.ScaffoldVersion)
	c.output.Line("Rewrote %d files. Next:", changed)
	c.output.Line("  1. cd api && go mod tidy && go build ./... && cd .. && make check")
	c.output.Line("  2. make dev, then sign in with the seeded accounts")
	c.output.Line("  3. Remove examples and unused starters: .agents/skills/downstream-app-extraction/SKILL.md")
	return nil
}

// InitProject rewrites the scaffold identity in every tracked text file under root and writes the
// project marker. It refuses to run twice or on invalid input, and returns the rewritten file count.
func InitProject(root string, options ProjectOptions, version string, now time.Time) (*ProjectMarker, int, error) {
	options, err := normalizeProjectOptions(options)
	if err != nil {
		return nil, 0, err
	}
	if _, statErr := os.Stat(filepath.Join(root, projectMarkerFile)); statErr == nil {
		return nil, 0, fmt.Errorf("%s exists: this project is already initialized", projectMarkerFile)
	}
	files, err := trackedFiles(root)
	if err != nil {
		return nil, 0, err
	}

	replacer := strings.NewReplacer(
		scaffoldModulePath, options.Module,
		scaffoldRepository, options.Repository,
		`"name": "`+scaffoldSlug+`-web"`, `"name": "`+options.Slug+`-web"`,
		`"name": "`+scaffoldSlug+`-admin"`, `"name": "`+options.Slug+`-admin"`,
		`"name": "@`+scaffoldSlug+`/contracts"`, `"name": "@`+options.Slug+`/contracts"`,
	)
	fileRewrites := map[string]func(string) string{
		"api/.env.example":   setLine("APP_NAME=", options.Name),
		"admin/.env.example": setLine("VITE_APP_NAME=", options.Name),
		"compose.dev.yaml":   setLine("name: ", options.Slug+"-dev"),
	}
	if options.Starters != nil {
		web, admin := browserFeatures(options.Starters)
		fileRewrites["api/.env.example"] = chain(fileRewrites["api/.env.example"],
			setLine("OPTIONAL_STARTERS=", strings.Join(options.Starters, ",")))
		fileRewrites["web/.env.example"] = setLine("NEXT_PUBLIC_OPTIONAL_FEATURES=", strings.Join(web, ","))
		fileRewrites["admin/.env.example"] = chain(fileRewrites["admin/.env.example"],
			setLine("VITE_OPTIONAL_FEATURES=", strings.Join(admin, ",")))
		fileRewrites["scripts/dev.sh"] = chain(
			setShellDefault("LUAS_DEV_STARTERS", strings.Join(options.Starters, ",")),
			setShellDefault("LUAS_DEV_WEB_FEATURES", strings.Join(web, ",")),
			setShellDefault("LUAS_DEV_ADMIN_FEATURES", strings.Join(admin, ",")),
		)
	}

	changed := 0
	for _, relative := range files {
		path := filepath.Join(root, relative)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, changed, readErr
		}
		if bytes.IndexByte(content, 0) >= 0 {
			continue // binary
		}
		updated := replacer.Replace(string(content))
		if rewrite := fileRewrites[filepath.ToSlash(relative)]; rewrite != nil {
			updated = rewrite(updated)
		}
		if updated == string(content) {
			continue
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, changed, statErr
		}
		if writeErr := os.WriteFile(path, []byte(updated), info.Mode().Perm()); writeErr != nil {
			return nil, changed, writeErr
		}
		changed++
	}

	marker := &ProjectMarker{
		Name:            options.Name,
		Slug:            options.Slug,
		Module:          options.Module,
		Repository:      options.Repository,
		Starters:        options.Starters,
		ScaffoldVersion: version,
		InitializedAt:   now,
	}
	encoded, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return nil, changed, err
	}
	if err := os.WriteFile(filepath.Join(root, projectMarkerFile), append(encoded, '\n'), 0o644); err != nil {
		return nil, changed, err
	}
	return marker, changed, nil
}

func normalizeProjectOptions(options ProjectOptions) (ProjectOptions, error) {
	options.Name = strings.TrimSpace(options.Name)
	options.Slug = strings.TrimSpace(options.Slug)
	options.Module = strings.TrimSuffix(strings.TrimSpace(options.Module), "/")
	options.Repository = strings.TrimSuffix(strings.TrimSpace(options.Repository), "/")
	var problems []string
	if options.Name == "" || len([]rune(options.Name)) > 60 || strings.ContainsAny(options.Name, "\n\r=\"") {
		problems = append(problems, "--name must be 1-60 characters without quotes, '=' or newlines")
	}
	if !projectSlugPattern.MatchString(options.Slug) {
		problems = append(problems, "--slug must be 3-32 lowercase letters, digits, or hyphens and start with a letter")
	}
	if !projectModulePattern.MatchString(options.Module) || options.Module == scaffoldModulePath {
		problems = append(problems, "--module must be a new Go module path such as github.com/acme/platform/api")
	}
	if options.Repository == "" && strings.HasPrefix(options.Module, "github.com/") {
		options.Repository = "https://" + strings.TrimSuffix(options.Module, "/api")
	}
	if options.Repository != "" && !strings.HasPrefix(options.Repository, "https://") {
		problems = append(problems, "--repository must be an https URL")
	}
	if options.Repository == "" {
		problems = append(problems, "--repository is required when --module is not on github.com")
	}
	if options.Starters != nil {
		cfg := &config.Config{Starters: config.StarterConfig{Optional: options.Starters}}
		if _, err := starter.ConfiguredManifests(cfg, nil); err != nil {
			problems = append(problems, "--starters: "+err.Error())
		}
	}
	if len(problems) > 0 {
		return options, errors.New(strings.Join(problems, "; "))
	}
	return options, nil
}

// browserFeatures derives the Web and Admin feature selections that match the API starters.
func browserFeatures(starters []string) (web, admin []string) {
	for _, name := range starters {
		if name != config.StarterOperator {
			web = append(web, name)
		}
	}
	if slices.Contains(starters, config.StarterOperator) {
		admin = append(admin, config.StarterOperator)
		for _, name := range []string{config.StarterOrganization, config.StarterWebhook, config.StarterNotification} {
			if slices.Contains(starters, name) {
				admin = append(admin, name)
			}
		}
	}
	return web, admin
}

func setLine(prefix, value string) func(string) string {
	return func(content string) string {
		lines := strings.Split(content, "\n")
		for index, line := range lines {
			if strings.HasPrefix(line, prefix) {
				lines[index] = prefix + value
			}
		}
		return strings.Join(lines, "\n")
	}
}

// setShellDefault rewrites `NAME="${NAME:-default}"`-style defaults in scripts/dev.sh.
func setShellDefault(variable, value string) func(string) string {
	pattern := regexp.MustCompile(`\$\{` + variable + `:-[^}]*\}`)
	return func(content string) string {
		return pattern.ReplaceAllLiteralString(content, "${"+variable+":-"+value+"}")
	}
}

func chain(rewrites ...func(string) string) func(string) string {
	return func(content string) string {
		for _, rewrite := range rewrites {
			if rewrite != nil {
				content = rewrite(content)
			}
		}
		return content
	}
}

func splitList(raw string) []string {
	values := []string{}
	for _, value := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func repositoryRoot() (string, error) {
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("project:init must run inside the project's git repository: %w", err)
	}
	root := strings.TrimSpace(string(output))
	for _, required := range []string{"api/go.mod", "contracts/openapi.yaml"} {
		if _, err := os.Stat(filepath.Join(root, required)); err != nil {
			return "", fmt.Errorf("%s is not a Luas repository: %s is missing", root, required)
		}
	}
	return root, nil
}

func trackedFiles(root string) ([]string, error) {
	command := exec.Command("git", "ls-files", "-z")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}
	var files []string
	for _, file := range strings.Split(string(output), "\x00") {
		if file != "" {
			files = append(files, file)
		}
	}
	return files, nil
}

func scaffoldVersion(root string) string {
	command := exec.Command("git", "describe", "--tags", "--always")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
