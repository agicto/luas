package organization

import (
	"time"

	"github.com/zgiai/luas/api/internal/infra/config"
)

// InvitationPolicy owns replaceable organization invitation timing policy.
type InvitationPolicy struct {
	TTL time.Duration
}

// NewInvitationPolicy builds policy from the validated startup snapshot.
func NewInvitationPolicy(cfg *config.Config) InvitationPolicy {
	ttl := config.DefaultOrganizationInvitationTTL
	if cfg != nil && cfg.Organization.InvitationTTL > 0 {
		ttl = cfg.Organization.InvitationTTL
	}
	return InvitationPolicy{TTL: ttl}
}
