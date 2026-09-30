package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/capabilities/workflow"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/infra/database"
)

// durableTaskCommand is shared by the operator commands for PostgreSQL-backed workflow tasks.
type durableTaskCommand struct {
	output *console.Output
}

// withDurableDriver opens the PostgreSQL task driver, runs fn, and closes the connection.
func withDurableDriver(fn func(context.Context, *workflow.PostgresDriver) error) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !strings.EqualFold(cfg.Queue.Driver, "postgres") {
		return fmt.Errorf("durable task commands require QUEUE_DRIVER=postgres")
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return loggerErr
	}
	db, err := database.NewDB(cfg)
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("QUEUE_DRIVER=postgres requires DB_ENABLED=true")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	driver, err := workflow.NewPostgresDriver(db)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return fn(ctx, driver)
}

func flagValue(args []string, name string) (string, bool) {
	prefix := "--" + name + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix), true
		}
	}
	return "", false
}

func intFlag(args []string, name string, fallback, minimum, maximum int) (int, error) {
	raw, ok := flagValue(args, name)
	if !ok {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("--%s must be an integer between %d and %d", name, minimum, maximum)
	}
	return value, nil
}

func taskIDArgument(command string, args []string) (string, error) {
	if len(args) != 1 || strings.HasPrefix(args[0], "--") || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("%s requires exactly one task ID", command)
	}
	return strings.TrimSpace(args[0]), nil
}

// WorkflowTasksCommand lists durable tasks for diagnosis.
type WorkflowTasksCommand struct{ durableTaskCommand }

func NewWorkflowTasksCommand() *WorkflowTasksCommand {
	return &WorkflowTasksCommand{durableTaskCommand{output: console.NewOutput()}}
}

func (c *WorkflowTasksCommand) Name() string { return "workflow:tasks" }
func (c *WorkflowTasksCommand) Description() string {
	return "List durable workflow tasks, newest first"
}
func (c *WorkflowTasksCommand) Usage() string {
	return "workflow:tasks [--queue=name] [--status=failed] [--limit=50]"
}

func (c *WorkflowTasksCommand) Run(args []string) error {
	queue, _ := flagValue(args, "queue")
	status, _ := flagValue(args, "status")
	limit, err := intFlag(args, "limit", 50, 1, 200)
	if err != nil {
		return err
	}
	return withDurableDriver(func(ctx context.Context, driver *workflow.PostgresDriver) error {
		tasks, listErr := driver.ListTasks(ctx, queue, status, limit)
		if listErr != nil {
			return listErr
		}
		rows := make([][]string, len(tasks))
		for index, task := range tasks {
			rows[index] = []string{
				task.ID,
				task.Queue,
				task.Status,
				fmt.Sprintf("%d/%d", task.Attempts, task.MaxAttempts),
				task.LastFailureCode,
				task.CreatedAt.UTC().Format(time.RFC3339),
			}
		}
		c.output.Table([]string{"ID", "QUEUE", "STATUS", "ATTEMPTS", "LAST FAILURE", "CREATED"}, rows)
		return nil
	})
}

// WorkflowRetryCommand returns one failed task to its queue with a fresh attempt budget.
type WorkflowRetryCommand struct{ durableTaskCommand }

func NewWorkflowRetryCommand() *WorkflowRetryCommand {
	return &WorkflowRetryCommand{durableTaskCommand{output: console.NewOutput()}}
}

func (c *WorkflowRetryCommand) Name() string        { return "workflow:retry" }
func (c *WorkflowRetryCommand) Description() string { return "Retry one failed durable workflow task" }
func (c *WorkflowRetryCommand) Usage() string       { return "workflow:retry <task-id>" }

func (c *WorkflowRetryCommand) Run(args []string) error {
	taskID, err := taskIDArgument("workflow:retry", args)
	if err != nil {
		return err
	}
	return withDurableDriver(func(ctx context.Context, driver *workflow.PostgresDriver) error {
		switch retryErr := driver.RetryFailed(ctx, taskID); {
		case errors.Is(retryErr, workflow.ErrTaskNotFound):
			return fmt.Errorf("no task %s", taskID)
		case errors.Is(retryErr, workflow.ErrTaskNotRetryable):
			return fmt.Errorf("task %s is not failed; only failed tasks can be retried", taskID)
		case retryErr != nil:
			return retryErr
		}
		slog.InfoContext(ctx, "workflow.task_retried", "task_id", taskID)
		c.output.Success("Task %s is pending again", taskID)
		return nil
	})
}

// WorkflowCancelCommand cancels one pending task or asks a running task to stop.
type WorkflowCancelCommand struct{ durableTaskCommand }

func NewWorkflowCancelCommand() *WorkflowCancelCommand {
	return &WorkflowCancelCommand{durableTaskCommand{output: console.NewOutput()}}
}

func (c *WorkflowCancelCommand) Name() string { return "workflow:cancel" }
func (c *WorkflowCancelCommand) Description() string {
	return "Cancel a pending durable workflow task or request cancellation of a running one"
}
func (c *WorkflowCancelCommand) Usage() string { return "workflow:cancel <task-id>" }

func (c *WorkflowCancelCommand) Run(args []string) error {
	taskID, err := taskIDArgument("workflow:cancel", args)
	if err != nil {
		return err
	}
	return withDurableDriver(func(ctx context.Context, driver *workflow.PostgresDriver) error {
		if cancelErr := driver.Cancel(ctx, taskID); errors.Is(cancelErr, workflow.ErrTaskNotFound) {
			return fmt.Errorf("no pending or running task %s", taskID)
		} else if cancelErr != nil {
			return cancelErr
		}
		slog.InfoContext(ctx, "workflow.task_canceled", "task_id", taskID)
		c.output.Success("Cancellation recorded for task %s", taskID)
		return nil
	})
}

// WorkflowPruneCommand deletes finished tasks past the retention window in bounded batches.
type WorkflowPruneCommand struct{ durableTaskCommand }

func NewWorkflowPruneCommand() *WorkflowPruneCommand {
	return &WorkflowPruneCommand{durableTaskCommand{output: console.NewOutput()}}
}

func (c *WorkflowPruneCommand) Name() string { return "workflow:prune" }
func (c *WorkflowPruneCommand) Description() string {
	return "Delete completed, failed, and canceled workflow tasks past the retention window"
}
func (c *WorkflowPruneCommand) Usage() string {
	return "workflow:prune [--older-than=720h] [--batch=1000]"
}

func (c *WorkflowPruneCommand) Run(args []string) error {
	retention := 30 * 24 * time.Hour
	if raw, ok := flagValue(args, "older-than"); ok {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed < time.Hour {
			return fmt.Errorf("--older-than must be a duration of at least 1h")
		}
		retention = parsed
	}
	batch, err := intFlag(args, "batch", 1000, 1, 10000)
	if err != nil {
		return err
	}
	return withDurableDriver(func(ctx context.Context, driver *workflow.PostgresDriver) error {
		deleted, pruneErr := driver.PruneFinished(ctx, time.Now().Add(-retention), batch)
		if pruneErr != nil {
			return pruneErr
		}
		slog.InfoContext(ctx, "workflow.tasks_pruned", "deleted", deleted, "retention", retention.String())
		c.output.Success("Pruned %d finished task(s) older than %s", deleted, retention)
		return nil
	})
}
