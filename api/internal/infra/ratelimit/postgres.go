package ratelimit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/fastagi/fastbuild/api/internal/infra/metrics"
)

const (
	// DefaultStoreTimeout bounds one shared-store decision.
	DefaultStoreTimeout = 50 * time.Millisecond

	defaultCleanupEvery   = 10 * time.Second
	defaultCleanupBatch   = 1000
	maxPruneBatch         = 10_000
	cleanupTimeout        = 2 * time.Second
	failureWarnInterval   = 10 * time.Second
	closedRetryAfter      = time.Second
	failureReasonTimeout  = "timeout"
	failureReasonCanceled = "canceled"
	failureReasonError    = "error"
)

// takeBucketSQL consumes one hit in a single atomic upsert evaluated on the database clock. The
// hit counter saturates at max+1, so an allowed decision returns hits <= max and a denied one
// returns max+1 without growing; an expired window restarts at one hit.
const takeBucketSQL = `
INSERT INTO rate_limit_buckets AS bucket (key_hash, hits, window_end)
VALUES ($1, 1, statement_timestamp() + $2::bigint * interval '1 microsecond')
ON CONFLICT (key_hash) DO UPDATE SET
    hits = CASE
        WHEN bucket.window_end <= statement_timestamp() THEN 1
        ELSE LEAST(bucket.hits + 1, $3::bigint)
    END,
    window_end = CASE
        WHEN bucket.window_end <= statement_timestamp() THEN EXCLUDED.window_end
        ELSE bucket.window_end
    END
RETURNING hits, window_end`

const resetBucketSQL = `DELETE FROM rate_limit_buckets WHERE key_hash = $1`

// pruneBucketsSQL deletes one bounded batch of expired buckets. SKIP LOCKED keeps cleanup from
// waiting behind, or blocking, concurrent decisions on hot keys.
const pruneBucketsSQL = `
DELETE FROM rate_limit_buckets
WHERE key_hash IN (
    SELECT key_hash FROM rate_limit_buckets
    WHERE window_end <= statement_timestamp()
    ORDER BY window_end
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)`

// bucketStore is the persistence seam behind PostgresStore; tests replace it to prove the
// failure policy without a database.
type bucketStore interface {
	take(ctx context.Context, keyHash []byte, maxPlusOne int64, window time.Duration) (hits int64, windowEnd time.Time, err error)
	reset(ctx context.Context, keyHash []byte) error
	pruneExpired(ctx context.Context, batch int) (int64, error)
}

type sqlBuckets struct {
	db *sql.DB
}

func (b sqlBuckets) take(ctx context.Context, keyHash []byte, maxPlusOne int64, window time.Duration) (int64, time.Time, error) {
	var hits int64
	var windowEnd time.Time
	err := b.db.QueryRowContext(ctx, takeBucketSQL, keyHash, window.Microseconds(), maxPlusOne).Scan(&hits, &windowEnd)
	return hits, windowEnd, err
}

func (b sqlBuckets) reset(ctx context.Context, keyHash []byte) error {
	_, err := b.db.ExecContext(ctx, resetBucketSQL, keyHash)
	return err
}

