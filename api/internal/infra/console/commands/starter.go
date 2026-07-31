package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/starter"
)

const (
	starterCatalogKind          = "luas.starter_catalog"
	starterCatalogSchemaVersion = 1
)

type starterListFormat string

const (
	starterListFormatTable starterListFormat = "table"
	starterListFormatJSON  starterListFormat = "json"
)

type starterCatalogEntry struct {
	Name           string   `json:"name"`
	Mode           string   `json:"mode"`
	Active         bool     `json:"active"`
	Dependencies   []string `json:"dependencies"`
	MigrationNames []string `json:"migration_names"`
	SeederNames    []string `json:"seeder_names"`
}

type starterCatalogDocument struct {
	Kind              string                `json:"kind"`
	SchemaVersion     int                   `json:"schema_version"`
	EnvironmentFile   string                `json:"environment_file"`
	EnvironmentExists bool                  `json:"environment_exists"`
	OptionalSelection []string              `json:"optional_selection"`
	Starters          []starterCatalogEntry `json:"starters"`
}

type starterCommandOptions struct {
	envFile string
	format  starterListFormat
	dryRun  bool
	cascade bool
	names   []string
}

type StarterListCommand struct {
	output             *console.Output
	suppressCompletion bool
}

type StarterCheckCommand struct {
	output *console.Output
}

type StarterEnableCommand struct {
	output *console.Output
}

type StarterDisableCommand struct {
	output *console.Output
}

func NewStarterListCommand() *StarterListCommand {
	return &StarterListCommand{output: console.NewOutput()}
}

func NewStarterCheckCommand() *StarterCheckCommand {
	return &StarterCheckCommand{output: console.NewOutput()}
}

func NewStarterEnableCommand() *StarterEnableCommand {
	return &StarterEnableCommand{output: console.NewOutput()}
}

func NewStarterDisableCommand() *StarterDisableCommand {
	return &StarterDisableCommand{output: console.NewOutput()}
}

func (c *StarterListCommand) Name() string { return "starter:list" }
func (c *StarterListCommand) Description() string {
	return "List available starters and the current selection"
}
func (c *StarterListCommand) Usage() string {
	return "starter:list [--env-file=path] [--format=table|json]"
}
func (c *StarterListCommand) SuppressCompletionOutput() bool { return c.suppressCompletion }

func (c *StarterCheckCommand) Name() string { return "starter:check" }
func (c *StarterCheckCommand) Description() string {
	return "Validate the configured starter selection"
}
func (c *StarterCheckCommand) Usage() string { return "starter:check [--env-file=path]" }

func (c *StarterEnableCommand) Name() string { return "starter:enable" }
func (c *StarterEnableCommand) Description() string {
	return "Enable optional starters and their dependencies"
}
func (c *StarterEnableCommand) Usage() string {
	return "starter:enable <name>... [--env-file=path] [--dry-run]"
}

func (c *StarterDisableCommand) Name() string { return "starter:disable" }
func (c *StarterDisableCommand) Description() string {
	return "Disable optional starters"
}
func (c *StarterDisableCommand) Usage() string {
	return "starter:disable <name>... [--env-file=path] [--cascade] [--dry-run]"
}

func (c *StarterListCommand) Run(args []string) error {
	c.suppressCompletion = false
	options, err := parseStarterCommandOptions(args, true, false)
	if err != nil {
		return err
	}
	c.suppressCompletion = options.format == starterListFormatJSON

	document, err := loadStarterCatalogDocument(options.envFile)
	if err != nil {
		return err
	}
	if options.format == starterListFormatJSON {
		return writeStarterCatalogDocument(os.Stdout, document)
	}

	c.output.Title("Starter Catalog")
	c.output.TwoColumn("Environment", document.EnvironmentFile)
	rows := make([][]string, 0, len(document.Starters))
	for _, entry := range document.Starters {
		state := "available"
		if entry.Active {
			state = "active"
		}
		dependencies := strings.Join(entry.Dependencies, ", ")
		if dependencies == "" {
			dependencies = "-"
		}
		rows = append(rows, []string{entry.Name, entry.Mode, state, dependencies})
	}
	c.output.Table([]string{"Name", "Mode", "State", "Dependencies"}, rows)
	return nil
}

