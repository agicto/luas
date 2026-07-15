package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const (
	defaultNotificationWorkerBatch = 25
	defaultNotificationWorkerPoll  = 2 * time.Second
)

type notificationWorkerConfig struct {
	Batch       int
	Poll        time.Duration
	MaxAttempts int
	Once        bool
}

type notificationDispatcher interface {
	DispatchDue(context.Context, int) (int, error)
}

var errNotificationDispatcherUnavailable = errors.New("notification dispatcher is unavailable")

// NotificationWorkCommand processes durable notification channel deliveries.
type NotificationWorkCommand struct {
	output *console.Output
}

func NewNotificationWorkCommand() *NotificationWorkCommand {
	return &NotificationWorkCommand{output: console.NewOutput()}
}

func (c *NotificationWorkCommand) Name() string { return "notification:work" }
func (c *NotificationWorkCommand) Description() string {
	return "Process durable notification deliveries"
}
func (c *NotificationWorkCommand) Usage() string {
	return "notification:work [--batch=25] [--poll=2s] [--max-attempts=0] [--once]"
}

func (c *NotificationWorkCommand) Run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.Starters.Optional, "notification") {
		return fmt.Errorf("notification starter is not selected in OPTIONAL_STARTERS")
	}
	workerConfig, err := parseNotificationWorkerArgs(args)
	if err != nil {
		return err
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("initialize notification worker: %w", err)
	}
	if application.NotificationDispatcher == nil {
		return errNotificationDispatcherUnavailable
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c.output.Info(
		"Starting notification worker with batch %d and poll interval %s",
		workerConfig.Batch,
		workerConfig.Poll,
	)

	processed, err := runNotificationWorker(ctx, application.NotificationDispatcher, workerConfig)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	c.output.Success("Notification worker stopped after processing %d deliveries", processed)
	return nil
}

func runNotificationWorker(
	ctx context.Context,
	dispatcher notificationDispatcher,
	cfg notificationWorkerConfig,
) (int, error) {
	if dispatcher == nil {
		return 0, errNotificationDispatcherUnavailable
	}
	total := 0
	errorDelay := cfg.Poll
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		limit := cfg.Batch
		if cfg.MaxAttempts > 0 && cfg.MaxAttempts-total < limit {
			limit = cfg.MaxAttempts - total
		}
		if limit <= 0 {
			return total, nil
		}

		processed, err := dispatcher.DispatchDue(ctx, limit)
		total += processed
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return total, contextErr
			}
			if errors.Is(err, context.Canceled) {
				return total, err
			}
			if cfg.Once || (cfg.MaxAttempts > 0 && total >= cfg.MaxAttempts) {
				return total, fmt.Errorf("dispatch notification batch: %w", err)
			}
			slog.ErrorContext(ctx, "notification.dispatch_batch_failed")
			if waitErr := waitForNotificationWorker(ctx, errorDelay); waitErr != nil {
				return total, waitErr
			}
			errorDelay = min(errorDelay*2, 30*time.Second)
			continue
		}
		errorDelay = cfg.Poll
		if cfg.Once || (cfg.MaxAttempts > 0 && total >= cfg.MaxAttempts) {
			return total, nil
		}
		if processed == limit {
			continue
		}
		if err := waitForNotificationWorker(ctx, cfg.Poll); err != nil {
			return total, err
		}
	}
}

func waitForNotificationWorker(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseNotificationWorkerArgs(args []string) (notificationWorkerConfig, error) {
	cfg := notificationWorkerConfig{
		Batch: defaultNotificationWorkerBatch,
		Poll:  defaultNotificationWorkerPoll,
	}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--once":
			cfg.Once = true
		case argument == "--batch" && index+1 < len(args):
			index++
			value, err := strconv.Atoi(args[index])
			if err != nil {
				return cfg, fmt.Errorf("invalid --batch value %q: %w", args[index], err)
			}
			cfg.Batch = value
		case strings.HasPrefix(argument, "--batch="):
			value, err := strconv.Atoi(strings.TrimPrefix(argument, "--batch="))
			if err != nil {
				return cfg, fmt.Errorf("invalid --batch value: %w", err)
			}
			cfg.Batch = value
		case argument == "--poll" && index+1 < len(args):
			index++
			value, err := time.ParseDuration(args[index])
			if err != nil {
				return cfg, fmt.Errorf("invalid --poll value %q: %w", args[index], err)
			}
			cfg.Poll = value
		case strings.HasPrefix(argument, "--poll="):
			value, err := time.ParseDuration(strings.TrimPrefix(argument, "--poll="))
			if err != nil {
				return cfg, fmt.Errorf("invalid --poll value: %w", err)
			}
			cfg.Poll = value
		case argument == "--max-attempts" && index+1 < len(args):
			index++
			value, err := strconv.Atoi(args[index])
			if err != nil {
				return cfg, fmt.Errorf("invalid --max-attempts value %q: %w", args[index], err)
			}
			cfg.MaxAttempts = value
		case strings.HasPrefix(argument, "--max-attempts="):
			value, err := strconv.Atoi(strings.TrimPrefix(argument, "--max-attempts="))
			if err != nil {
				return cfg, fmt.Errorf("invalid --max-attempts value: %w", err)
			}
			cfg.MaxAttempts = value
		default:
			return cfg, fmt.Errorf("unknown notification worker argument %q", argument)
		}
	}
	if cfg.Batch < 1 || cfg.Batch > 100 {
		return cfg, fmt.Errorf("--batch must be between 1 and 100")
	}
	if cfg.Poll < 100*time.Millisecond || cfg.Poll > time.Minute {
		return cfg, fmt.Errorf("--poll must be between 100ms and 1m")
	}
	if cfg.MaxAttempts < 0 {
		return cfg, fmt.Errorf("--max-attempts cannot be negative")
	}
	return cfg, nil
}
