package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeModuleCommandCreatesDDDScaffold(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeModuleCommand()
	require.NoError(t, cmd.Run([]string{"BlogPost"}))

	domainPath := filepath.Join(tmp, "internal", "domain", "blog_post.go")
	moduleDir := filepath.Join(tmp, "internal", "modules", "blog_post")

	requiredFiles := []string{
		domainPath,
		filepath.Join(moduleDir, "model.go"),
		filepath.Join(moduleDir, "service.go"),
		filepath.Join(moduleDir, "handler.go"),
		filepath.Join(moduleDir, "repository.go"),
		filepath.Join(moduleDir, "dto.go"),
		filepath.Join(moduleDir, "routes.go"),
		filepath.Join(moduleDir, "service_test.go"),
		filepath.Join(moduleDir, "provider.go"),
	}

	for _, path := range requiredFiles {
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, path)
	}

	handlerContent, err := os.ReadFile(filepath.Join(moduleDir, "handler.go"))
	require.NoError(t, err)
	assert.Contains(t, string(handlerContent), "contracts.RouteModule")
	assert.Contains(t, string(handlerContent), "Failed to list blog_posts")

	routesContent, err := os.ReadFile(filepath.Join(moduleDir, "routes.go"))
	require.NoError(t, err)
	assert.Contains(t, string(routesContent), "func (h *Handler) RegisterRoutes")
	assert.Contains(t, string(routesContent), "/blog_posts")
}

func TestMakeModuleCommandWithContractAndMigration(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	apiDir := filepath.Join(tmp, "api")
	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.Chdir(apiDir))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeModuleCommand()
	require.NoError(t, cmd.Run([]string{"Invoice", "--with=contract,migration,web"}))

	_, err = os.Stat(filepath.Join(tmp, "contracts", "invoice.md"))
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(apiDir, "database", "migrations", "*_create_invoices_table.go"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	webFeatureDir := filepath.Join(tmp, "web", "src", "features", "invoice")
	for _, path := range []string{
		filepath.Join(webFeatureDir, "types.ts"),
		filepath.Join(webFeatureDir, "services", "invoice-service.ts"),
		filepath.Join(webFeatureDir, "hooks", "use-invoice.ts"),
		filepath.Join(webFeatureDir, "server", "mock-invoice-store.ts"),
		filepath.Join(webFeatureDir, "index.ts"),
	} {
		_, err = os.Stat(path)
		require.NoError(t, err, path)
	}
}

func TestMakeServiceCommandUsesExistingModuleScaffold(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	require.NoError(t, os.MkdirAll(filepath.Join("internal", "modules", "order_item"), 0755))

	cmd := NewMakeServiceCommand()
	require.NoError(t, cmd.Run([]string{"OrderItem"}))

	servicePath := filepath.Join(tmp, "internal", "modules", "order_item", "service.go")
	domainPath := filepath.Join(tmp, "internal", "domain", "order_item.go")

	_, err = os.Stat(servicePath)
	require.NoError(t, err)
	_, err = os.Stat(domainPath)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmp, "app", "order_item", "service.go"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	serviceContent, err := os.ReadFile(servicePath)
	require.NoError(t, err)
	assert.Contains(t, string(serviceContent), "domain.OrderItemRepository")
	assert.Contains(t, string(serviceContent), "CreateOrderItemRequest")
}

func TestMakeServiceCommandRequiresExistingModule(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeServiceCommand()
	err = cmd.Run([]string{"Invoice"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "make:module Invoice"))
}

func TestMakeContractCommandCreatesRootContractFromAPICwd(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	apiDir := filepath.Join(tmp, "api")
	require.NoError(t, os.MkdirAll(apiDir, 0755))
	require.NoError(t, os.Chdir(apiDir))
	defer func() {
		_ = os.Chdir(wd)
	}()

	cmd := NewMakeContractCommand()
	require.NoError(t, cmd.Run([]string{"BillingPlan", "--resource=/v1/billing-plans"}))

	contractPath := filepath.Join(tmp, "contracts", "billing_plan.md")
	content, err := os.ReadFile(contractPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# BillingPlan Contract")
	assert.Contains(t, string(content), "Base path: `/v1/billing-plans`")
	assert.Contains(t, string(content), "`POST /v1/billing-plans`")
}
