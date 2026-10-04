package commands

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/infra/migration"
)

// MakeModelCommand creates a new model
type MakeModelCommand struct {
	output *console.Output
}

func NewMakeModelCommand() *MakeModelCommand {
	return &MakeModelCommand{output: console.NewOutput()}
}

func (c *MakeModelCommand) Name() string        { return "make:model" }
func (c *MakeModelCommand) Description() string { return "Create a new model" }
func (c *MakeModelCommand) Usage() string       { return "make:model <name>" }

func (c *MakeModelCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("model name is required")
	}

	name := args[0]
	dir, domainPath, data, err := existingModuleScaffold(name)
	if err != nil {
		return err
	}

	if err := ensureDomainScaffold(domainPath, data); err != nil {
		return err
	}

	path := filepath.Join(dir, "model.go")
	if err := generateFile(path, modelTemplate, data); err != nil {
		return err
	}

	c.output.Success("Model created: %s", path)
	return nil
}

// MakeServiceCommand creates a new service
type MakeServiceCommand struct {
	output *console.Output
}

func NewMakeServiceCommand() *MakeServiceCommand {
	return &MakeServiceCommand{output: console.NewOutput()}
}

func (c *MakeServiceCommand) Name() string        { return "make:service" }
func (c *MakeServiceCommand) Description() string { return "Create a new service" }
func (c *MakeServiceCommand) Usage() string       { return "make:service <name>" }

func (c *MakeServiceCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("service name is required")
	}

	name := args[0]
	dir, domainPath, data, err := existingModuleScaffold(name)
	if err != nil {
		return err
	}

	if err := ensureDomainScaffold(domainPath, data); err != nil {
		return err
	}

	path := filepath.Join(dir, "service.go")
	if err := generateFile(path, serviceTemplate, data); err != nil {
		return err
	}

	c.output.Success("Service created: %s", path)
	return nil
}

// MakeHandlerCommand creates a new handler
type MakeHandlerCommand struct {
	output *console.Output
}

func NewMakeHandlerCommand() *MakeHandlerCommand {
	return &MakeHandlerCommand{output: console.NewOutput()}
}

func (c *MakeHandlerCommand) Name() string        { return "make:handler" }
func (c *MakeHandlerCommand) Description() string { return "Create a new HTTP handler" }
func (c *MakeHandlerCommand) Usage() string       { return "make:handler <name>" }

func (c *MakeHandlerCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("handler name is required")
	}

	name := args[0]
	dir, domainPath, data, err := existingModuleScaffold(name)
	if err != nil {
		return err
	}

	if err := ensureDomainScaffold(domainPath, data); err != nil {
		return err
	}

	path := filepath.Join(dir, "handler.go")
	if err := generateFile(path, handlerTemplate, data); err != nil {
		return err
	}

	c.output.Success("Handler created: %s", path)
	return nil
}

// MakeRepositoryCommand creates a new repository
type MakeRepositoryCommand struct {
	output *console.Output
}

func NewMakeRepositoryCommand() *MakeRepositoryCommand {
	return &MakeRepositoryCommand{output: console.NewOutput()}
}

func (c *MakeRepositoryCommand) Name() string        { return "make:repository" }
func (c *MakeRepositoryCommand) Description() string { return "Create a new repository" }
func (c *MakeRepositoryCommand) Usage() string       { return "make:repository <name>" }

func (c *MakeRepositoryCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repository name is required")
	}

	name := args[0]
	dir, domainPath, data, err := existingModuleScaffold(name)
	if err != nil {
		return err
	}

	if err := ensureDomainScaffold(domainPath, data); err != nil {
		return err
	}

	path := filepath.Join(dir, "repository.go")
	if err := generateFile(path, repositoryTemplate, data); err != nil {
		return err
	}

	c.output.Success("Repository created: %s", path)
	return nil
}

// MakeSeederCommand creates a new seeder
type MakeSeederCommand struct {
	output *console.Output
}

func NewMakeSeederCommand() *MakeSeederCommand {
	return &MakeSeederCommand{output: console.NewOutput()}
}

func (c *MakeSeederCommand) Name() string        { return "make:seeder" }
func (c *MakeSeederCommand) Description() string { return "Create a new database seeder" }
func (c *MakeSeederCommand) Usage() string       { return "make:seeder <name>" }