func (c *StarterCheckCommand) Run(args []string) error {
	options, err := parseStarterCommandOptions(args, false, false)
	if err != nil {
		return err
	}
	document, err := loadStarterCatalogDocument(options.envFile)
	if err != nil {
		return err
	}

	c.output.Title("Starter Selection")
	c.output.TwoColumn("Environment", document.EnvironmentFile)
	selected := strings.Join(document.OptionalSelection, ", ")
	if selected == "" {
		selected = "(none)"
	}
	c.output.TwoColumn("Optional starters", selected)
	c.output.Success("Starter dependencies and names are valid")
	return nil
}

func (c *StarterEnableCommand) Run(args []string) error {
	options, err := parseStarterCommandOptions(args, false, true)
	if err != nil {
		return err
	}
	return changeStarterSelection(c.output, options, true)
}

func (c *StarterDisableCommand) Run(args []string) error {
	options, err := parseStarterCommandOptions(args, false, true)
	if err != nil {
		return err
	}
	return changeStarterSelection(c.output, options, false)
}

func parseStarterCommandOptions(args []string, allowFormat, requireNames bool) (starterCommandOptions, error) {
	options := starterCommandOptions{format: starterListFormatTable}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--env-file":
			index++
			if index >= len(args) {
				return starterCommandOptions{}, fmt.Errorf("--env-file requires a value")
			}
			options.envFile = args[index]
		case strings.HasPrefix(argument, "--env-file="):
			options.envFile = strings.TrimPrefix(argument, "--env-file=")
		case argument == "--env":
			index++
			if index >= len(args) {
				return starterCommandOptions{}, fmt.Errorf("--env requires a value")
			}
		case strings.HasPrefix(argument, "--env="):
			// The global environment option is resolved before command dispatch.
		case argument == "--format" && allowFormat:
			index++
			if index >= len(args) {
				return starterCommandOptions{}, fmt.Errorf("--format requires a value")
			}
			if err := setStarterListFormat(&options, args[index]); err != nil {
				return starterCommandOptions{}, err
			}
		case strings.HasPrefix(argument, "--format=") && allowFormat:
			if err := setStarterListFormat(&options, strings.TrimPrefix(argument, "--format=")); err != nil {
				return starterCommandOptions{}, err
			}
		case argument == "--dry-run" && requireNames:
			options.dryRun = true
		case argument == "--cascade" && requireNames:
			options.cascade = true
		case strings.HasPrefix(argument, "-"):
			return starterCommandOptions{}, fmt.Errorf("unknown starter option %q", argument)
		default:
			options.names = append(options.names, argument)
		}
	}

	if strings.TrimSpace(options.envFile) == "" {
		options.envFile = os.Getenv("LUAS_ENV_FILE")
	}
	if !requireNames && len(options.names) != 0 {
		return starterCommandOptions{}, fmt.Errorf("this command does not accept starter names")
	}
	if requireNames && len(options.names) == 0 {
		return starterCommandOptions{}, fmt.Errorf("at least one starter name is required")
	}
	return options, nil
}

func setStarterListFormat(options *starterCommandOptions, raw string) error {
	switch starterListFormat(raw) {
	case starterListFormatTable, starterListFormatJSON:
		options.format = starterListFormat(raw)
		return nil
	default:
		return fmt.Errorf("--format must be table or json")
	}
}

