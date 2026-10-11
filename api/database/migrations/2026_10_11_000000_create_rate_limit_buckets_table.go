package migrations

import (
	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/infra/migration"
)

// RateLimitBucketsMigration names the shared fixed-window rate-limit table migration.
const RateLimitBucketsMigration = "2026_10_11_000000_create_rate_limit_buckets_table"

func init() {
	register(RateLimitBucketsMigration, &createRateLimitBucketsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createRateLimitBucketsTable struct {
	migration.BaseMigration
}

// Up creates the shared rate-limit bucket table used when RATE_LIMIT_STORE=postgres.
//
// The table is UNLOGGED: counters are short-lived and losing them after a crash only resets the
// current windows, while skipping WAL makes the per-request upsert much cheaper. Keys are SHA-256
// digests, never raw client IPs or subjects. A lower fillfactor leaves room for HOT updates of the
// hit counter, and the window_end index keeps bounded expired-row cleanup from scanning the table.
func (m *createRateLimitBucketsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE UNLOGGED TABLE rate_limit_buckets (
		    key_hash bytea NOT NULL,
		    hits bigint NOT NULL,
		    window_end timestamptz NOT NULL,
		    CONSTRAINT rate_limit_buckets_key_hash_check CHECK ((octet_length(key_hash) = 32)),
		    CONSTRAINT rate_limit_buckets_hits_check CHECK ((hits >= 1)),
		    CONSTRAINT rate_limit_buckets_pkey PRIMARY KEY (key_hash)
		) WITH (fillfactor = 70)`,
		`CREATE INDEX idx_rate_limit_buckets_window_end ON rate_limit_buckets (window_end)`,
	)
}

// Down drops the shared rate-limit bucket table.
func (m *createRateLimitBucketsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS rate_limit_buckets`,
	)
}
