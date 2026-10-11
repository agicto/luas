package migrations

import (
	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/infra/migration"
)

func init() {
	register("2026_10_11_000100_create_tags_tables", &createTagsTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createTagsTables struct {
	migration.BaseMigration
}

// Up creates organization-owned tags and their assignments to resources of code-registered types.
// Names are unique per organization ignoring case; the byte-wise collation keeps that rule and the
// list order identical on every PostgreSQL locale. Assignments reference their tag through
// (organization_id, tag_id), so an assignment can never point at another organization's tag.
func (m *createTagsTables) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE tags (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    name varchar(50) NOT NULL,
		    color varchar(16) NOT NULL DEFAULT 'gray',
		    description varchar(200) NOT NULL DEFAULT '',
		    created_by bigint,
		    version bigint NOT NULL DEFAULT 1,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT tags_pkey PRIMARY KEY (id),
		    CONSTRAINT tags_organization_id_id_key UNIQUE (organization_id, id),
		    CONSTRAINT tags_organization_fkey FOREIGN KEY (organization_id)
		        REFERENCES organizations (id) ON DELETE CASCADE,
		    CONSTRAINT tags_created_by_fkey FOREIGN KEY (created_by)
		        REFERENCES users (id) ON DELETE SET NULL,
		    CONSTRAINT tags_name_check CHECK (char_length(name) BETWEEN 1 AND 50 AND name = btrim(name)),
		    CONSTRAINT tags_color_check CHECK (color IN ('gray', 'red', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink')),
		    CONSTRAINT tags_description_check CHECK (char_length(description) <= 200),
		    CONSTRAINT tags_version_check CHECK (version >= 1)
		)`,
		`CREATE UNIQUE INDEX idx_tags_organization_name ON tags (organization_id, (lower(name) COLLATE "C"))`,
		`CREATE TABLE tag_assignments (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    tag_id bigint NOT NULL,
		    resource_type varchar(50) NOT NULL,
		    resource_id varchar(64) NOT NULL,
		    created_at timestamptz NOT NULL,
		    CONSTRAINT tag_assignments_pkey PRIMARY KEY (id),
		    CONSTRAINT tag_assignments_organization_fkey FOREIGN KEY (organization_id)
		        REFERENCES organizations (id) ON DELETE CASCADE,
		    CONSTRAINT tag_assignments_tag_fkey FOREIGN KEY (organization_id, tag_id)
		        REFERENCES tags (organization_id, id) ON DELETE CASCADE,
		    CONSTRAINT tag_assignments_resource_type_check CHECK (resource_type ~ '^[a-z][a-z0-9_]{0,49}$'),
		    CONSTRAINT tag_assignments_resource_id_check CHECK (resource_id ~ '^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$')
		)`,
		`CREATE UNIQUE INDEX idx_tag_assignments_resource_tag
		    ON tag_assignments (organization_id, resource_type, resource_id, tag_id)`,
		`CREATE INDEX idx_tag_assignments_tag ON tag_assignments (organization_id, tag_id, id)`,
	)
}

// Down removes tag assignments and tags.
func (m *createTagsTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS tag_assignments CASCADE`,
		`DROP TABLE IF EXISTS tags CASCADE`,
	)
}