func (c *MakeSeederCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("seeder name is required")
	}

	name := args[0]
	pascal := toPascalCase(name)
	snake := toSnakeCase(name)

	dir := filepath.Join("database", "seeders")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(dir, snake+"_seeder.go")
	if err := generateFile(filename, seederTemplate, map[string]string{
		"SeederName": pascal,
	}); err != nil {
		return err
	}

	c.output.Success("Seeder created: %s", filename)
	c.output.Info("Run with: ./luas db:seed")
	return nil
}

// MakeMigrationCommand creates a new migration using the migration Creator.
type MakeMigrationCommand struct {
	output *console.Output
}

// NewMakeMigrationCommand creates a new MakeMigrationCommand instance.
func NewMakeMigrationCommand() *MakeMigrationCommand {
	return &MakeMigrationCommand{output: console.NewOutput()}
}

func (c *MakeMigrationCommand) Name() string        { return "make:migration" }
func (c *MakeMigrationCommand) Description() string { return "Create a new database migration" }
func (c *MakeMigrationCommand) Usage() string {
	return "make:migration <name> [--create=table] [--table=table]"
}

func (c *MakeMigrationCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("migration name is required")
	}

	// Parse migration name (first non-flag argument)
	var name string
	var createTable string
	var modifyTable string

	for i, arg := range args {
		// Parse --create=table flag
		if val, found := strings.CutPrefix(arg, "--create="); found {
			createTable = val
			continue
		}
		if arg == "--create" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			createTable = args[i+1]
			continue
		}

		// Parse --table=table flag
		if val, found := strings.CutPrefix(arg, "--table="); found {
			modifyTable = val
			continue
		}
		if arg == "--table" && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			modifyTable = args[i+1]
			continue
		}

		// First non-flag argument is the migration name
		if !strings.HasPrefix(arg, "--") && name == "" {
			name = arg
		}
	}

	if name == "" {
		return fmt.Errorf("migration name is required")
	}

	// Use the migration Creator
	creator := migration.NewCreator("database/migrations")

	opts := migration.CreatorOptions{
		Create: createTable,
		Table:  modifyTable,
	}

	result, err := creator.Create(name, opts)
	if err != nil {
		return err
	}

	c.output.Success("Migration created: %s", result.Path)
	c.output.Info("Migration ID: %s", result.Name)

	// Show helpful hints based on migration type
	if createTable != "" {
		c.output.Info("Table: %s (create)", createTable)
	} else if modifyTable != "" {
		c.output.Info("Table: %s (modify)", modifyTable)
	}

	return nil
}

// MakeModuleCommand creates a complete module with all components
type MakeModuleCommand struct {
	output *console.Output
}

func NewMakeModuleCommand() *MakeModuleCommand {
	return &MakeModuleCommand{output: console.NewOutput()}
}

func (c *MakeModuleCommand) Name() string { return "make:module" }
func (c *MakeModuleCommand) Description() string {
	return "Create a complete module (model, service, handler, repository)"
}
func (c *MakeModuleCommand) Usage() string { return "make:module <name>" }

