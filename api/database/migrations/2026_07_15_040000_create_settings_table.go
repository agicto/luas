package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_15_040000_create_settings_table", &createSettingsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createSettingsTable struct {
	migration.BaseMigration
}

// Up creates typed setting override history after user and organization ownership tables.
func (m *createSettingsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE settings (
		    id bigserial NOT NULL,
		    scope varchar(24) NOT NULL,
		    subject_id bigint NOT NULL,
		    user_id bigint,
		    organization_id bigint,
		    key varchar(96) NOT NULL,
		    value_json text DEFAULT ''::text NOT NULL,
		    is_overridden boolean DEFAULT false NOT NULL,
		    version bigint NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT settings_key_check CHECK (((length((key)::text) >= 1) AND (length((key)::text) <= 96))),
		    CONSTRAINT settings_scope_check CHECK ((scope IN ('app', 'organization', 'user'))),
		    CONSTRAINT settings_subject_check CHECK (((((scope)::text = 'app'::text) AND (subject_id = 0) AND (user_id IS NULL) AND (organization_id IS NULL)) OR (((scope)::text = 'user'::text) AND (subject_id = user_id) AND (user_id IS NOT NULL) AND (organization_id IS NULL)) OR (((scope)::text = 'organization'::text) AND (subject_id = organization_id) AND (organization_id IS NOT NULL) AND (user_id IS NULL)))),
		    CONSTRAINT settings_value_state_check CHECK (((length(value_json) <= 4096) AND (((is_overridden = true) AND (value_json <> ''::text)) OR ((is_overridden = false) AND (value_json = ''::text))))),
		    CONSTRAINT settings_version_check CHECK ((version > 0)),
		    CONSTRAINT settings_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_settings_organization ON settings (organization_id, subject_id)`,
		`CREATE UNIQUE INDEX idx_settings_scope_subject_key ON settings (scope, subject_id, key)`,
		`CREATE INDEX idx_settings_user ON settings (user_id, subject_id)`,
		`ALTER TABLE settings ADD CONSTRAINT fk_settings_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE settings ADD CONSTRAINT fk_settings_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes setting overrides and reset tombstones.
func (m *createSettingsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS settings CASCADE`,
	)
}
