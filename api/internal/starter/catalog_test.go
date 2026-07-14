package starter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/starter/assembly"
)

func TestCatalogSelectKeepsDefaultsAndAddsRequestedOptionalStarters(t *testing.T) {
	catalog, err := NewCatalog(
		[]assembly.StarterManifest{testManifest("audit"), testManifest("user")},
		[]assembly.StarterManifest{testManifest("organization"), testManifest("notification")},
	)
	require.NoError(t, err)

	selected, err := catalog.Select([]string{"organization"})
	require.NoError(t, err)
	require.Len(t, selected, 3)
	assert.Equal(t, "audit", selected[0].Name())
	assert.Equal(t, "user", selected[1].Name())
	assert.Equal(t, "organization", selected[2].Name())
	assert.Equal(t, []string{"notification", "organization"}, catalog.OptionalNames())
}

func TestCatalogSelectRejectsAmbiguousConfiguration(t *testing.T) {
	catalog, err := NewCatalog(
		[]assembly.StarterManifest{testManifest("user")},
		[]assembly.StarterManifest{testManifest("organization")},
	)
	require.NoError(t, err)

	tests := []struct {
		name     string
		selected []string
		contains string
	}{
		{name: "unknown", selected: []string{"billing"}, contains: "unknown optional starter"},
		{name: "duplicate", selected: []string{"organization", "organization"}, contains: "duplicate optional starter"},
		{name: "default", selected: []string{"user"}, contains: "default starter"},
		{name: "non canonical", selected: []string{"Organization"}, contains: "canonical lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, selectErr := catalog.Select(tt.selected)
			require.Error(t, selectErr)
			assert.Contains(t, selectErr.Error(), tt.contains)
		})
	}
}

func TestNewCatalogRejectsDuplicateManifestNames(t *testing.T) {
	_, err := NewCatalog(
		[]assembly.StarterManifest{testManifest("user")},
		[]assembly.StarterManifest{testManifest("user")},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate starter manifest")
}

func TestNewCatalogRejectsNonCanonicalAndTypedNilManifests(t *testing.T) {
	tests := []struct {
		name     string
		manifest assembly.StarterManifest
	}{
		{name: "surrounding whitespace", manifest: testManifest(" user ")},
		{name: "typed nil", manifest: typedNilManifest()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCatalog([]assembly.StarterManifest{tt.manifest}, nil)
			require.Error(t, err)
		})
	}
}

type nilManifest struct{}

func (*nilManifest) Name() string               { return "nil-manifest" }
func (*nilManifest) Modules() []assembly.Module { return nil }
func (*nilManifest) MigrationNames() []string   { return nil }
func (*nilManifest) SeederNames() []string      { return nil }

func typedNilManifest() assembly.StarterManifest {
	var manifest *nilManifest
	return manifest
}

func testManifest(name string) assembly.StarterManifest {
	return assembly.NewStaticStarterManifest(name)
}
