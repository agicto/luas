package commands

import (
	"fmt"
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

// MakeContractCommand creates a new HTTP contract document.
type MakeContractCommand struct {
	output *console.Output
}

func NewMakeContractCommand() *MakeContractCommand {
	return &MakeContractCommand{output: console.NewOutput()}
}

func (c *MakeContractCommand) Name() string        { return "make:contract" }
func (c *MakeContractCommand) Description() string { return "Create a new HTTP contract document" }
func (c *MakeContractCommand) Usage() string {
	return "make:contract <name> [--resource=/v1/resources]"
}

func (c *MakeContractCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("contract name is required")
	}

	var name string
	var resourcePath string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--resource="):
			resourcePath = strings.TrimSpace(strings.TrimPrefix(arg, "--resource="))
		case arg == "--resource" && i+1 < len(args):
			resourcePath = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--"):
			return fmt.Errorf("unknown flag: %s", arg)
		case name == "":
			name = arg
		}
	}
	if name == "" {
		return fmt.Errorf("contract name is required")
	}

	data := moduleScaffoldData(name)
	if resourcePath == "" {
		resourcePath = "/v1/" + data["RouteCollection"]
	}
	data["ResourcePath"] = resourcePath

	dir := filepath.Join("..", "contracts")
	if cwd, err := os.Getwd(); err == nil && filepath.Base(cwd) != "api" {
		dir = "contracts"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, data["Package"]+".md")
	if err := generateFile(path, contractTemplate, data); err != nil {
		return err
	}

	c.output.Success("Contract created: %s", path)
	c.output.Info("Next: wire this contract to the API module and web feature service")
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
func (c *MakeModuleCommand) Usage() string {
	return "make:module <name> [--with=contract,migration,web]"
}

func (c *MakeModuleCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("module name is required")
	}

	name, requested, err := parseMakeModuleArgs(args)
	if err != nil {
		return err
	}
	snake := toSnakeCase(name)

	// Target directory: internal/modules/[name]
	dir := filepath.Join("internal", "modules", snake)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	domainDir := filepath.Join("internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return err
	}

	files := []struct {
		name     string
		template string
	}{
		{"model.go", modelTemplate},
		{"service.go", serviceTemplate},
		{"handler.go", handlerTemplate},
		{"repository.go", repositoryTemplate},
		{"dto.go", dtoTemplate},
		{"routes.go", routesTemplate},
		{"service_test.go", serviceTestTemplate},
		{"provider.go", providerTemplate},
	}

	data := moduleScaffoldData(name)

	domainPath := filepath.Join(domainDir, snake+".go")
	if err := generateFile(domainPath, domainTemplate, data); err != nil {
		return err
	}
	c.output.Success("Created: %s", domainPath)

	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := generateFile(path, f.template, data); err != nil {
			return err
		}
		c.output.Success("Created: %s", path)
	}

	if requested["contract"] {
		contractCmd := NewMakeContractCommand()
		if err := contractCmd.Run([]string{name}); err != nil {
			return err
		}
	}
	if requested["migration"] {
		migrationCmd := NewMakeMigrationCommand()
		if err := migrationCmd.Run([]string{"create_" + data["TableName"] + "_table", "--create=" + data["TableName"]}); err != nil {
			return err
		}
	}
	if requested["web"] {
		if err := createWebFeatureScaffold(data); err != nil {
			return err
		}
		c.output.Success("Web feature created: %s", webFeatureDir(data["Package"]))
	}

	c.output.Info("Module '%s' created successfully!", name)
	c.output.Info("Next steps:")
	c.output.Info("  1. Refine internal/domain/%s.go with real business fields", snake)
	c.output.Info("  2. Decide whether the module is a starter, optional starter, or example")
	c.output.Info("  3. If it becomes a default starter, add its starter manifest to internal/starter/defaults.go")
	c.output.Info("  4. Run make wire and go test ./...")
	return nil
}

