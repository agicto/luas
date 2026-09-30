package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_15_050000_create_usage_tables", &createUsageTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createUsageTables struct {
	migration.BaseMigration
}

// Up creates quota history, current counters, and minimized idempotency receipts.
func (m *createUsageTables) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE usage_counters (
		    id bigserial NOT NULL,
		    scope varchar(24) NOT NULL,
		    subject_id bigint NOT NULL,
		    user_id bigint,
		    organization_id bigint,
		    metric varchar(96) NOT NULL,
		    period_start timestamptz NOT NULL,
		    period_end timestamptz NOT NULL,
		    value bigint NOT NULL,
		    version bigint NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT usage_counters_period_check CHECK ((period_end > period_start)),
		    CONSTRAINT usage_counters_scope_check CHECK ((scope IN ('user', 'organization'))),
		    CONSTRAINT usage_counters_subject_check CHECK (((((scope)::text = 'user'::text) AND (subject_id = user_id) AND (user_id IS NOT NULL) AND (organization_id IS NULL)) OR (((scope)::text = 'organization'::text) AND (subject_id = organization_id) AND (organization_id IS NOT NULL) AND (user_id IS NULL)))),
		    CONSTRAINT usage_counters_value_check CHECK (((value >= 0) AND (value <= '9007199254740991'::bigint))),
		    CONSTRAINT usage_counters_version_check CHECK ((version > 0)),
		    CONSTRAINT usage_counters_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_usage_counters_current ON usage_counters (scope, subject_id, metric, period_end)`,
		`CREATE UNIQUE INDEX idx_usage_counters_identity ON usage_counters (scope, subject_id, metric, period_start)`,
		`CREATE INDEX idx_usage_counters_organization ON usage_counters (organization_id)`,
		`CREATE INDEX idx_usage_counters_user ON usage_counters (user_id)`,
		`CREATE TABLE usage_quotas (
		    id bigserial NOT NULL,
		    scope varchar(24) NOT NULL,
		    subject_id bigint NOT NULL,
		    user_id bigint,
		    organization_id bigint,
		    metric varchar(96) NOT NULL,
		    limit_value bigint,
		    is_overridden boolean DEFAULT false NOT NULL,
		    version bigint NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT usage_quotas_limit_check CHECK (((limit_value IS NULL) OR ((limit_value >= 0) AND (limit_value <= '9007199254740991'::bigint)))),
		    CONSTRAINT usage_quotas_scope_check CHECK ((scope IN ('user', 'organization'))),
		    CONSTRAINT usage_quotas_state_check CHECK ((((is_overridden = true) AND (limit_value IS NOT NULL)) OR ((is_overridden = false) AND (limit_value IS NULL)))),
		    CONSTRAINT usage_quotas_subject_check CHECK (((((scope)::text = 'user'::text) AND (subject_id = user_id) AND (user_id IS NOT NULL) AND (organization_id IS NULL)) OR (((scope)::text = 'organization'::text) AND (subject_id = organization_id) AND (organization_id IS NOT NULL) AND (user_id IS NULL)))),
		    CONSTRAINT usage_quotas_version_check CHECK ((version > 0)),
		    CONSTRAINT usage_quotas_pkey PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX idx_usage_quotas_identity ON usage_quotas (scope, subject_id, metric)`,
		`CREATE INDEX idx_usage_quotas_organization ON usage_quotas (organization_id)`,
		`CREATE INDEX idx_usage_quotas_user ON usage_quotas (user_id)`,
		`CREATE TABLE usage_events (
		    id bigserial NOT NULL,
		    source varchar(64) NOT NULL,
		    event_id varchar(128) NOT NULL,
		    fingerprint varchar(64) NOT NULL,
		    operation varchar(16) NOT NULL,
		    scope varchar(24) NOT NULL,
		    subject_id bigint NOT NULL,
		    user_id bigint,
		    organization_id bigint,
		    metric varchar(96) NOT NULL,
		    quantity bigint NOT NULL,
		    dimensions_json text DEFAULT '{}'::text NOT NULL,
		    occurred_at timestamptz NOT NULL,
		    period_start timestamptz NOT NULL,
		    period_end timestamptz NOT NULL,
		    decision varchar(16) NOT NULL,
		    counter_before bigint NOT NULL,
		    counter_after bigint NOT NULL,
		    limit_value bigint,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT usage_events_counter_after_check CHECK (counter_after BETWEEN 0 AND 9007199254740991 AND (decision <> 'denied' OR counter_after = counter_before)),
		    CONSTRAINT usage_events_counter_before_check CHECK (((counter_before >= 0) AND (counter_before <= '9007199254740991'::bigint))),
		    CONSTRAINT usage_events_decision_check CHECK ((decision IN ('pending', 'accepted', 'denied'))),
		    CONSTRAINT usage_events_dimensions_check CHECK (((length(dimensions_json) >= 2) AND (length(dimensions_json) <= 4096))),
		    CONSTRAINT usage_events_fingerprint_check CHECK ((length((fingerprint)::text) = 64)),
		    CONSTRAINT usage_events_limit_check CHECK (((limit_value IS NULL) OR ((limit_value >= 0) AND (limit_value <= '9007199254740991'::bigint)))),
		    CONSTRAINT usage_events_operation_check CHECK ((operation IN ('record', 'consume'))),
		    CONSTRAINT usage_events_period_check CHECK ((period_end > period_start)),
		    CONSTRAINT usage_events_quantity_check CHECK (((quantity <> 0) AND ((quantity >= '-9007199254740991'::bigint) AND (quantity <= '9007199254740991'::bigint)) AND (((operation)::text = 'record'::text) OR (quantity > 0)))),
		    CONSTRAINT usage_events_scope_check CHECK ((scope IN ('user', 'organization'))),
		    CONSTRAINT usage_events_subject_check CHECK (((((scope)::text = 'user'::text) AND (subject_id = user_id) AND (user_id IS NOT NULL) AND (organization_id IS NULL)) OR (((scope)::text = 'organization'::text) AND (subject_id = organization_id) AND (organization_id IS NOT NULL) AND (user_id IS NULL)))),
		    CONSTRAINT usage_events_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_usage_events_organization ON usage_events (organization_id)`,
		`CREATE UNIQUE INDEX idx_usage_events_source_event ON usage_events (source, event_id)`,
		`CREATE INDEX idx_usage_events_subject ON usage_events (scope, subject_id, metric, period_start)`,
		`CREATE INDEX idx_usage_events_user ON usage_events (user_id)`,
		`ALTER TABLE usage_counters ADD CONSTRAINT fk_usage_counters_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE usage_counters ADD CONSTRAINT fk_usage_counters_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE usage_quotas ADD CONSTRAINT fk_usage_quotas_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE usage_quotas ADD CONSTRAINT fk_usage_quotas_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE usage_events ADD CONSTRAINT fk_usage_events_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE usage_events ADD CONSTRAINT fk_usage_events_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes usage receipts before quota and counter ownership tables.
func (m *createUsageTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS usage_events CASCADE`,
		`DROP TABLE IF EXISTS usage_quotas CASCADE`,
		`DROP TABLE IF EXISTS usage_counters CASCADE`,
	)
}
