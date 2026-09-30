package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_04_26_000000_create_audit_logs_table", &createAuditLogsTable{})
}

type createAuditLogsTable struct {
	migration.BaseMigration
}

// Up creates the audit_logs table.
func (m *createAuditLogsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE audit_logs (
		    id bigserial NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    user_id bigint,
		    actor_type varchar(20) NOT NULL,
		    actor_id bigint,
		    api_key_id bigint,
		    action varchar(120) NOT NULL,
		    resource varchar(180) NOT NULL,
		    target_type varchar(80),
		    target_id varchar(120),
		    result varchar(40),
		    method varchar(10) NOT NULL,
		    path varchar(255) NOT NULL,
		    route_name varchar(180),
		    status_code bigint NOT NULL,
		    request_id varchar(80),
		    ip_address varchar(64),
		    user_agent varchar(512),
		    changes text,
		    metadata text,
		    CONSTRAINT audit_logs_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_audit_logs_action ON audit_logs (action)`,
		`CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id)`,
		`CREATE INDEX idx_audit_logs_actor_type ON audit_logs (actor_type)`,
		`CREATE INDEX idx_audit_logs_api_key_id ON audit_logs (api_key_id)`,
		`CREATE INDEX idx_audit_logs_method ON audit_logs (method)`,
		`CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id)`,
		`CREATE INDEX idx_audit_logs_resource ON audit_logs (resource)`,
		`CREATE INDEX idx_audit_logs_result ON audit_logs (result)`,
		`CREATE INDEX idx_audit_logs_route_name ON audit_logs (route_name)`,
		`CREATE INDEX idx_audit_logs_status_code ON audit_logs (status_code)`,
		`CREATE INDEX idx_audit_logs_target_id ON audit_logs (target_id)`,
		`CREATE INDEX idx_audit_logs_target_type ON audit_logs (target_type)`,
		`CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id)`,
	)
}

// Down drops the audit_logs table.
func (m *createAuditLogsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS audit_logs CASCADE`,
	)
}