func (c *MakeModuleCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("module name is required")
	}

	name := args[0]
	data := moduleScaffoldData(name)
	if err := validateModuleName(data); err != nil {
		return err
	}
	snake := data["Package"]
	data["MigrationID"] = migration.GenerateTimestamp() + "_create_" + data["TableName"] + "_table"

	dir := filepath.Join("internal", "modules", snake)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("module directory already exists: %s", dir)
	}
	for _, target := range []string{dir, filepath.Join("internal", "domain"), filepath.Join("database", "migrations")} {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
	}

	files := []struct {
		path     string
		template string
	}{
		{filepath.Join("internal", "domain", snake+".go"), domainTemplate},
		{filepath.Join(dir, "model.go"), modelTemplate},
		{filepath.Join(dir, "service.go"), serviceTemplate},
		{filepath.Join(dir, "handler.go"), handlerTemplate},
		{filepath.Join(dir, "repository.go"), repositoryTemplate},
		{filepath.Join(dir, "dto.go"), dtoTemplate},
		{filepath.Join(dir, "routes.go"), routesTemplate},
		{filepath.Join(dir, "error_mappings.go"), errorMappingsTemplate},
		{filepath.Join(dir, "service_test.go"), serviceTestTemplate},
		{filepath.Join(dir, "provider.go"), providerTemplate},
		{filepath.Join("database", "migrations", data["MigrationID"]+".go"), migrationTemplate},
	}
	for _, f := range files {
		if err := generateFile(f.path, f.template, data); err != nil {
			return err
		}
		c.output.Success("Created: %s", f.path)
	}

	if fragment, written, fragmentErr := writeOpenAPIFragment(data); fragmentErr != nil {
		c.output.Warning("Could not write the OpenAPI fragment: %v", fragmentErr)
	} else if written {
		c.output.Success("Created: %s", fragment)
	}

	wired, err := wireOptionalStarter(data)
	if err != nil {
		c.output.Warning("Could not wire the starter automatically: %v", err)
	}
	for _, path := range wired {
		c.output.Success("Updated: %s", path)
	}

	c.output.Info("Optional starter '%s' created in %s. Next steps:", data["PackageName"], dir)
	if err != nil {
		c.output.Info("  - Add %s to internal/infra/config/starters.go and the starter to internal/starter/defaults.go", data["StarterConst"])
	}
	c.output.Info("  1. make wire")
	c.output.Info("  2. LUAS_UPDATE_GOLDEN_SCHEMA=1 go test ./database/migrations -run TestMigrationsProduceGoldenSchema (with LUAS_TEST_POSTGRES_DSN)")
	c.output.Info("  3. Review ../contracts/fragments/%s.yaml, then: cd ../contracts && corepack pnpm merge-fragment fragments/%s.yaml", snake, snake)
	c.output.Info("  4. Set the starter summary in provider.go, then: make starter-catalog")
	c.output.Info("  5. Enable it with OPTIONAL_STARTERS=%s and run go test ./...", data["PackageName"])
	c.output.Info("Routes require a signed-in user but no ownership: add record ownership before production use.")
	return nil
}

// reservedModuleNames would shadow identifiers used where the starter is wired in.
var reservedModuleNames = map[string]bool{
	"assembly": true, "config": true, "domain": true, "fmt": true, "migration": true,
	"response": true, "router": true, "seeders": true, "starter": true, "wire": true,
}

func validateModuleName(data map[string]string) error {
	pkg := data["PackageName"]
	if pkg == "" || !isLowerAlnum(pkg) || pkg[0] < 'a' || pkg[0] > 'z' {
		return fmt.Errorf("module name must start with a letter and contain only letters and digits")
	}
	if reservedModuleNames[pkg] {
		return fmt.Errorf("module name %q is reserved", pkg)
	}
	return nil
}

