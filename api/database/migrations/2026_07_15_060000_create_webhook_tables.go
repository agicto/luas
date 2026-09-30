package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_15_060000_create_webhook_tables", &createWebhookTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createWebhookTables struct {
	migration.BaseMigration
}

// Up creates endpoint custody, durable events, deliveries, and minimized attempts.
func (m *createWebhookTables) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE webhook_endpoints (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    name varchar(100) NOT NULL,
		    url varchar(2048) NOT NULL,
		    url_hash varchar(64) NOT NULL,
		    status varchar(16) NOT NULL,
		    disabled_reason varchar(64) DEFAULT ''::varchar NOT NULL,
		    consecutive_failures smallint DEFAULT 0 NOT NULL,
		    version bigint NOT NULL,
		    secret_ciphertext text NOT NULL,
		    secret_hint varchar(16) NOT NULL,
		    secret_version bigint NOT NULL,
		    previous_secret_ciphertext text DEFAULT ''::text NOT NULL,
		    previous_secret_valid_until timestamptz,
		    created_by bigint NOT NULL,
		    updated_by bigint NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    deleted_at timestamptz,
		    CONSTRAINT webhook_endpoints_name_check CHECK (((length((name)::text) >= 1) AND (length((name)::text) <= 100))),
		    CONSTRAINT webhook_endpoints_secret_version_check CHECK ((secret_version > 0)),
		    CONSTRAINT webhook_endpoints_status_check CHECK ((status IN ('active', 'disabled'))),
		    CONSTRAINT webhook_endpoints_url_check CHECK (((length((url)::text) >= 1) AND (length((url)::text) <= 2048))),
		    CONSTRAINT webhook_endpoints_url_hash_check CHECK ((length((url_hash)::text) = 64)),
		    CONSTRAINT webhook_endpoints_version_check CHECK ((version > 0)),
		    CONSTRAINT webhook_endpoints_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_webhook_endpoints_deleted_at ON webhook_endpoints (deleted_at)`,
		`CREATE INDEX idx_webhook_endpoints_organization_created ON webhook_endpoints (organization_id, created_at, id)`,
		`CREATE INDEX idx_webhook_endpoints_organization_status ON webhook_endpoints (organization_id, status, id)`,
		`CREATE INDEX idx_webhook_endpoints_previous_secret_expiry ON webhook_endpoints (previous_secret_valid_until)`,
		`CREATE TABLE webhook_subscriptions (
		    id bigserial NOT NULL,
		    endpoint_id bigint NOT NULL,
		    organization_id bigint NOT NULL,
		    event_type varchar(100) NOT NULL,
		    created_at timestamptz,
		    CONSTRAINT webhook_subscriptions_pkey PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX idx_webhook_subscriptions_endpoint_type ON webhook_subscriptions (endpoint_id, event_type)`,
		`CREATE INDEX idx_webhook_subscriptions_organization_type ON webhook_subscriptions (organization_id, event_type)`,
		`CREATE TABLE webhook_events (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    message_id varchar(64) NOT NULL,
		    source varchar(64) NOT NULL,
		    event_id varchar(128) NOT NULL,
		    fingerprint varchar(64) NOT NULL,
		    event_type varchar(100) NOT NULL,
		    payload_json text NOT NULL,
		    occurred_at timestamptz NOT NULL,
		    created_at timestamptz,
		    CONSTRAINT webhook_events_fingerprint_check CHECK ((length((fingerprint)::text) = 64)),
		    CONSTRAINT webhook_events_payload_check CHECK (((length(payload_json) >= 2) AND (length(payload_json) <= 65536))),
		    CONSTRAINT webhook_events_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_webhook_events_created ON webhook_events (created_at, id)`,
		`CREATE INDEX idx_webhook_events_event_type ON webhook_events (event_type)`,
		`CREATE UNIQUE INDEX idx_webhook_events_identity ON webhook_events (organization_id, source, event_id)`,
		`CREATE UNIQUE INDEX idx_webhook_events_message_id ON webhook_events (message_id)`,
		`CREATE INDEX idx_webhook_events_organization_created ON webhook_events (organization_id, created_at)`,
		`CREATE TABLE webhook_deliveries (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    endpoint_id bigint NOT NULL,
		    event_id bigint NOT NULL,
		    destination_hash varchar(64) NOT NULL,
		    status varchar(16) NOT NULL,
		    attempt_count integer DEFAULT 0 NOT NULL,
		    cycle_attempt smallint DEFAULT 0 NOT NULL,
		    replay_count integer DEFAULT 0 NOT NULL,
		    available_at timestamptz NOT NULL,
		    lease_token varchar(64) DEFAULT ''::varchar NOT NULL,
		    lease_expires_at timestamptz,
		    http_status bigint,
		    failure_code varchar(64) DEFAULT ''::varchar NOT NULL,
		    response_truncated boolean DEFAULT false NOT NULL,
		    delivered_at timestamptz,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT webhook_deliveries_destination_hash_check CHECK ((length((destination_hash)::text) = 64)),
		    CONSTRAINT webhook_deliveries_status_check CHECK ((status IN ('pending', 'processing', 'delivered', 'failed', 'canceled'))),
		    CONSTRAINT webhook_deliveries_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_webhook_deliveries_due ON webhook_deliveries (status, available_at, id)`,
		`CREATE INDEX idx_webhook_deliveries_endpoint ON webhook_deliveries (endpoint_id)`,
		`CREATE UNIQUE INDEX idx_webhook_deliveries_event_endpoint ON webhook_deliveries (event_id, endpoint_id)`,
		`CREATE INDEX idx_webhook_deliveries_lease_expiry ON webhook_deliveries (status, lease_expires_at)`,
		`CREATE INDEX idx_webhook_deliveries_organization_created ON webhook_deliveries (organization_id, created_at, id)`,
		`CREATE INDEX idx_webhook_deliveries_organization_endpoint_created ON webhook_deliveries (organization_id, endpoint_id, created_at, id)`,
		`CREATE INDEX idx_webhook_deliveries_organization_status_created ON webhook_deliveries (organization_id, status, created_at, id)`,
		`CREATE TABLE webhook_delivery_attempts (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    delivery_id bigint NOT NULL,
		    number integer NOT NULL,
		    outcome varchar(32) NOT NULL,
		    http_status bigint,
		    failure_code varchar(64) DEFAULT ''::varchar NOT NULL,
		    duration_ms bigint NOT NULL,
		    response_truncated boolean DEFAULT false NOT NULL,
		    started_at timestamptz NOT NULL,
		    completed_at timestamptz NOT NULL,
		    CONSTRAINT webhook_delivery_attempts_pkey PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX idx_webhook_attempts_delivery_number ON webhook_delivery_attempts (delivery_id, number)`,
		`CREATE INDEX idx_webhook_delivery_attempts_organization_id ON webhook_delivery_attempts (organization_id)`,
		`ALTER TABLE webhook_endpoints ADD CONSTRAINT fk_webhook_endpoints_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE webhook_subscriptions ADD CONSTRAINT fk_webhook_endpoints_subscriptions FOREIGN KEY (endpoint_id) REFERENCES webhook_endpoints(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE webhook_events ADD CONSTRAINT fk_webhook_events_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE webhook_deliveries ADD CONSTRAINT fk_webhook_deliveries_endpoint FOREIGN KEY (endpoint_id) REFERENCES webhook_endpoints(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE webhook_deliveries ADD CONSTRAINT fk_webhook_deliveries_event FOREIGN KEY (event_id) REFERENCES webhook_events(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE webhook_delivery_attempts ADD CONSTRAINT fk_webhook_delivery_attempts_delivery FOREIGN KEY (delivery_id) REFERENCES webhook_deliveries(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes attempt and delivery history before endpoint and event owners.
func (m *createWebhookTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS webhook_delivery_attempts CASCADE`,
		`DROP TABLE IF EXISTS webhook_deliveries CASCADE`,
		`DROP TABLE IF EXISTS webhook_events CASCADE`,
		`DROP TABLE IF EXISTS webhook_subscriptions CASCADE`,
		`DROP TABLE IF EXISTS webhook_endpoints CASCADE`,
	)
}