func loadStarterCatalogDocument(explicitPath string) (starterCatalogDocument, error) {
	catalog, err := starter.AvailableCatalog()
	if err != nil {
		return starterCatalogDocument{}, fmt.Errorf("build starter catalog: %w", err)
	}
	environment, err := loadStarterEnvironment(explicitPath)
	if err != nil {
		return starterCatalogDocument{}, err
	}
	selection, err := parseOptionalStarterSelection(environment.content)
	if err != nil {
		return starterCatalogDocument{}, fmt.Errorf("read %s: %w", environment.path, err)
	}
	resolved, err := validateOptionalSelection(catalog, selection)
	if err != nil {
		return starterCatalogDocument{}, fmt.Errorf("validate OPTIONAL_STARTERS: %w", err)
	}
	active := make(map[string]struct{}, len(resolved))
	for _, name := range resolved {
		active[name] = struct{}{}
	}

	document := starterCatalogDocument{
		Kind:              starterCatalogKind,
		SchemaVersion:     starterCatalogSchemaVersion,
		EnvironmentFile:   environment.path,
		EnvironmentExists: environment.exists,
		OptionalSelection: resolved,
	}
	for _, entry := range catalog.Entries() {
		_, optionalActive := active[entry.Name]
		mode := "optional"
		if entry.Default {
			mode = "default"
		}
		document.Starters = append(document.Starters, starterCatalogEntry{
			Name:           entry.Name,
			Mode:           mode,
			Active:         entry.Default || optionalActive,
			Dependencies:   append([]string{}, entry.Dependencies...),
			MigrationNames: append([]string{}, entry.MigrationNames...),
			SeederNames:    append([]string{}, entry.SeederNames...),
		})
	}
	return document, nil
}

func writeStarterCatalogDocument(writer io.Writer, document starterCatalogDocument) error {
	if writer == nil {
		return fmt.Errorf("starter catalog writer is required")
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode starter catalog: %w", err)
	}
	return nil
}

type starterEnvironment struct {
	path    string
	content []byte
	exists  bool
	mode    os.FileMode
}

func loadStarterEnvironment(explicitPath string) (starterEnvironment, error) {
	path, err := resolveStarterEnvironmentPath(explicitPath)
	if err != nil {
		return starterEnvironment{}, err
	}
	if filepath.Base(path) == ".env.example" {
		return starterEnvironment{}, fmt.Errorf("refusing to modify or select .env.example; choose a runtime env file")
	}

	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return starterEnvironment{}, fmt.Errorf("environment path %s is not a regular file", path)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return starterEnvironment{}, fmt.Errorf("read environment file %s: %w", path, readErr)
		}
		return starterEnvironment{path: path, content: content, exists: true, mode: info.Mode().Perm()}, nil
	}
	if !os.IsNotExist(err) {
		return starterEnvironment{}, fmt.Errorf("inspect environment file %s: %w", path, err)
	}

	templatePath := filepath.Join(filepath.Dir(path), ".env.example")
	content, readErr := os.ReadFile(templatePath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return starterEnvironment{}, fmt.Errorf("read environment template %s: %w", templatePath, readErr)
	}
	return starterEnvironment{path: path, content: content, mode: 0o600}, nil
}

func resolveStarterEnvironmentPath(explicitPath string) (string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		return filepath.Clean(explicitPath), nil
	}
	for _, candidate := range []string{".env", "api/.env"} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	for _, candidate := range []string{".env.example", "api/.env.example"} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return filepath.Join(filepath.Dir(candidate), ".env"), nil
		}
	}
	return "", fmt.Errorf("cannot locate .env or .env.example; use --env-file")
}

func parseOptionalStarterSelection(content []byte) ([]string, error) {
	values, err := godotenv.Unmarshal(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse environment file: %w", err)
	}
	raw := strings.TrimSpace(values["OPTIONAL_STARTERS"])
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	selection := make([]string, 0, len(parts))
	for _, part := range parts {
		selection = append(selection, strings.TrimSpace(part))
	}
	return selection, nil
}