func isLowerAlnum(value string) bool {
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// wireOptionalStarter registers the generated starter in the starter vocabulary and the optional
// catalog. Each edit is a small, anchored insertion followed by gofmt; if an anchor is missing the
// file is left untouched and the caller prints manual steps.
func wireOptionalStarter(data map[string]string) ([]string, error) {
	startersPath := filepath.Join("internal", "infra", "config", "starters.go")
	defaultsPath := filepath.Join("internal", "starter", "defaults.go")
	pkg, dir, model, constant := data["PackageName"], data["Package"], data["ModelName"], data["StarterConst"]

	var updated []string
	err := editGoFile(startersPath, func(source string) (string, error) {
		if strings.Contains(source, "\t"+constant+" ") {
			return source, nil
		}
		source, insertErr := insertBefore(source, "\n)\n\n// StarterNames", "\n\t"+constant+" = \""+pkg+"\"")
		if insertErr != nil {
			return "", insertErr
		}
		return insertAfterFunc(source, "func StarterNames() []string {", "\t}\n}", "\t\t"+constant+",\n")
	})
	if err != nil {
		return updated, err
	}
	updated = append(updated, startersPath)

	err = editGoFile(defaultsPath, func(source string) (string, error) {
		importPath := "\"github.com/zgiai/luas/api/internal/modules/" + dir + "\""
		importLine := "\t" + importPath + "\n"
		if pkg != dir {
			// goimports requires an alias when the package name differs from its directory.
			importLine = "\t" + pkg + " " + importPath + "\n"
		}
		if strings.Contains(source, importLine) {
			return source, nil
		}
		source, insertErr := insertBefore(source, "\t\"github.com/zgiai/luas/api/internal/starter/assembly\"", importLine)
		if insertErr != nil {
			return "", insertErr
		}
		var editErr error
		if source, editErr = insertBefore(source, "\twire.Struct(new(Handlers), \"*\"),", "\t"+pkg+".ProviderSet,\n"); editErr != nil {
			return "", editErr
		}
		if source, editErr = insertBefore(source, "}\n\nfunc (h *Handlers) orMetadataOnly", "\t"+model+" *"+pkg+".Handler\n"); editErr != nil {
			return "", editErr
		}
		return insertAfterFunc(source, "func OptionalManifests(", "\t}\n}", "\t\t"+pkg+".NewStarterManifest(h."+model+"),\n")
	})
	if err != nil {
		return updated, err
	}
	return append(updated, defaultsPath), nil
}

func editGoFile(path string, edit func(string) (string, error)) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	edited, err := edit(string(raw))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	formatted, err := format.Source([]byte(edited))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

func insertBefore(source, anchor, text string) (string, error) {
	index := strings.Index(source, anchor)
	if index < 0 {
		return "", fmt.Errorf("anchor %q not found", anchor)
	}
	return source[:index] + text + source[index:], nil
}

// insertAfterFunc inserts text before the first closing anchor that follows a function signature.
func insertAfterFunc(source, signature, closing, text string) (string, error) {
	start := strings.Index(source, signature)
	if start < 0 {
		return "", fmt.Errorf("function %q not found", signature)
	}
	offset := strings.Index(source[start:], closing)
	if offset < 0 {
		return "", fmt.Errorf("end of %q not found", signature)
	}
	index := start + offset
	return source[:index] + text + source[index:], nil
}

// Helper functions
func generateFile(path, tmpl string, data map[string]string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}
	var rendered bytes.Buffer
	if execErr := t.Execute(&rendered, data); execErr != nil {
		return execErr
	}
	content := rendered.Bytes()
	if strings.HasSuffix(path, ".go") {
		// Generated Go must be gofmt-clean so it passes the repository lint without edits.
		if content, err = format.Source(content); err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}
	}
	return os.WriteFile(path, content, 0o644)
}

// writeOpenAPIFragment writes the contract fragment for a generated starter next to the API, when the
// repository has a contracts directory. It never overwrites an existing fragment.
func writeOpenAPIFragment(data map[string]string) (string, bool, error) {
	contracts := filepath.Join("..", "contracts")
	if stat, statErr := os.Stat(contracts); errors.Is(statErr, os.ErrNotExist) || (statErr == nil && !stat.IsDir()) {
		return "", false, nil // a standalone API tree has no shared contract to extend
	} else if statErr != nil {
		return "", false, statErr
	}
	path := filepath.Join(contracts, "fragments", data["Package"]+".yaml")
	if _, err := os.Stat(path); err == nil {
		return "", false, fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	if err := generateFile(path, openAPIFragmentTemplate, data); err != nil {
		return "", false, err
	}
	return path, true, nil
}

func moduleScaffoldData(name string) map[string]string {
	snake := toSnakeCase(name)
	pascal := toPascalCase(name)

	return map[string]string{
		"Package":         snake,
		"PackageName":     strings.ReplaceAll(snake, "_", ""),
		"StarterConst":    "Starter" + pascal,
		"ModelName":       pascal,
		"ServiceName":     pascal,
		"HandlerName":     pascal,
		"RepositoryName":  pascal,
		"TableName":       snake + "s",
		"RouteCollection": snake + "s",
	}
}

func existingModuleScaffold(name string) (string, string, map[string]string, error) {
	data := moduleScaffoldData(name)
	dir := filepath.Join("internal", "modules", data["Package"])
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		// Return a guidance-oriented error regardless of whether the stat
		// failed or the path exists but is a file; the user's next action
		// is the same.
		return "", "", nil, fmt.Errorf(
			"module %q does not exist in %s; run 'luas make:module %s' to scaffold a complete module",
			data["Package"],
			dir,
			data["ModelName"],
		)
	}

	domainPath := filepath.Join("internal", "domain", data["Package"]+".go")
	return dir, domainPath, data, nil
}