func (b sqlBuckets) pruneExpired(ctx context.Context, batch int) (int64, error) {
	result, err := b.db.ExecContext(ctx, pruneBucketsSQL, batch)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PruneExpiredBuckets deletes expired shared buckets in batches until fewer than batch rows remain
// expired or maxBatches batches ran. It returns the number of deleted rows.
func PruneExpiredBuckets(ctx context.Context, db *sql.DB, batch, maxBatches int) (int64, error) {
	if db == nil {
		return 0, errors.New("rate-limit prune requires a database")
	}
	if batch < 1 || batch > maxPruneBatch {
		return 0, fmt.Errorf("rate-limit prune batch must be between 1 and %d", maxPruneBatch)
	}
	if maxBatches < 1 {
		return 0, errors.New("rate-limit prune requires at least one batch")
	}
	buckets := sqlBuckets{db: db}
	var total int64
	for range maxBatches {
		deleted, err := buckets.pruneExpired(ctx, batch)
		total += deleted
		if err != nil {
			return total, err
		}
		if deleted < int64(batch) {
			break
		}
	}
	return total, nil
}

// PostgresOptions configures the shared PostgreSQL-backed stores.
type PostgresOptions struct {
	// Timeout bounds one decision round trip (default 50ms).
	Timeout time.Duration
	// Failure decides a request whose decision failed or timed out (default local).
	Failure FailurePolicy
	// CleanupEvery spaces opportunistic expired-bucket cleanup per process (default 10s).
	CleanupEvery time.Duration
	// CleanupBatch bounds the rows one opportunistic cleanup deletes (default 1000).
	CleanupBatch int
	// Logger receives rate-limited outage warnings (default slog.Default()).
	Logger *slog.Logger
}

// NewPostgresStores returns a factory whose stores share one fixed-window budget per key across
// every replica connected to the same database.
func NewPostgresStores(db *sql.DB, options PostgresOptions) (*Stores, error) {
	if db == nil {
		return nil, errors.New("shared rate-limit store requires a database")
	}
	return newSharedStores(sqlBuckets{db: db}, options)
}

func newSharedStores(buckets bucketStore, options PostgresOptions) (*Stores, error) {
	backend, err := newSharedBackend(buckets, options)
	if err != nil {
		return nil, err
	}
	return &Stores{shared: backend}, nil
}

// sharedBackend holds per-process state shared by every PostgreSQL-backed rule: the outage
// policy, the warning throttle, and single-flight opportunistic cleanup.
type sharedBackend struct {
	buckets      bucketStore
	timeout      time.Duration
	failure      FailurePolicy
	cleanupEvery time.Duration
	cleanupBatch int
	logger       *slog.Logger
	now          func() time.Time

	nextCleanup    atomic.Int64
	cleaning       atomic.Bool
	nextWarn       atomic.Int64
	suppressedWarn atomic.Int64
}

func newSharedBackend(buckets bucketStore, options PostgresOptions) (*sharedBackend, error) {
	if options.Timeout <= 0 {
		options.Timeout = DefaultStoreTimeout
	}
	switch options.Failure {
	case "":
		options.Failure = FailureLocal
	case FailureLocal, FailureOpen, FailureClosed:
	default:
		return nil, fmt.Errorf("unknown rate-limit store failure policy %q", options.Failure)
	}
	if options.CleanupEvery <= 0 {
		options.CleanupEvery = defaultCleanupEvery
	}
	if options.CleanupBatch <= 0 || options.CleanupBatch > maxPruneBatch {
		options.CleanupBatch = defaultCleanupBatch
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	return &sharedBackend{
		buckets:      buckets,
		timeout:      options.Timeout,
		failure:      options.Failure,
		cleanupEvery: options.CleanupEvery,
		cleanupBatch: options.CleanupBatch,
		logger:       options.Logger,
		now:          time.Now,
	}, nil
}

// PostgresStore enforces one fixed-window rule against the shared rate_limit_buckets table. Each
// Take is one atomic upsert on the database clock; keys are stored only as SHA-256 digests of the
// rule namespace and the caller's key.
type PostgresStore struct {
	backend   *sharedBackend
	rule      Rule
	namespace []byte
	local     *MemoryStore
}

var _ Store = (*PostgresStore)(nil)

func newPostgresStore(backend *sharedBackend, rule Rule) *PostgresStore {
	store := &PostgresStore{
		backend: backend,
		rule:    rule,
		// The window is part of the namespace so a changed window starts fresh buckets instead of
		// inheriting window ends computed under the previous policy.
		namespace: []byte(rule.Name + "\x00" + strconv.FormatInt(rule.Window.Nanoseconds(), 10) + "\x00"),
	}
	if backend.failure == FailureLocal {
		store.local = newLocalStore(rule)
	}
	return store
}

// Take atomically consumes one hit from the shared bucket, falling back per the configured
// failure policy when the database errors or exceeds the decision timeout.
func (s *PostgresStore) Take(ctx context.Context, key string) (bool, int, time.Time) {
	queryCtx, cancel := context.WithTimeout(ctx, s.backend.timeout)
	hits, windowEnd, err := s.backend.buckets.take(queryCtx, s.keyHash(key), int64(s.rule.Max)+1, s.rule.Window)
	if err != nil {
		reason := failureReason(queryCtx, err)
		cancel()
		return s.fallback(ctx, key, reason, err)
	}
	cancel()
	s.backend.maybeCleanup()

	if hits > int64(s.rule.Max) {
		return false, 0, windowEnd
	}
	return true, s.rule.Max - int(hits), windowEnd
}

// Reset forgets the shared bucket and any outage-time local bucket for key.
func (s *PostgresStore) Reset(ctx context.Context, key string) {
	if s.local != nil {
		s.local.Reset(ctx, key)
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.backend.timeout)
	defer cancel()
	if err := s.backend.buckets.reset(queryCtx, s.keyHash(key)); err != nil {
		reason := failureReason(queryCtx, err)
		metrics.RecordRateLimitStoreFailure(s.rule.Name, reason, "reset")
		s.backend.warn(s.rule.Name, "reset", reason, err)
	}
}

func (s *PostgresStore) keyHash(key string) []byte {
	digest := sha256.New()
	digest.Write(s.namespace)
	digest.Write([]byte(key))
	return digest.Sum(nil)
}

func (s *PostgresStore) fallback(ctx context.Context, key, reason string, err error) (bool, int, time.Time) {
	var (
		allowed   bool
		remaining int
		resetAt   time.Time
		decision  string
	)
	switch s.backend.failure {
	case FailureOpen:
		allowed, remaining, resetAt, decision = true, s.rule.Max-1, s.backend.now().Add(s.rule.Window), "allow"
	case FailureClosed:
		allowed, remaining, resetAt, decision = false, 0, s.backend.now().Add(closedRetryAfter), "deny"
	default:
		allowed, remaining, resetAt = s.local.Take(ctx, key)
		decision = "local"
	}
	metrics.RecordRateLimitStoreFailure(s.rule.Name, reason, decision)
	s.backend.warn(s.rule.Name, decision, reason, err)
	return allowed, remaining, resetAt
}

func failureReason(queryCtx context.Context, err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(queryCtx.Err(), context.DeadlineExceeded):
		return failureReasonTimeout
	case errors.Is(err, context.Canceled) || errors.Is(queryCtx.Err(), context.Canceled):
		return failureReasonCanceled
	default:
		return failureReasonError
	}
}

// warn logs at most one outage warning per interval per process and reports how many it
// suppressed, so a database outage cannot flood logs at request rate.
func (b *sharedBackend) warn(rule, decision, reason string, err error) {
	now := b.now().UnixNano()
	next := b.nextWarn.Load()
	if now < next || !b.nextWarn.CompareAndSwap(next, now+failureWarnInterval.Nanoseconds()) {
		b.suppressedWarn.Add(1)
		return
	}
	b.logger.Warn("ratelimit.store_unavailable",
		"rule", rule,
		"reason", reason,
		"decision", decision,
		"policy", string(b.failure),
		"suppressed", b.suppressedWarn.Swap(0),
		"error", err.Error(),
	)
}

// maybeCleanup starts at most one bounded expired-bucket deletion per interval per process. It
// runs in a short-lived goroutine with its own timeout so no request waits for it; a full batch
// schedules the next pass immediately so a backlog drains at traffic rate.
func (b *sharedBackend) maybeCleanup() {
	if b.now().UnixNano() < b.nextCleanup.Load() || !b.cleaning.CompareAndSwap(false, true) {
		return
	}
	// Re-check under the flag: a cleanup that finished after the first check has already
	// rescheduled the next one.
	if b.now().UnixNano() < b.nextCleanup.Load() {
		b.cleaning.Store(false)
		return
	}
	go func() {
		defer b.cleaning.Store(false)
		b.cleanupOnce()
	}()
}

func (b *sharedBackend) cleanupOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	deleted, err := b.buckets.pruneExpired(ctx, b.cleanupBatch)
	next := b.now().Add(b.cleanupEvery)
	if err != nil {
		metrics.RecordRateLimitStoreFailure("cleanup", failureReason(ctx, err), "none")
	} else if deleted >= int64(b.cleanupBatch) {
		next = b.now()
	}
	b.nextCleanup.Store(next.UnixNano())
}
