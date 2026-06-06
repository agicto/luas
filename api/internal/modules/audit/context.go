package audit

import (
	"context"

	"github.com/zgiai/luas/api/internal/contracts"
)

// Change describes business-level audit metadata captured during request handling.
type Change = contracts.AuditChange

func withChangeCollector(ctx context.Context) context.Context {
	return contracts.WithAuditChangeCollector(ctx)
}

// RecordChange enriches the current request's audit entry with business-level semantics.
//
// Deprecated: use contracts.RecordAuditChange from starter modules. This wrapper
// is kept so existing audit package tests and callers continue to exercise the
// same collector seam.
func RecordChange(ctx context.Context, change Change) {
	contracts.RecordAuditChange(ctx, change)
}

func changeFromContext(ctx context.Context) *Change {
	return contracts.AuditChangeFromContext(ctx)
}
