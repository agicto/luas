package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

func init() {
	register("2025_06_18_000000_create_users_table", &createUsersTable{})
}

type createUsersTable struct {
	migration.BaseMigration
}

// Up creates the users table.
func (m *createUsersTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE users (
		    id bigserial NOT NULL,
		    created_at timestamptz,
		    updated_at timestamptz,
		    deleted_at timestamptz,
		    username varchar(50) NOT NULL,
		    password varchar(100) NOT NULL,
		    email varchar(100) NOT NULL,
		    nickname varchar(50),
		    avatar varchar(255),
		    phone varchar(20),
		    bio varchar(500),
		    status bigint DEFAULT 1,
		    last_login timestamptz,
		    CONSTRAINT uni_users_email UNIQUE (email),
		    CONSTRAINT users_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_users_deleted_at ON users (deleted_at)`,
		`CREATE UNIQUE INDEX idx_users_username ON users (username)`,
	)
}

// Down drops the users table.
func (m *createUsersTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS users CASCADE`,
	)
}