func ensureDomainScaffold(path string, data map[string]string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return generateFile(path, domainTemplate, data)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toPascalCase(s string) string {
	normalized := toSnakeCase(strings.ReplaceAll(s, "-", "_"))
	parts := strings.Split(normalized, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "")
}

// Templates
const domainTemplate = `package domain

import (
	"context"
	"time"
)

// {{.ModelName}} is the core domain entity for the {{.Package}} module.
type {{.ModelName}} struct {
	ID        uint      ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}

// {{.ModelName}}Repository defines persistence for {{.ModelName}}.
type {{.ModelName}}Repository interface {
	Create(ctx context.Context, item *{{.ModelName}}) error
	Update(ctx context.Context, item *{{.ModelName}}) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*{{.ModelName}}, error)
	FindAll(ctx context.Context, page, pageSize int) ([]*{{.ModelName}}, int64, error)
}
`

const modelTemplate = `package {{.PackageName}}

import (
	"time"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

// {{.ModelName}}PO is the persistent object for {{.ModelName}}.
type {{.ModelName}}PO struct {
	ID        uint           ` + "`gorm:\"primaryKey\"`" + `
	Name      string         ` + "`gorm:\"size:255;not null;index\"`" + `
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt ` + "`gorm:\"index\"`" + `
}

func ({{.ModelName}}PO) TableName() string {
	return "{{.TableName}}"
}

func (po *{{.ModelName}}PO) toDomain() *domain.{{.ModelName}} {
	if po == nil {
		return nil
	}

	return &domain.{{.ModelName}}{
		ID:        po.ID,
		Name:      po.Name,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

func new{{.ModelName}}PO(item *domain.{{.ModelName}}) *{{.ModelName}}PO {
	if item == nil {
		return nil
	}

	return &{{.ModelName}}PO{
		ID:        item.ID,
		Name:      item.Name,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
`

const serviceTemplate = `package {{.PackageName}}

import (
	"context"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
)

// Service defines the business interface for {{.ModelName}}.
type Service interface {
	Create(ctx context.Context, req *Create{{.ModelName}}Request) (*domain.{{.ModelName}}, error)
	Update(ctx context.Context, id uint, req *Update{{.ModelName}}Request) (*domain.{{.ModelName}}, error)
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*domain.{{.ModelName}}, error)
	List(ctx context.Context, page, pageSize int) ([]*domain.{{.ModelName}}, int64, error)
}

type service struct {
	repo domain.{{.ModelName}}Repository
}

var _ Service = (*service)(nil)

// NewService creates a new {{.Package}} service.
func NewService(repo domain.{{.ModelName}}Repository) *service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req *Create{{.ModelName}}Request) (*domain.{{.ModelName}}, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	item := &domain.{{.ModelName}}{
		Name: name,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *service) Update(ctx context.Context, id uint, req *Update{{.ModelName}}Request) (*domain.{{.ModelName}}, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		item.Name = name
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *service) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id uint) (*domain.{{.ModelName}}, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context, page, pageSize int) ([]*domain.{{.ModelName}}, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize)
}
`

const handlerTemplate = `package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler handles HTTP requests for {{.ModelName}}.
type Handler struct {
	service Service
}

var (
	_ assembly.Module      = (*Handler)(nil)
	_ assembly.RouteModule = (*Handler)(nil)
	_ assembly.ErrorModule = (*Handler)(nil)
)

// NewHandler creates a new handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the starter name.
func (h *Handler) Name() string {
	return config.{{.StarterConst}}
}

func (h *Handler) List(c *gin.Context) {
	req := pagination.FromContext(c)

	items, total, err := h.service.List(c.Request.Context(), req.GetPage(), req.GetPerPage())
	if err != nil {
		response.HandleError(c, "Failed to list {{.RouteCollection}}", err)
		return
	}

	paginator := pagination.NewPaginator(to{{.ModelName}}Responses(items), total, req.GetPage(), req.GetPerPage())
	paginator.SetPath(c.Request.URL.Path)
	response.Success(c, paginator)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, "Failed to get {{.Package}}", err)
		return
	}

	response.Success(c, to{{.ModelName}}Response(item))
}

func (h *Handler) Create(c *gin.Context) {
	var req Create{{.ModelName}}Request
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, "Failed to create {{.Package}}", err)
		return
	}

	response.Created(c, to{{.ModelName}}Response(item))
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	var req Update{{.ModelName}}Request
	if !httphandler.BindJSON(c, &req) {
		return
	}

	item, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.HandleError(c, "Failed to update {{.Package}}", err)
		return
	}

	response.Success(c, to{{.ModelName}}Response(item))
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httphandler.ParseID(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.HandleError(c, "Failed to delete {{.Package}}", err)
		return
	}

	response.NoContent(c)
}
`

const repositoryTemplate = `package {{.PackageName}}

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/domain"
)

type repository struct {
	db *gorm.DB
}

var _ domain.{{.ModelName}}Repository = (*repository)(nil)

// NewRepository creates a new repository.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, item *domain.{{.ModelName}}) error {
	po := new{{.ModelName}}PO(item)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}

	item.ID = po.ID
	item.CreatedAt = po.CreatedAt
	item.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Update(ctx context.Context, item *domain.{{.ModelName}}) error {
	po := new{{.ModelName}}PO(item)
	if err := r.db.WithContext(ctx).Save(po).Error; err != nil {
		return err
	}

	item.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&{{.ModelName}}PO{}, id).Error
}

func (r *repository) FindByID(ctx context.Context, id uint) (*domain.{{.ModelName}}, error) {
	var po {{.ModelName}}PO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *repository) FindAll(ctx context.Context, page, pageSize int) ([]*domain.{{.ModelName}}, int64, error) {
	var (
		rows  []{{.ModelName}}PO
		total int64
	)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}

	query := r.db.WithContext(ctx).Model(&{{.ModelName}}PO{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Offset((page-1)*pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.{{.ModelName}}, 0, len(rows))
	for i := range rows {
		items = append(items, rows[i].toDomain())
	}
	return items, total, nil
}
`

const dtoTemplate = `package {{.PackageName}}

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

// Create{{.ModelName}}Request represents the request to create a {{.ModelName}}.
type Create{{.ModelName}}Request struct {
	Name string ` + "`json:\"name\" binding:\"required,max=255\"`" + `
}

// Update{{.ModelName}}Request represents the request to update a {{.ModelName}}.
type Update{{.ModelName}}Request struct {
	Name *string ` + "`json:\"name,omitempty\" binding:\"omitempty,max=255\"`" + `
}

// {{.ModelName}}Response represents the API response for {{.ModelName}}.
type {{.ModelName}}Response struct {
	ID        uint      ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}

func to{{.ModelName}}Response(item *domain.{{.ModelName}}) *{{.ModelName}}Response {
	if item == nil {
		return nil
	}

	return &{{.ModelName}}Response{
		ID:        item.ID,
		Name:      item.Name,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func to{{.ModelName}}Responses(items []*domain.{{.ModelName}}) []*{{.ModelName}}Response {
	result := make([]*{{.ModelName}}Response, 0, len(items))
	for _, item := range items {
		result = append(result, to{{.ModelName}}Response(item))
	}
	return result
}
`

const seederTemplate = `package seeders

import (
	"gorm.io/gorm"
)

type {{.SeederName}}Seeder struct{}

func (s *{{.SeederName}}Seeder) Name() string {
	return "{{.SeederName}}"
}

func (s *{{.SeederName}}Seeder) Run(db *gorm.DB) error {
	// TODO: Implement seeder logic
	// Example:
	// items := []YourModel{
	//     {Field: "value"},
	// }
	// for _, item := range items {
	//     db.FirstOrCreate(&item, YourModel{Field: item.Field})
	// }

	return nil
}

func init() {
	register(&{{.SeederName}}Seeder{})
}
`

const routesTemplate = `package {{.PackageName}}

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes registers HTTP routes for the {{.Package}} starter. Every route requires a signed-in
// user; record ownership is not enforced yet and must be added before production use.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("/{{.RouteCollection}}", func(group *router.Router) {
		group.WithMiddleware("auth")
		group.GET("", h.List).Name("{{.Package}}.index")
		group.POST("", h.Create).Name("{{.Package}}.store")
		group.GET("/:id", h.Get).Name("{{.Package}}.show").WhereNumber("id")
		group.PUT("/:id", h.Update).Name("{{.Package}}.update").WhereNumber("id")
		group.DELETE("/:id", h.Delete).Name("{{.Package}}.destroy").WhereNumber("id")
	})
}
`

const serviceTestTemplate = `package {{.PackageName}}

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/zgiai/luas/api/internal/domain"
)

type mockRepository struct {
	mock.Mock
}

var _ domain.{{.ModelName}}Repository = (*mockRepository)(nil)

func (m *mockRepository) Create(ctx context.Context, item *domain.{{.ModelName}}) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *mockRepository) Update(ctx context.Context, item *domain.{{.ModelName}}) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *mockRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockRepository) FindByID(ctx context.Context, id uint) (*domain.{{.ModelName}}, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.{{.ModelName}}), args.Error(1)
}

func (m *mockRepository) FindAll(ctx context.Context, page, pageSize int) ([]*domain.{{.ModelName}}, int64, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.{{.ModelName}}), args.Get(1).(int64), args.Error(2)
}

func Test{{.ServiceName}}GetByID(t *testing.T) {
	repo := new(mockRepository)
	svc := NewService(repo)
	ctx := context.Background()

	expected := &domain.{{.ModelName}}{ID: 1, Name: "Example"}
	repo.On("FindByID", ctx, uint(1)).Return(expected, nil)

	result, err := svc.GetByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected.Name, result.Name)
	repo.AssertExpectations(t)
}
`

const providerTemplate = `package {{.PackageName}}

import (
	"github.com/google/wire"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/starter/assembly"
)

// ProviderSet is the provider set for this starter.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.{{.ModelName}}Repository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)

// NewStarterManifest describes the {{.Package}} starter: its dependencies, routes, and migrations.
func NewStarterManifest(handler *Handler) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(
		config.{{.StarterConst}},
		assembly.WithStarterSummary("TODO: describe the {{.Package}} starter in one line."),
		assembly.WithStarterDependencies(config.StarterUser, config.StarterAudit),
		assembly.WithStarterModule(handler),
		assembly.WithStarterMigrationNames("{{.MigrationID}}"),
	)
}
`

// openAPIFragmentTemplate describes the generated CRUD routes in the shared envelope and error
// conventions. Merge it with `cd contracts && corepack pnpm merge-fragment <file>`.
const openAPIFragmentTemplate = `# OpenAPI fragment generated by luas make:module for the {{.Package}} starter.
# Review it, then merge it into contracts/openapi.yaml:
#   cd contracts && corepack pnpm merge-fragment fragments/{{.Package}}.yaml
tags:
  - name: {{.ModelName}}
    description: Generated {{.Package}} starter resources owned by no one yet; add ownership before production use.
paths:
  /v1/{{.RouteCollection}}:
    get:
      operationId: list{{.ModelName}}s
      summary: List {{.RouteCollection}}
      tags:
        - {{.ModelName}}
      security:
        - bearerSession: []
      parameters:
        - $ref: '#/components/parameters/Page'
        - $ref: '#/components/parameters/PerPage'
      responses:
        '200':
          description: One page of {{.RouteCollection}}.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{.ModelName}}PageResponse'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '503':
          $ref: '#/components/responses/ServiceUnavailable'
    post:
      operationId: create{{.ModelName}}
      summary: Create a {{.Package}}
      tags:
        - {{.ModelName}}
      security:
        - bearerSession: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Create{{.ModelName}}Request'
      responses:
        '201':
          description: The created {{.Package}}.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{.ModelName}}Response'
        '400':
          $ref: '#/components/responses/InvalidInput'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '413':
          $ref: '#/components/responses/RequestTooLarge'
        '422':
          $ref: '#/components/responses/ValidationFailed'
        '503':
          $ref: '#/components/responses/ServiceUnavailable'
  /v1/{{.RouteCollection}}/{{"{"}}{{.Package}}_id{{"}"}}:
    get:
      operationId: get{{.ModelName}}
      summary: Get a {{.Package}}
      tags:
        - {{.ModelName}}
      security:
        - bearerSession: []
      parameters:
        - $ref: '#/components/parameters/{{.ModelName}}ID'
      responses:
        '200':
          description: The {{.Package}}.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{.ModelName}}Response'
        '400':
          $ref: '#/components/responses/InvalidInput'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '404':
          $ref: '#/components/responses/NotFound'
        '503':
          $ref: '#/components/responses/ServiceUnavailable'
    put:
      operationId: update{{.ModelName}}
      summary: Update a {{.Package}}
      tags:
        - {{.ModelName}}
      security:
        - bearerSession: []
      parameters:
        - $ref: '#/components/parameters/{{.ModelName}}ID'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Update{{.ModelName}}Request'
      responses:
        '200':
          description: The updated {{.Package}}.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{.ModelName}}Response'
        '400':
          $ref: '#/components/responses/InvalidInput'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '404':
          $ref: '#/components/responses/NotFound'
        '413':
          $ref: '#/components/responses/RequestTooLarge'
        '422':
          $ref: '#/components/responses/ValidationFailed'
        '503':
          $ref: '#/components/responses/ServiceUnavailable'
    delete:
      operationId: delete{{.ModelName}}
      summary: Delete a {{.Package}}
      tags:
        - {{.ModelName}}
      security:
        - bearerSession: []
      parameters:
        - $ref: '#/components/parameters/{{.ModelName}}ID'
      responses:
        '204':
          description: The {{.Package}} is deleted.
        '400':
          $ref: '#/components/responses/InvalidInput'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '404':
          $ref: '#/components/responses/NotFound'
        '503':
          $ref: '#/components/responses/ServiceUnavailable'
components:
  parameters:
    {{.ModelName}}ID:
      name: {{.Package}}_id
      in: path
      required: true
      description: Positive {{.Package}} identifier.
      schema:
        $ref: '#/components/schemas/PositiveID'
  schemas:
    {{.ModelName}}:
      type: object
      additionalProperties: false
      required:
        - id
        - name
        - created_at
        - updated_at
      properties:
        id:
          $ref: '#/components/schemas/PositiveID'
        name:
          type: string
          minLength: 1
          maxLength: 255
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    Create{{.ModelName}}Request:
      type: object
      additionalProperties: false
      required:
        - name
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 255
    Update{{.ModelName}}Request:
      type: object
      additionalProperties: false
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 255
    {{.ModelName}}Response:
      type: object
      additionalProperties: false
      required:
        - code
        - message
        - data
      properties:
        code:
          const: 0
        message:
          type: string
        data:
          $ref: '#/components/schemas/{{.ModelName}}'
    {{.ModelName}}PageResponse:
      type: object
      additionalProperties: false
      required:
        - code
        - message
        - data
        - meta
        - links
      properties:
        code:
          const: 0
        message:
          type: string
        data:
          type: array
          items:
            $ref: '#/components/schemas/{{.ModelName}}'
        meta:
          $ref: '#/components/schemas/PaginationMeta'
        links:
          $ref: '#/components/schemas/PaginationLinks'
`

const errorMappingsTemplate = `package {{.PackageName}}

import "github.com/zgiai/luas/api/pkg/response"

// RegisterErrorMappings maps this starter's own domain errors to public status and error codes.
// Shared errors such as domain.ErrNotFound and domain.ErrInvalidInput are already mapped by core.
func (h *Handler) RegisterErrorMappings(mapper *response.ErrorMapper) {
	_ = mapper
}
`

const migrationTemplate = `package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("{{.MigrationID}}", &create{{.ModelName}}Table{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type create{{.ModelName}}Table struct {
	migration.BaseMigration
}

// Up creates the {{.TableName}} table as frozen SQL; later model changes need a new migration.
func (m *create{{.ModelName}}Table) Up(db *gorm.DB) error {
	return execStatements(db,
		` + "`" + `CREATE TABLE {{.TableName}} (
		    id bigserial NOT NULL,
		    name varchar(255) NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    deleted_at timestamptz,
		    CONSTRAINT {{.TableName}}_pkey PRIMARY KEY (id)
		)` + "`" + `,
		` + "`" + `CREATE INDEX idx_{{.TableName}}_name ON {{.TableName}} (name)` + "`" + `,
		` + "`" + `CREATE INDEX idx_{{.TableName}}_deleted_at ON {{.TableName}} (deleted_at)` + "`" + `,
	)
}

// Down drops the {{.TableName}} table.
func (m *create{{.ModelName}}Table) Down(db *gorm.DB) error {
	return execStatements(db, ` + "`" + `DROP TABLE IF EXISTS {{.TableName}} CASCADE` + "`" + `)
}
`
