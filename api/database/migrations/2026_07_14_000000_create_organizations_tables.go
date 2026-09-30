package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_14_000000_create_organizations_tables", &createOrganizationsTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createOrganizationsTables struct {
	migration.BaseMigration
}

// Up creates organizations before their memberships.
func (m *createOrganizationsTables) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE organizations (
		    id bigserial NOT NULL,
		    name varchar(100) NOT NULL,
		    slug varchar(50) NOT NULL,
		    created_by bigint NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT organizations_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_organizations_created_by ON organizations (created_by)`,
		`CREATE UNIQUE INDEX idx_organizations_slug ON organizations (slug)`,
		`CREATE TABLE organization_memberships (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    user_id bigint NOT NULL,
		    role varchar(16) NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT organization_memberships_role_check CHECK ((role IN ('owner', 'admin', 'member'))),
		    CONSTRAINT organization_memberships_pkey PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX idx_organization_memberships_org_user ON organization_memberships (organization_id, user_id)`,
		`CREATE INDEX idx_organization_memberships_user_created ON organization_memberships (user_id, created_at)`,
		`CREATE INDEX idx_organization_memberships_user_role ON organization_memberships (user_id, role)`,
		`ALTER TABLE organizations ADD CONSTRAINT fk_organizations_creator FOREIGN KEY (created_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE organization_memberships ADD CONSTRAINT fk_organization_memberships_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE organization_memberships ADD CONSTRAINT fk_organization_memberships_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
	)
}

// Down removes memberships before their parent organizations.
func (m *createOrganizationsTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS organization_memberships CASCADE`,
		`DROP TABLE IF EXISTS organizations CASCADE`,
	)
}
