package config

import "slices"

// Starter names are the single vocabulary shared by OPTIONAL_STARTERS, starter manifests,
// selection-dependent validation, and starter-owned commands. Reference these constants instead of
// string literals so a rename or a new starter cannot drift between assembly and runtime checks.
const (
	StarterAudit        = "audit"
	StarterAPIKey       = "apikey"
	StarterUser         = "user"
	StarterOrganization = "organization"
	StarterPermission   = "permission"
	StarterNotification = "notification"
	StarterAsset        = "asset"
	StarterSetting      = "setting"
	StarterUsage        = "usage"
	StarterWebhook      = "webhook"
)

// StarterNames returns every starter name known to this build, defaults first.
func StarterNames() []string {
	return []string{
		StarterAudit,
		StarterAPIKey,
		StarterUser,
		StarterOrganization,
		StarterPermission,
		StarterNotification,
		StarterAsset,
		StarterSetting,
		StarterUsage,
		StarterWebhook,
	}
}

// Selected reports whether the named optional starter is selected in this snapshot.
// Default starters are always active and are not listed in OPTIONAL_STARTERS.
func (c StarterConfig) Selected(name string) bool {
	return slices.Contains(c.Optional, name)
}
