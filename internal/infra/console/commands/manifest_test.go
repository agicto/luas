package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgiai/zgo/internal/infra/console"
)

func TestDefaultManifestsRegisterExpectedCommands(t *testing.T) {
	app := console.New("zgo", "test")
	RegisterManifests(app, DefaultManifests("test")...)

	assert.True(t, app.HasCommand("make:module"))
	assert.True(t, app.HasCommand("migrate"))
	assert.True(t, app.HasCommand("db:status"))
	assert.True(t, app.HasCommand("migrate:status"))
	assert.True(t, app.HasCommand("seed"))
	assert.True(t, app.HasCommand("ai:chat"))
	assert.True(t, app.HasCommand("workflow:work"))
	assert.True(t, app.HasCommand("workflow:schedule:run"))
	assert.True(t, app.HasCommand("workflow:schedule:work"))
	assert.False(t, app.HasCommand("deploy:run"))
}
