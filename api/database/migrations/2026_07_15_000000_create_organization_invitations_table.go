package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2026_07_15_000000_create_organization_invitations_table", &createOrganizationInvitationsTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createOrganizationInvitationsTable struct {
	migration.BaseMigration
}

// Up adds the invitation lifecycle without changing the deployed ownership tables.
func (m *createOrganizationInvitationsTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE organization_invitations (
		    id bigserial NOT NULL,
		    organization_id bigint NOT NULL,
		    invited_by bigint NOT NULL,
		    email varchar(100) NOT NULL,
		    role varchar(16) NOT NULL,
		    token_hash varchar(64) NOT NULL,
		    pending_key varchar(64),
		    expires_at timestamptz NOT NULL,
		    accepted_at timestamptz,
		    accepted_by bigint,
		    revoked_at timestamptz,
		    revoked_by bigint,
		    created_at timestamptz,
		    updated_at timestamptz,
		    CONSTRAINT organization_invitations_role_check CHECK ((role IN ('admin', 'member'))),
		    CONSTRAINT organization_invitations_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_organization_invitations_accepted_at ON organization_invitations (accepted_at)`,
		`CREATE INDEX idx_organization_invitations_accepted_by ON organization_invitations (accepted_by)`,
		`CREATE INDEX idx_organization_invitations_expires_at ON organization_invitations (expires_at)`,
		`CREATE INDEX idx_organization_invitations_invited_by ON organization_invitations (invited_by)`,
		`CREATE INDEX idx_organization_invitations_org_created ON organization_invitations (organization_id, created_at)`,
		`CREATE INDEX idx_organization_invitations_org_email ON organization_invitations (organization_id, email)`,
		`CREATE UNIQUE INDEX idx_organization_invitations_pending_key ON organization_invitations (pending_key)`,
		`CREATE INDEX idx_organization_invitations_revoked_at ON organization_invitations (revoked_at)`,
		`CREATE INDEX idx_organization_invitations_revoked_by ON organization_invitations (revoked_by)`,
		`CREATE UNIQUE INDEX idx_organization_invitations_token_hash ON organization_invitations (token_hash)`,
		`ALTER TABLE organization_invitations ADD CONSTRAINT fk_organization_invitations_acceptor FOREIGN KEY (accepted_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE organization_invitations ADD CONSTRAINT fk_organization_invitations_inviter FOREIGN KEY (invited_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
		`ALTER TABLE organization_invitations ADD CONSTRAINT fk_organization_invitations_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON UPDATE CASCADE ON DELETE CASCADE`,
		`ALTER TABLE organization_invitations ADD CONSTRAINT fk_organization_invitations_revoker FOREIGN KEY (revoked_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT`,
	)
}

// Down removes only the additive invitation table.
func (m *createOrganizationInvitationsTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS organization_invitations CASCADE`,
	)
}
