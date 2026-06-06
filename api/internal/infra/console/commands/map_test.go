package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectProjectMap(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "api", "internal", "modules", "user"))
	mkdirAll(t, filepath.Join(root, "api", "database", "migrations"))
	mkdirAll(t, filepath.Join(root, "web", "src", "features", "auth"))
	mkdirAll(t, filepath.Join(root, "contracts"))
	writeFile(t, filepath.Join(root, "api", "go.mod"), "module example.test/api\n")
	writeFile(t, filepath.Join(root, "api", "internal", "modules", "user", "provider.go"), `package user

import "github.com/zgiai/luas/api/internal/contracts"

func manifest() {
	_ = contracts.StarterMigration{Name: "2026_01_01_000000_create_users"}
	_ = contracts.StarterSeeder{Name: "users"}
}
`)
	writeFile(t, filepath.Join(root, "api", "internal", "modules", "user", "routes.go"), `package user

import "github.com/zgiai/luas/api/internal/infra/router"

func (h *Handler) RegisterRoutes(r *router.Router) {
	r.Group("/users", func(group *router.Router) {
		group.GET("", h.List).Name("users.index")
		group.POST("", h.Create).Name("users.store")
		group.GET("/:id", h.Get).Name("users.show").WhereNumber("id")
	})
}
`)
	writeFile(t, filepath.Join(root, "api", "database", "migrations", "migrations.go"), "package migrations\n")
	writeFile(t, filepath.Join(root, "api", "database", "migrations", "2026_01_01_000000_create_users.go"), "package migrations\n")
	writeFile(t, filepath.Join(root, "contracts", "auth.md"), "# Auth Contract\n\nBase path: `/v1/auth`\n\n`POST /v1/auth/login`\n")
	writeFile(t, filepath.Join(root, "contracts", "README.md"), "# Contracts\n")

	project, err := collectProjectMap(root)
	require.NoError(t, err)
	require.Len(t, project.API.Modules, 1)
	require.Equal(t, "user", project.API.Modules[0].Name)
	require.Equal(t, "api/internal/modules/user", project.API.Modules[0].Path)
	require.Equal(t, []string{"provider.go", "routes.go"}, project.API.Modules[0].Files)
	require.Equal(t, []string{"2026_01_01_000000_create_users"}, project.API.Modules[0].Manifest.Migrations)
	require.Equal(t, []string{"users"}, project.API.Modules[0].Manifest.Seeders)
	require.Equal(t, []string{"2026_01_01_000000_create_users"}, project.API.Migrations)
	require.Len(t, project.API.Routes, 3)
	require.Equal(t, routeMapItem{Module: "user", Method: "GET", Path: "/users", Name: "users.index"}, project.API.Routes[0])
	require.Equal(t, []string{"auth"}, project.Web.Features)
	require.Len(t, project.Contracts, 1)
	require.Equal(t, "auth", project.Contracts[0].Name)
	require.Equal(t, "contracts/auth.md", project.Contracts[0].Path)
	require.Equal(t, "Auth Contract", project.Contracts[0].Title)
	require.Equal(t, "/v1/auth", project.Contracts[0].BasePath)
	require.Equal(t, []string{"POST /v1/auth/login"}, project.Contracts[0].Endpoints)
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(path, 0o755))
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