func createWebFeatureScaffold(data map[string]string) error {
	featureDir := webFeatureDir(data["Package"])
	dirs := []string{
		featureDir,
		filepath.Join(featureDir, "components"),
		filepath.Join(featureDir, "hooks"),
		filepath.Join(featureDir, "server"),
		filepath.Join(featureDir, "services"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	files := []struct {
		path     string
		template string
	}{
		{filepath.Join(featureDir, "types.ts"), webFeatureTypesTemplate},
		{filepath.Join(featureDir, "services", data["Package"]+"-service.ts"), webFeatureServiceTemplate},
		{filepath.Join(featureDir, "hooks", "use-"+data["Package"]+".ts"), webFeatureHookTemplate},
		{filepath.Join(featureDir, "server", "mock-"+data["Package"]+"-store.ts"), webFeatureServerTemplate},
		{filepath.Join(featureDir, "index.ts"), webFeatureIndexTemplate},
	}
	for _, file := range files {
		if err := generateFile(file.path, file.template, data); err != nil {
			return err
		}
	}
	return nil
}

func webFeatureDir(name string) string {
	base := filepath.Join("..", "web", "src", "features")
	if cwd, err := os.Getwd(); err == nil && filepath.Base(cwd) != "api" {
		base = filepath.Join("web", "src", "features")
	}
	return filepath.Join(base, name)
}

func parseMakeModuleArgs(args []string) (string, map[string]bool, error) {
	requested := make(map[string]bool)
	var name string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--with="):
			addRequestedScaffold(strings.TrimPrefix(arg, "--with="), requested)
		case arg == "--with" && i+1 < len(args):
			addRequestedScaffold(args[i+1], requested)
			i++
		case strings.HasPrefix(arg, "--"):
			return "", nil, fmt.Errorf("unknown flag: %s", arg)
		case name == "":
			name = arg
		}
	}
	if name == "" {
		return "", nil, fmt.Errorf("module name is required")
	}
	return name, requested, nil
}

func addRequestedScaffold(value string, requested map[string]bool) {
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			requested[part] = true
		}
	}
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

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}

