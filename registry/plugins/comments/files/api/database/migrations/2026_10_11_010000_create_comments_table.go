package migrations

import (
	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/infra/migration"
)

func init() {
	register("2026_10_11_010000_create_comments_table", &createCommentsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createCommentsTable struct {
	migration.BaseMigration
}

// Up creates organization-owned comments. Threads are one level deep: a reply's parent is a root
// comment of the same resource, which the application enforces under a row lock. A deleted
// comment keeps its row with body and mentions erased, so the thread keeps its shape.
func (m *createCommentsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE comments (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    resource_type varchar(40) NOT NULL,
		    resource_id varchar(64) NOT NULL,
		    parent_id bigint,
		    author_id bigint,
		    body text,
		    mentions jsonb NOT NULL DEFAULT '[]'::jsonb,
		    version bigint NOT NULL DEFAULT 1,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    edited_at timestamptz,
		    deleted_at timestamptz,
		    CONSTRAINT comments_pkey PRIMARY KEY (id),
		    CONSTRAINT comments_organization_fkey FOREIGN KEY (organization_id)
		        REFERENCES organizations (id) ON DELETE CASCADE,
		    CONSTRAINT comments_author_fkey FOREIGN KEY (author_id)
		        REFERENCES users (id) ON DELETE SET NULL,
		    CONSTRAINT comments_parent_fkey FOREIGN KEY (parent_id)
		        REFERENCES comments (id) ON DELETE CASCADE,
		    CONSTRAINT comments_resource_type_check CHECK (resource_type ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$'),
		    CONSTRAINT comments_resource_id_check CHECK (resource_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$'),
		    CONSTRAINT comments_parent_check CHECK (parent_id IS NULL OR parent_id <> id),
		    CONSTRAINT comments_body_check CHECK (
		        (deleted_at IS NULL AND body IS NOT NULL AND char_length(body) BETWEEN 1 AND 10000)
		        OR (deleted_at IS NOT NULL AND body IS NULL)
		    ),
		    CONSTRAINT comments_mentions_check CHECK (
		        CASE WHEN jsonb_typeof(mentions) = 'array' THEN jsonb_array_length(mentions) <= 20 ELSE false END
		    ),
		    CONSTRAINT comments_version_check CHECK (version >= 1)
		)`,
		`CREATE INDEX idx_comments_resource ON comments (organization_id, resource_type, resource_id, id)`,
		`CREATE INDEX idx_comments_parent_id ON comments (parent_id, id) WHERE parent_id IS NOT NULL`,
		`CREATE INDEX idx_comments_author_id ON comments (author_id) WHERE author_id IS NOT NULL`,
	)
}

// Down removes comments.
func (m *createCommentsTable) Down(db *gorm.DB) error {
	return execStatements(db, `DROP TABLE IF EXISTS comments CASCADE`)
}
