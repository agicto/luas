package operatorcommands

import infracommands "github.com/zgiai/luas/api/internal/infra/console/commands"

// Manifest returns business-aware operator commands assembled outside infrastructure packages.
func Manifest() infracommands.Manifest {
	return infracommands.NewManifest(
		"setting",
		infracommands.Registration{Command: NewSettingListCommand()},
		infracommands.Registration{Command: NewSettingSetCommand()},
		infracommands.Registration{Command: NewSettingResetCommand()},
	)
}
