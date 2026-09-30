package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/infra/migration"
)

const workflowTasksMigration = "2026_07_31_000000_create_workflow_tasks_table"

func init() {
	register(workflowTasksMigration, &createWorkflowTasksTable{
		BaseMigration: migration.BaseMigration{UseTransaction: true},
	})
}

type createWorkflowTasksTable struct {
	migration.BaseMigration
}

// Up creates the durable workflow task table and its idempotency index.
func (m *createWorkflowTasksTable) Up(db *gorm.DB) error {
	return execStatements(db,
		`CREATE TABLE workflow_tasks (
		    id uuid NOT NULL,
		    queue varchar(128) NOT NULL,
		    idempotency_key varchar(128),
		    payload jsonb NOT NULL,
		    payload_hash varchar(64) NOT NULL,
		    status varchar(20) NOT NULL,
		    attempts bigint DEFAULT 0 NOT NULL,
		    max_attempts bigint NOT NULL,
		    available_at timestamptz NOT NULL,
		    lease_token varchar(64) DEFAULT ''::varchar NOT NULL,
		    lease_expires_at timestamptz,
		    fencing_token bigint DEFAULT 0 NOT NULL,
		    cancel_requested_at timestamptz,
		    completed_at timestamptz,
		    failed_at timestamptz,
		    last_failure_code varchar(64) DEFAULT ''::varchar NOT NULL,
		    created_at timestamptz NOT NULL,
		    updated_at timestamptz NOT NULL,
		    CONSTRAINT workflow_tasks_status_check CHECK ((status IN ('pending', 'processing', 'completed', 'failed', 'canceled'))),
		    CONSTRAINT workflow_tasks_pkey PRIMARY KEY (id)
		)`,
		`CREATE INDEX idx_workflow_tasks_claim ON workflow_tasks (queue, status, available_at)`,
		`CREATE INDEX idx_workflow_tasks_lease ON workflow_tasks (status, lease_expires_at)`,
		`CREATE UNIQUE INDEX idx_workflow_tasks_queue_idempotency ON workflow_tasks (queue, idempotency_key) WHERE (idempotency_key IS NOT NULL)`,
	)
}

// Down drops the durable workflow task table.
func (m *createWorkflowTasksTable) Down(db *gorm.DB) error {
	return execStatements(db,
		`DROP TABLE IF EXISTS workflow_tasks CASCADE`,
	)
}
