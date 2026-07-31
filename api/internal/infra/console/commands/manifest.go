package commands

import "github.com/zgiai/luas/api/internal/infra/console"

// Registration pairs a command with an optional alias name.
type Registration struct {
	Name    string
	Command console.Command
}

// Manifest groups CLI registrations behind a single seam.
type Manifest interface {
	Name() string
	Registrations() []Registration
}

type staticManifest struct {
	name          string
	registrations []Registration
}

func (m *staticManifest) Name() string {
	return m.name
}

func (m *staticManifest) Registrations() []Registration {
	return append([]Registration(nil), m.registrations...)
}

// NewManifest creates an immutable CLI command manifest for an assembly package.
func NewManifest(name string, registrations ...Registration) Manifest {
	return &staticManifest{name: name, registrations: registrations}
}

// RegisterManifest adds every command in a manifest to the CLI application.
func RegisterManifest(app *console.Application, manifest Manifest) {
	if app == nil || manifest == nil {
		return
	}

	for _, registration := range manifest.Registrations() {
		if registration.Command == nil {
			continue
		}
		if registration.Name == "" {
			app.Register(registration.Command)
			continue
		}
		app.RegisterAs(registration.Name, registration.Command)
	}
}

// RegisterManifests adds multiple manifests to the CLI application.
func RegisterManifests(app *console.Application, manifests ...Manifest) {
	for _, manifest := range manifests {
		RegisterManifest(app, manifest)
	}
}

// DefaultManifests returns the built-in CLI command manifests.
func DefaultManifests(version string) []Manifest {
	makeManifest := NewManifest(
		"make",
		Registration{Command: NewMakeModelCommand()},
		Registration{Command: NewMakeServiceCommand()},
		Registration{Command: NewMakeHandlerCommand()},
		Registration{Command: NewMakeRepositoryCommand()},
		Registration{Command: NewMakeSeederCommand()},
		Registration{Command: NewMakeMigrationCommand()},
		Registration{Command: NewMakeModuleCommand()},
	)

	migrateCommand := NewMigrateCommand()
	freshCommand := NewFreshCommand()
	rollbackCommand := NewRollbackCommand()
	resetCommand := NewResetCommand()
	statusCommand := NewStatusCommand()
	seedCommand := NewDBSeedCommand()

	databaseManifest := NewManifest(
		"database",
		Registration{Command: migrateCommand},
		Registration{Name: "migrate", Command: migrateCommand},
		Registration{Command: freshCommand},
		Registration{Name: "migrate:fresh", Command: freshCommand},
		Registration{Command: rollbackCommand},
		Registration{Name: "migrate:rollback", Command: rollbackCommand},
		Registration{Command: resetCommand},
		Registration{Name: "migrate:reset", Command: resetCommand},
		Registration{Command: statusCommand},
		Registration{Name: "migrate:status", Command: statusCommand},
		Registration{Command: seedCommand},
		Registration{Name: "seed", Command: seedCommand},
	)

	coreManifest := NewManifest(
		"core",
		Registration{Command: NewAIChatCommand()},
		Registration{Command: NewDoctorCommand()},
		Registration{Command: NewHealthCheckCommand()},
		Registration{Command: NewServeCommand()},
		Registration{Command: NewEnvCommand()},
		Registration{Command: NewVersionCommand(version)},
		Registration{Command: NewRouteListCommand()},
		Registration{Command: NewPluginListCommand()},
		Registration{Command: NewWorkflowWorkCommand()},
		Registration{Command: NewWorkflowScheduleRunCommand()},
		Registration{Command: NewWorkflowScheduleWorkCommand()},
		Registration{Command: NewNotificationWorkCommand()},
		Registration{Command: NewAssetPruneCommand()},
	)

	starterManifest := NewManifest(
		"starter",
		Registration{Command: NewStarterListCommand()},
		Registration{Command: NewStarterCheckCommand()},
		Registration{Command: NewStarterEnableCommand()},
		Registration{Command: NewStarterDisableCommand()},
	)

	return []Manifest{makeManifest, databaseManifest, starterManifest, coreManifest}
}
