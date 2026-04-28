package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zgiai/zgo/internal/bootstrap"
	"github.com/zgiai/zgo/internal/capabilities/workflow"
	"github.com/zgiai/zgo/internal/infra/config"
	"github.com/zgiai/zgo/internal/infra/console"
	"github.com/zgiai/zgo/internal/infra/queue"
)

// WorkflowWorkCommand runs a workflow queue worker.
type WorkflowWorkCommand struct {
	output *console.Output
}

// WorkflowScheduleRunCommand runs due scheduled workflow tasks once.
type WorkflowScheduleRunCommand struct {
	output *console.Output
}

// WorkflowScheduleWorkCommand runs the workflow scheduler loop.
type WorkflowScheduleWorkCommand struct {
	output *console.Output
}

func NewWorkflowWorkCommand() *WorkflowWorkCommand {
	return &WorkflowWorkCommand{output: console.NewOutput()}
}

func NewWorkflowScheduleRunCommand() *WorkflowScheduleRunCommand {
	return &WorkflowScheduleRunCommand{output: console.NewOutput()}
}

func NewWorkflowScheduleWorkCommand() *WorkflowScheduleWorkCommand {
	return &WorkflowScheduleWorkCommand{output: console.NewOutput()}
}

func (c *WorkflowWorkCommand) Name() string        { return "workflow:work" }
func (c *WorkflowWorkCommand) Description() string { return "Run a background workflow worker" }
func (c *WorkflowWorkCommand) Usage() string {
	return "workflow:work [--queue=default] [--concurrency=1] [--max-jobs=0]"
}

func (c *WorkflowScheduleRunCommand) Name() string        { return "workflow:schedule:run" }
func (c *WorkflowScheduleRunCommand) Description() string { return "Run due workflow schedules once" }
func (c *WorkflowScheduleRunCommand) Usage() string       { return "workflow:schedule:run" }

func (c *WorkflowScheduleWorkCommand) Name() string        { return "workflow:schedule:work" }
func (c *WorkflowScheduleWorkCommand) Description() string { return "Run the workflow scheduler loop" }
func (c *WorkflowScheduleWorkCommand) Usage() string       { return "workflow:schedule:work" }

func (c *WorkflowWorkCommand) Run(args []string) error {
	bootstrap.InitLogger()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	manager, err := workflow.Bootstrap(cfg)
	if err != nil {
		return err
	}

	workerCfg := workflow.WorkerConfig{
		Queue:       cfg.Queue.DefaultQueue,
		Concurrency: cfg.Queue.WorkerConcurrency,
		Sleep:       cfg.Queue.WorkerSleep,
		Timeout:     cfg.Queue.WorkerTimeout,
	}
	if err := applyWorkflowWorkerArgs(args, &workerCfg); err != nil {
		return err
	}
	if strings.TrimSpace(workerCfg.Queue) == "" {
		workerCfg.Queue = "default"
	}

	driverName := queue.Global().DefaultDriverName()
	if driverName == "sync" {
		c.output.Warning("QUEUE_DRIVER=sync executes jobs inline; use QUEUE_DRIVER=memory for a long-running worker")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c.output.Info("Starting workflow worker on queue %q with driver %q", workerCfg.Queue, driverName)
	worker, err := manager.StartWorker(ctx, workerCfg)
	if err != nil {
		return fmt.Errorf("failed to start workflow worker: %w", err)
	}

	worker.Wait()
	c.output.Success("Workflow worker stopped")
	return nil
}

func (c *WorkflowScheduleRunCommand) Run(args []string) error {
	bootstrap.InitLogger()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	manager, err := workflow.Bootstrap(cfg)
	if err != nil {
		return err
	}

	if !cfg.Scheduler.Enabled {
		c.output.Warning("SCHEDULER_ENABLED=false; running due schedules because command was invoked explicitly")
	}

	if err := manager.RunDue(context.Background(), time.Now()); err != nil {
		return fmt.Errorf("failed to run due workflow schedules: %w", err)
	}

	c.output.Success("Workflow schedules run completed")
	return nil
}

func (c *WorkflowScheduleWorkCommand) Run(args []string) error {
	bootstrap.InitLogger()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	manager, err := workflow.Bootstrap(cfg)
	if err != nil {
		return err
	}

	if !cfg.Scheduler.Enabled {
		c.output.Warning("SCHEDULER_ENABLED=false; starting scheduler because command was invoked explicitly")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c.output.Info("Starting workflow scheduler loop")
	manager.StartScheduler(ctx)
	c.output.Success("Workflow scheduler stopped")
	return nil
}

func applyWorkflowWorkerArgs(args []string, cfg *workflow.WorkerConfig) error {
	if cfg == nil {
		return nil
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--queue" && i+1 < len(args):
			cfg.Queue = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--queue="):
			cfg.Queue = strings.TrimSpace(strings.TrimPrefix(arg, "--queue="))

		case arg == "--concurrency" && i+1 < len(args):
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid --concurrency value %q: %w", args[i+1], err)
			}
			cfg.Concurrency = value
			i++
		case strings.HasPrefix(arg, "--concurrency="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--concurrency="))
			if err != nil {
				return fmt.Errorf("invalid --concurrency value: %w", err)
			}
			cfg.Concurrency = value

		case arg == "--max-jobs" && i+1 < len(args):
			value, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid --max-jobs value %q: %w", args[i+1], err)
			}
			cfg.MaxJobs = value
			i++
		case strings.HasPrefix(arg, "--max-jobs="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--max-jobs="))
			if err != nil {
				return fmt.Errorf("invalid --max-jobs value: %w", err)
			}
			cfg.MaxJobs = value
		}
	}

	return nil
}
