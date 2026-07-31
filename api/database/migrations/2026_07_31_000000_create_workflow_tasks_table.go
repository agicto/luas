package migrations

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/capabilities/workflow"
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

func (m *createWorkflowTasksTable) Up(db *gorm.DB) error {
	if err := db.AutoMigrate(&workflow.TaskPO{}); err != nil {
		return err
	}
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_tasks_queue_idempotency
		ON workflow_tasks (queue, idempotency_key)
		WHERE idempotency_key IS NOT NULL
	`).Error
}

func (m *createWorkflowTasksTable) Down(db *gorm.DB) error {
	return db.Migrator().DropTable(&workflow.TaskPO{})
}
