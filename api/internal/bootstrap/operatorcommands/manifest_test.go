package operatorcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fastagi/fastbuild/api/internal/infra/console"
	infracommands "github.com/fastagi/fastbuild/api/internal/infra/console/commands"
)

func TestManifestRegistersBusinessOperatorCommands(t *testing.T) {
	application := console.New("fastbuild", "test")
	infracommands.RegisterManifest(application, Manifest())

	assert.True(t, application.HasCommand("setting:list"))
	assert.True(t, application.HasCommand("auth-session:prune"))
	assert.True(t, application.HasCommand("audit:prune"))
	assert.True(t, application.HasCommand("usage:record"))
	assert.True(t, application.HasCommand("webhook:work"))
	assert.True(t, application.HasCommand("webhook:publish-test"))
	assert.True(t, application.HasCommand("webhook:replay"))
	assert.True(t, application.HasCommand("webhook:prune"))
}
