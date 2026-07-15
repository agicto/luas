package domain

import "context"

// OrganizationContext is a membership verified for one request.
type OrganizationContext struct {
	OrganizationID   uint
	OrganizationName string
	OrganizationSlug string
	MembershipID     uint
	UserID           uint
	Role             OrganizationRole
}

// IsValid reports whether the resolved context contains a complete membership identity.
func (c OrganizationContext) IsValid() bool {
	return c.OrganizationID > 0 &&
		c.OrganizationName != "" &&
		c.OrganizationSlug != "" &&
		c.MembershipID > 0 &&
		c.UserID > 0 &&
		c.Role.IsValid()
}

type organizationContextKey struct{}

// WithOrganizationContext binds one verified organization membership to a request context.
func WithOrganizationContext(ctx context.Context, value OrganizationContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, organizationContextKey{}, value)
}

// OrganizationContextFromContext returns the verified organization membership for this request.
func OrganizationContextFromContext(ctx context.Context) (OrganizationContext, bool) {
	if ctx == nil {
		return OrganizationContext{}, false
	}
	value, ok := ctx.Value(organizationContextKey{}).(OrganizationContext)
	return value, ok && value.IsValid()
}
