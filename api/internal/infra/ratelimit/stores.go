package ratelimit

import (
	"context"
	"time"
)

// Store is a fixed-window limiter whose bucket can also be forgotten, for example after a
// successful sign-in clears an account's failure budget.
type Store interface {
	Limiter
	Reset(ctx context.Context, key string)
}

// Rule describes one fixed-window quota. Name must be a stable, code-owned identifier: it
// namespaces shared buckets and labels metrics, so it must never contain request data.
type Rule struct {
	Name       string
	Max        int
	Window     time.Duration
	MaxBuckets int
}

// FailurePolicy decides a shared-store decision that cannot reach its backend in time.
type FailurePolicy string

const (
	// FailureLocal enforces the rule against a bounded process-local budget during the outage.
	FailureLocal FailurePolicy = "local"
	// FailureOpen allows the request.
	FailureOpen FailurePolicy = "open"
	// FailureClosed denies the request with the limiter's normal 429 response.
	FailureClosed FailurePolicy = "closed"
)

// Stores builds the store for each rate-limit rule from one backend selection, so the global
// limiter and every authentication quota share the same enforcement scope.
type Stores struct {
	shared *sharedBackend
}

// NewMemoryStores returns a factory for process-local stores. Each replica enforces its own
// budget, so N replicas admit up to N times every configured limit.
func NewMemoryStores() *Stores {
	return &Stores{}
}

// Shared reports whether stores enforce one budget across every replica.
func (s *Stores) Shared() bool {
	return s != nil && s.shared != nil
}

// New returns the store for one rule. A nil factory yields a process-local store.
func (s *Stores) New(rule Rule) Store {
	rule = normalizeRule(rule)
	if !s.Shared() {
		return newLocalStore(rule)
	}
	return newPostgresStore(s.shared, rule)
}

func newLocalStore(rule Rule) *MemoryStore {
	return NewMemoryStore(rule.Max, rule.Window, WithMaxBuckets(rule.MaxBuckets))
}

func normalizeRule(rule Rule) Rule {
	defaults := DefaultConfig()
	if rule.Max <= 0 {
		rule.Max = defaults.Max
	}
	if rule.Window <= 0 {
		rule.Window = defaults.Duration
	}
	if rule.MaxBuckets <= 0 {
		rule.MaxBuckets = DefaultMaxBuckets
	}
	if rule.Name == "" {
		rule.Name = "default"
	}
	return rule
}
