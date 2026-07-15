package assembly

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type typedNilModule struct{}

func (*typedNilModule) Name() string { return "typed-nil" }

func TestWithStarterModuleIgnoresTypedNilPointers(t *testing.T) {
	var module *typedNilModule
	manifest := NewStaticStarterManifest("test", WithStarterModule(module))

	assert.Empty(t, manifest.Modules())
}

func TestStaticStarterManifestReturnsDefensiveDependencyCopies(t *testing.T) {
	manifest := NewStaticStarterManifest(
		"permission",
		WithStarterDependencies("organization"),
	)

	dependencies := manifest.Dependencies()
	assert.Equal(t, []string{"organization"}, dependencies)
	dependencies[0] = "changed"
	assert.Equal(t, []string{"organization"}, manifest.Dependencies())
}
