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
