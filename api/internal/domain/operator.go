package domain

import (
	"context"
	"time"
)

// OperatorGrant makes a user a platform operator. Operators run the deployment through the Admin
// Console; the grant is not an organization role or a permission key.
type OperatorGrant struct {
	UserID    uint
	Username  string
	Email     string
	GrantedAt time.Time
}

// OperatorGrantStore is the CLI-facing seam for creating, removing, and listing operator grants.
type OperatorGrantStore interface {
	GrantOperator(ctx context.Context, email string) (grant *OperatorGrant, created bool, err error)
	RevokeOperator(ctx context.Context, email string) (userID uint, removed bool, err error)
	ListOperators(ctx context.Context) ([]OperatorGrant, error)
}