func moduleScaffoldData(name string) map[string]string {
	snake := toSnakeCase(name)
	pascal := toPascalCase(name)

	return map[string]string{
		"Package":         snake,
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

const modelTemplate = `package {{.Package}}

import (
	"time"

	"github.com/zgiai/luas/api/internal/domain"
	"gorm.io/gorm"
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

const serviceTemplate = `package {{.Package}}

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

const handlerTemplate = `package {{.Package}}

import (
	"github.com/gin-gonic/gin"
	"github.com/zgiai/luas/api/internal/contracts"
	httphandler "github.com/zgiai/luas/api/pkg/handler"
	"github.com/zgiai/luas/api/pkg/pagination"
	"github.com/zgiai/luas/api/pkg/response"
)

// Handler handles HTTP requests for {{.ModelName}}.
type Handler struct {
	service Service
}

var (
	_ contracts.Module      = (*Handler)(nil)
	_ contracts.RouteModule = (*Handler)(nil)
)

// NewHandler creates a new handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Name returns the module name.
func (h *Handler) Name() string {
	return "{{.Package}}"
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

const repositoryTemplate = `package {{.Package}}

import (
	"context"
	"errors"

	"github.com/zgiai/luas/api/internal/domain"
	"gorm.io/gorm"
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

const dtoTemplate = `package {{.Package}}

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

const routesTemplate = `package {{.Package}}

import "github.com/zgiai/luas/api/internal/infra/router"

// RegisterRoutes registers HTTP routes for the {{.Package}} module.
func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("/{{.RouteCollection}}", func(group *router.Router) {
		group.GET("", h.List).Name("{{.Package}}.index")
		group.POST("", h.Create).Name("{{.Package}}.store")
		group.GET("/:id", h.Get).Name("{{.Package}}.show").WhereNumber("id")
		group.PUT("/:id", h.Update).Name("{{.Package}}.update").WhereNumber("id")
		group.DELETE("/:id", h.Delete).Name("{{.Package}}.destroy").WhereNumber("id")
	})
}
`

const contractTemplate = `# {{.ModelName}} Contract

Base path: ` + "`{{.ResourcePath}}`" + `

All JSON responses use the Luas envelope:

` + "```json" + `
{
  "code": 0,
  "message": "success",
  "data": {}
}
` + "```" + `

Errors expose stable ` + "`error_code`" + ` values and may include ` + "`request_id`" + `.

## {{.ModelName}} Shape

` + "```json" + `
{
  "id": 1,
  "name": "Example",
  "created_at": "2026-04-01T00:00:00Z",
  "updated_at": "2026-04-01T00:00:00Z"
}
` + "```" + `

## List {{.ModelName}}s

` + "`GET {{.ResourcePath}}?page=1&page_size=20`" + `

Success: ` + "`200 OK`" + `

The response is paginated with ` + "`data`" + `, ` + "`meta`" + `, and ` + "`links`" + `.

## Get {{.ModelName}}

` + "`GET {{.ResourcePath}}/{id}`" + `

Success: ` + "`200 OK`" + `

` + "`data`" + ` is a {{.ModelName}} object.

## Create {{.ModelName}}

` + "`POST {{.ResourcePath}}`" + `

Request:

` + "```json" + `
{
  "name": "Example"
}
` + "```" + `

Success: ` + "`201 Created`" + `

` + "`data`" + ` is the created {{.ModelName}} object.

## Update {{.ModelName}}

` + "`PUT {{.ResourcePath}}/{id}`" + `

Request:

` + "```json" + `
{
  "name": "Updated example"
}
` + "```" + `

Success: ` + "`200 OK`" + `

` + "`data`" + ` is the updated {{.ModelName}} object.

## Delete {{.ModelName}}

` + "`DELETE {{.ResourcePath}}/{id}`" + `

Success: ` + "`204 No Content`" + `

## Error Codes

- ` + "`COMMON.VALIDATION_FAILED`" + `
- ` + "`COMMON.NOT_FOUND`" + `
- ` + "`COMMON.INVALID_INPUT`" + `
`

const webFeatureTypesTemplate = `export interface {{.ModelName}} {
  id: number;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Create{{.ModelName}}Request {
  name: string;
}

export interface Update{{.ModelName}}Request {
  name?: string;
}

export interface Paginated{{.ModelName}}Response {
  data: {{.ModelName}}[];
  meta?: unknown;
  links?: unknown;
}
`

const webFeatureServiceTemplate = `import request from '@/http';
import type {
  Create{{.ModelName}}Request,
  Paginated{{.ModelName}}Response,
  {{.ModelName}},
  Update{{.ModelName}}Request,
} from '@/features/{{.Package}}/types';

const basePath = '/{{.RouteCollection}}';

export const {{.Package}}Service = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<Paginated{{.ModelName}}Response>(basePath, { params }),

  get: (id: number) => request.get<{{.ModelName}}>(` + "`${basePath}/${id}`" + `),

  create: (data: Create{{.ModelName}}Request) =>
    request.post<{{.ModelName}}, Create{{.ModelName}}Request>(basePath, data),

  update: (id: number, data: Update{{.ModelName}}Request) =>
    request.put<{{.ModelName}}, Update{{.ModelName}}Request>(` + "`${basePath}/${id}`" + `, data),

  delete: (id: number) => request.delete<void>(` + "`${basePath}/${id}`" + `),
};

export type {{.ModelName}}Service = typeof {{.Package}}Service;
`

const webFeatureHookTemplate = `'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { {{.Package}}Service } from '@/features/{{.Package}}/services/{{.Package}}-service';
import type { Create{{.ModelName}}Request, Update{{.ModelName}}Request } from '@/features/{{.Package}}/types';

export const {{.Package}}QueryKeys = {
  all: ['{{.Package}}'] as const,
  list: (page?: number) => [...{{.Package}}QueryKeys.all, 'list', page] as const,
  detail: (id: number) => [...{{.Package}}QueryKeys.all, 'detail', id] as const,
};

export function use{{.ModelName}}List(page = 1) {
  return useQuery({
    queryKey: {{.Package}}QueryKeys.list(page),
    queryFn: () => {{.Package}}Service.list({ page }),
  });
}

export function use{{.ModelName}}(id: number) {
  return useQuery({
    queryKey: {{.Package}}QueryKeys.detail(id),
    queryFn: () => {{.Package}}Service.get(id),
    enabled: id > 0,
  });
}

export function useCreate{{.ModelName}}() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Create{{.ModelName}}Request) => {{.Package}}Service.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: {{.Package}}QueryKeys.all }),
  });
}

export function useUpdate{{.ModelName}}(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Update{{.ModelName}}Request) => {{.Package}}Service.update(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: {{.Package}}QueryKeys.all });
      void queryClient.invalidateQueries({ queryKey: {{.Package}}QueryKeys.detail(id) });
    },
  });
}

export function useDelete{{.ModelName}}() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => {{.Package}}Service.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: {{.Package}}QueryKeys.all }),
  });
}
`

const webFeatureServerTemplate = `import type { {{.ModelName}} } from '@/features/{{.Package}}/types';

export const mock{{.ModelName}}Items: {{.ModelName}}[] = [];
`

const webFeatureIndexTemplate = `export * from './hooks/use-{{.Package}}';
export * from './services/{{.Package}}-service';
export type * from './types';
`

const serviceTestTemplate = `package {{.Package}}

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

const providerTemplate = `package {{.Package}}

import (
	"github.com/google/wire"
	"github.com/zgiai/luas/api/internal/domain"
)

// ProviderSet is the provider set for this module.
var ProviderSet = wire.NewSet(
	NewRepository,
	wire.Bind(new(domain.{{.ModelName}}Repository), new(*repository)),
	NewService,
	wire.Bind(new(Service), new(*service)),
	NewHandler,
)
`
