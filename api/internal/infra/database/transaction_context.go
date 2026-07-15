package database

import (
	"context"

	"gorm.io/gorm"
)

type transactionContextKey struct{}

// ContextWithTransaction scopes one GORM transaction to cooperating repositories.
// The returned context must not escape the transaction callback.
func ContextWithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

// TransactionFromContext returns a transaction explicitly bound by its owner.
func TransactionFromContext(ctx context.Context) (*gorm.DB, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

// ResolveContextDB prefers a bound transaction and otherwise applies ctx to fallback.
func ResolveContextDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := TransactionFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	if fallback == nil {
		return nil
	}
	return fallback.WithContext(ctx)
}
