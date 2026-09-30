package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_15_010000_create_permission_tables", &createPermissionTables{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createPermissionTables struct {
	migration.BaseMigration
}

// Up creates roles before their grants and membership assignments.
func (m *createPermissionTables) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE permission_roles (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    name varchar(100) NOT NULL,
		    slug varchar(50) NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT permission_roles_pkey PRIMARY KEY (id)
		)`,
		`CREATE UNIQUE INDEX idx_permission_roles_org_slug ON permission_roles (organization_id, slug)`,
		`CREATE INDEX idx_permission_roles_organization_id ON permission_roles (organization_id)`,
		`CREATE TABLE permission_role_grants (
		    access_role_id bigint NOT NULL,
		    permission varchar(100) NOT NULL,
		    created_at timestamptz,
		    CONSTRAINT permission_role_grants_pkey PRIMARY KEY (access_role_id, permission)
		)`,
		`CREATE INDEX idx_permission_role_grants_permission_role ON permission_role_grants (permission, access_role_id)`,
		`CREATE TABLE permission_role_assignments (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    membership_id bigint NOT NULL,
		    access_role_id bigint NOT NULL,
		    created_at timestamptz,
		    CONSTRAINT permission_role_assignments_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_permission_role_assignments_access_role_id ON permission_role_assignments (access_role_id)`,
		`CREATE INDEX idx_permission_role_assignments_org_member ON permission_role_assignments (organization_id, membership_id)`,
		`CREATE UNIQUE INDEX idx_permission_role_assignments_org_member_role ON permission_role_assignments (organization_id, membership_id, access_role_id)`,
		`ALTER TABLE permission_roles ADD CONSTRAINT fk_permission_roles_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE permission_role_grants ADD CONSTRAINT fk_permission_roles_permissions FOREIGN KEY (access_role_id) REFERENCES permission_roles(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE permission_role_assignments ADD CONSTRAINT fk_permission_role_assignments_access_role FOREIGN KEY (access_role_id) REFERENCES permission_roles(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE permission_role_assignments ADD CONSTRAINT fk_permission_role_assignments_membership FOREIGN KEY (membership_id) REFERENCES organization_memberships(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE permission_role_assignments ADD CONSTRAINT fk_permission_role_assignments_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
	)
}

// Down removes assignments and grants before their parent roles.
func (m *createPermissionTables) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS permission_role_assignments CASCADE`,
		`DROP TABLE IF EXISTS permission_role_grants CASCADE`,
		`DROP TABLE IF EXISTS permission_roles CASCADE`,
	)
}
