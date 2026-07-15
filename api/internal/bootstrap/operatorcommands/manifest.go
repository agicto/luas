package operatorcommands

import infracommands "github.com/zgiai/luas/api/internal/infra/console/commands"

// Manifest returns business-aware operator commands assembled outside infrastructure packages.
func Manifest() infracommands.Manifest {
	return infracommands.NewManifest(
		"operator",
		infracommands.Registration{Command: NewSettingListCommand()},
		infracommands.Registration{Command: NewSettingSetCommand()},
		infracommands.Registration{Command: NewSettingResetCommand()},
		infracommands.Registration{Command: NewUsageListCommand()},
		infracommands.Registration{Command: NewUsageRecordCommand()},
		infracommands.Registration{Command: NewUsageConsumeCommand()},
		infracommands.Registration{Command: NewUsageQuotaSetCommand()},
		infracommands.Registration{Command: NewUsageQuotaResetCommand()},
		infracommands.Registration{Command: NewUsagePruneCommand()},
	)
}