func changeStarterSelection(output *console.Output, options starterCommandOptions, enable bool) error {
	catalog, err := starter.AvailableCatalog()
	if err != nil {
		return fmt.Errorf("build starter catalog: %w", err)
	}
	environment, err := loadStarterEnvironment(options.envFile)
	if err != nil {
		return err
	}
	current, err := parseOptionalStarterSelection(environment.content)
	if err != nil {
		return fmt.Errorf("read %s: %w", environment.path, err)
	}
	if enable {
		current, err = catalog.ResolveOptional(current)
	} else {
		current, err = validateOptionalSelection(catalog, current)
	}
	if err != nil {
		return fmt.Errorf("validate current OPTIONAL_STARTERS: %w", err)
	}

	var next []string
	if enable {
		next, err = enableStarters(catalog, current, options.names)
	} else {
		next, err = disableStarters(catalog, current, options.names, options.cascade)
	}
	if err != nil {
		return err
	}
	updated := replaceEnvironmentValue(environment.content, "OPTIONAL_STARTERS", strings.Join(next, ","))

	action := "Enable"
	if !enable {
		action = "Disable"
	}
	output.Title(action + " Starters")
	output.TwoColumn("Environment", environment.path)
	output.TwoColumn("Before", printableSelection(current))
	output.TwoColumn("After", printableSelection(next))
	if options.dryRun {
		output.Info("Dry run: no files changed")
		return nil
	}
	if string(updated) == string(environment.content) && environment.exists {
		output.Info("Selection already up to date")
		return nil
	}
	if err := writeFileAtomic(environment.path, updated, environment.mode); err != nil {
		return err
	}
	output.Success("Updated OPTIONAL_STARTERS")
	return nil
}

func enableStarters(catalog *starter.Catalog, current, requested []string) ([]string, error) {
	combined := append([]string(nil), current...)
	seen := make(map[string]struct{}, len(combined)+len(requested))
	for _, name := range combined {
		seen[name] = struct{}{}
	}
	for _, name := range requested {
		if _, exists := seen[name]; !exists {
			combined = append(combined, name)
			seen[name] = struct{}{}
		}
	}
	return catalog.ResolveOptional(combined)
}

func disableStarters(catalog *starter.Catalog, current, requested []string, cascade bool) ([]string, error) {
	available := make(map[string]starter.CatalogEntry)
	for _, entry := range catalog.Entries() {
		if !entry.Default {
			available[entry.Name] = entry
		}
	}
	remove := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		if _, exists := available[name]; !exists {
			return nil, fmt.Errorf("unknown optional starter %q", name)
		}
		remove[name] = struct{}{}
	}

	changed := true
	for changed {
		changed = false
		for _, name := range current {
			if _, alreadyRemoved := remove[name]; alreadyRemoved {
				continue
			}
			for _, dependency := range available[name].Dependencies {
				if _, dependencyRemoved := remove[dependency]; dependencyRemoved {
					if !cascade {
						return nil, fmt.Errorf("cannot disable %q while %q depends on it; use --cascade", dependency, name)
					}
					remove[name] = struct{}{}
					changed = true
					break
				}
			}
		}
	}

	remaining := make([]string, 0, len(current))
	for _, name := range current {
		if _, removed := remove[name]; !removed {
			remaining = append(remaining, name)
		}
	}
	return catalog.ResolveOptional(remaining)
}

func replaceEnvironmentValue(content []byte, key, value string) []byte {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	prefix := key + "="
	exportPrefix := "export " + prefix
	replaced := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, exportPrefix) {
			lines[index] = prefix + value
			replaced = true
			break
		}
	}
	if !replaced {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, prefix+value, "")
	}
	return []byte(strings.Join(lines, "\n"))
}

func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create environment directory %s: %w", directory, err)
	}
	temporary, err := os.CreateTemp(directory, ".luas-env-*")
	if err != nil {
		return fmt.Errorf("create temporary environment file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if mode == 0 {
		mode = 0o600
	}
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary environment permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary environment file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary environment file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary environment file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace environment file %s: %w", path, err)
	}
	return nil
}

func printableSelection(selection []string) string {
	if len(selection) == 0 {
		return "(none)"
	}
	return strings.Join(selection, ", ")
}

func validateOptionalSelection(catalog *starter.Catalog, selection []string) ([]string, error) {
	selected, err := catalog.Select(selection)
	if err != nil {
		return nil, err
	}
	defaultNames := make(map[string]struct{})
	for _, entry := range catalog.Entries() {
		if entry.Default {
			defaultNames[entry.Name] = struct{}{}
		}
	}
	validated := make([]string, 0, len(selection))
	for _, manifest := range selected {
		if _, isDefault := defaultNames[manifest.Name()]; !isDefault {
			validated = append(validated, manifest.Name())
		}
	}
	return validated, nil
}
