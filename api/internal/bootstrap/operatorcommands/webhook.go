package operatorcommands

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

	"github.com/zgiai/luas/api/internal/app"
	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const (
	defaultWebhookWorkerBatch = 25
	defaultWebhookWorkerPoll  = 2 * time.Second
)

type webhookWorkerConfig struct {
	Batch int
	Poll  time.Duration
	Once  bool
}

type webhookWorker interface {
	DispatchWebhooks(context.Context, int) (int, error)
}

var errWebhookRuntimeUnavailable = errors.New("webhook runtime is unavailable")

// WebhookWorkCommand processes durable outbound deliveries.
type WebhookWorkCommand struct {
	output *console.Output
}

func NewWebhookWorkCommand() *WebhookWorkCommand {
	return &WebhookWorkCommand{output: console.NewOutput()}
}

func (c *WebhookWorkCommand) Name() string { return "webhook:work" }
func (c *WebhookWorkCommand) Description() string {
	return "Process durable outbound webhook deliveries"
}
func (c *WebhookWorkCommand) Usage() string {
	return "webhook:work [--batch=25] [--poll=2s] [--once]"
}

func (c *WebhookWorkCommand) Run(args []string) error {
	workerConfig, err := parseWebhookWorkerArgs(args)
	if err != nil {
		return err
	}
	application, _, err := initWebhookApplication()
	if err != nil {
		return err
	}
	if application.WebhookDispatcher == nil {
		return errWebhookRuntimeUnavailable
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c.output.Info("Starting webhook worker with batch %d and poll interval %s", workerConfig.Batch, workerConfig.Poll)
	processed, err := runWebhookWorker(ctx, application.WebhookDispatcher, workerConfig)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	c.output.Success("Webhook worker stopped after processing %d deliveries", processed)
	return nil
}

func runWebhookWorker(ctx context.Context, worker webhookWorker, cfg webhookWorkerConfig) (int, error) {
	if worker == nil {
		return 0, errWebhookRuntimeUnavailable
	}
	total := 0
	errorDelay := cfg.Poll
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		processed, err := worker.DispatchWebhooks(ctx, cfg.Batch)
		total += processed
		if err != nil {
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			if cfg.Once {
				return total, fmt.Errorf("dispatch webhook batch: %w", err)
			}
			slog.ErrorContext(ctx, "webhook.dispatch_batch_failed")
			if waitErr := waitForWebhookWorker(ctx, errorDelay); waitErr != nil {
				return total, waitErr
			}
			errorDelay = min(errorDelay*2, 30*time.Second)
			continue
		}
		errorDelay = cfg.Poll
		if cfg.Once {
			return total, nil
		}
		if processed == cfg.Batch {
			continue
		}
		if err := waitForWebhookWorker(ctx, cfg.Poll); err != nil {
			return total, err
		}
	}
}

func waitForWebhookWorker(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseWebhookWorkerArgs(args []string) (webhookWorkerConfig, error) {
	cfg := webhookWorkerConfig{Batch: defaultWebhookWorkerBatch, Poll: defaultWebhookWorkerPoll}
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
		default:
			return cfg, fmt.Errorf("unknown webhook worker argument %q", argument)
		}
	}
	if cfg.Batch < 1 || cfg.Batch > 100 {
		return cfg, fmt.Errorf("--batch must be between 1 and 100")
	}
	if cfg.Poll < 100*time.Millisecond || cfg.Poll > time.Minute {
		return cfg, fmt.Errorf("--poll must be between 100ms and 1m")
	}
	return cfg, nil
}

// WebhookPublishTestCommand queues the fixed test event without exposing arbitrary publication.
type WebhookPublishTestCommand struct {
	output *console.Output
}

func NewWebhookPublishTestCommand() *WebhookPublishTestCommand {
	return &WebhookPublishTestCommand{output: console.NewOutput()}
}

func (c *WebhookPublishTestCommand) Name() string { return "webhook:publish-test" }
func (c *WebhookPublishTestCommand) Description() string {
	return "Queue the fixed test event for one webhook endpoint"
}
func (c *WebhookPublishTestCommand) Usage() string {
	return "webhook:publish-test --organization=<id> --endpoint=<id> --actor=<user-id> --idempotency-key=<key>"
}

func (c *WebhookPublishTestCommand) Run(args []string) error {
	values, err := parseWebhookOperatorArgs(args, "organization", "endpoint", "actor", "idempotency-key")
	if err != nil {
		return err
	}
	organizationID, err := webhookUintArgument(values, "organization")
	if err != nil {
		return err
	}
	endpointID, err := webhookUintArgument(values, "endpoint")
	if err != nil {
		return err
	}
	actorID, err := webhookUintArgument(values, "actor")
	if err != nil {
		return err
	}
	application, _, err := initWebhookApplication()
	if err != nil {
		return err
	}
	if application.WebhookTester == nil {
		return errWebhookRuntimeUnavailable
	}
	ctx := context.Background()
	delivery, err := application.WebhookTester.PublishWebhookTest(
		ctx,
		organizationID,
		endpointID,
		actorID,
		values["idempotency-key"],
	)
	if err != nil {
		return err
	}
	recordWebhookCLIAudit(ctx, application, &actorID, "publish_test", endpointID, map[string]any{
		"organization_id": organizationID,
		"delivery_id":     delivery.ID,
		"message_id":      delivery.MessageID,
	})
	c.output.Success("Queued webhook delivery %d with message %s", delivery.ID, delivery.MessageID)
	return nil
}

// WebhookReplayCommand explicitly requeues one retained terminal delivery.
type WebhookReplayCommand struct {
	output *console.Output
}

func NewWebhookReplayCommand() *WebhookReplayCommand {
	return &WebhookReplayCommand{output: console.NewOutput()}
}

func (c *WebhookReplayCommand) Name() string { return "webhook:replay" }
func (c *WebhookReplayCommand) Description() string {
	return "Replay one retained terminal webhook delivery"
}
func (c *WebhookReplayCommand) Usage() string {
	return "webhook:replay --organization=<id> --delivery=<id> --actor=<user-id>"
}

func (c *WebhookReplayCommand) Run(args []string) error {
	values, err := parseWebhookOperatorArgs(args, "organization", "delivery", "actor")
	if err != nil {
		return err
	}
	organizationID, err := webhookUintArgument(values, "organization")
	if err != nil {
		return err
	}
	deliveryID, err := webhookUint64Argument(values, "delivery")
	if err != nil {
		return err
	}
	actorID, err := webhookUintArgument(values, "actor")
	if err != nil {
		return err
	}
	application, _, err := initWebhookApplication()
	if err != nil {
		return err
	}
	if application.WebhookMaintainer == nil {
		return errWebhookRuntimeUnavailable
	}
	ctx := context.Background()
	delivery, err := application.WebhookMaintainer.ReplayWebhookDelivery(ctx, organizationID, deliveryID, actorID)
	if err != nil {
		return err
	}
	recordWebhookCLIAudit(ctx, application, &actorID, "replay", delivery.EndpointID, map[string]any{
		"organization_id": organizationID,
		"delivery_id":     delivery.ID,
		"message_id":      delivery.MessageID,
		"replay_count":    delivery.ReplayCount,
	})
	c.output.Success("Requeued webhook delivery %d with message %s", delivery.ID, delivery.MessageID)
	return nil
}

// WebhookPruneCommand removes bounded terminal history beyond retention.
type WebhookPruneCommand struct {
	output *console.Output
}

func NewWebhookPruneCommand() *WebhookPruneCommand {
	return &WebhookPruneCommand{output: console.NewOutput()}
}

func (c *WebhookPruneCommand) Name() string        { return "webhook:prune" }
func (c *WebhookPruneCommand) Description() string { return "Prune retained terminal webhook history" }
func (c *WebhookPruneCommand) Usage() string       { return "webhook:prune [--before=<RFC3339>]" }

func (c *WebhookPruneCommand) Run(args []string) error {
	values, err := parseOptionalWebhookBefore(args)
	if err != nil {
		return err
	}
	application, cfg, err := initWebhookApplication()
	if err != nil {
		return err
	}
	if application.WebhookMaintainer == nil {
		return errWebhookRuntimeUnavailable
	}
	before := time.Now().UTC().Add(-cfg.Webhook.EventRetention)
	if values != "" {
		before, err = time.Parse(time.RFC3339, values)
		if err != nil {
			return fmt.Errorf("invalid --before value %q: %w", values, err)
		}
	}
	ctx := context.Background()
	result, err := application.WebhookMaintainer.PruneWebhookHistory(ctx, before)
	if err != nil {
		return err
	}
	recordWebhookCLIAudit(ctx, application, nil, "prune", 0, map[string]any{
		"before":     before.UTC(),
		"attempts":   result.Attempts,
		"deliveries": result.Deliveries,
		"events":     result.Events,
		"secrets":    result.Secrets,
	})
	c.output.Success(
		"Pruned %d attempts, %d deliveries, %d events, and %d expired secrets",
		result.Attempts,
		result.Deliveries,
		result.Events,
		result.Secrets,
	)
	return nil
}

func initWebhookApplication() (*app.Application, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	if !slices.Contains(cfg.Starters.Optional, "webhook") {
		return nil, nil, fmt.Errorf("webhook starter is not selected in OPTIONAL_STARTERS")
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return nil, nil, loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize webhook runtime: %w", err)
	}
	return application, cfg, nil
}

func parseWebhookOperatorArgs(args []string, required ...string) (map[string]string, error) {
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
	}
	values := make(map[string]string, len(required))
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, fmt.Errorf("unknown webhook operator argument %q", argument)
		}
		keyValue := strings.SplitN(strings.TrimPrefix(argument, "--"), "=", 2)
		key := keyValue[0]
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("unknown webhook operator argument %q", argument)
		}
		value := ""
		if len(keyValue) == 2 {
			value = keyValue[1]
		} else if index+1 < len(args) {
			index++
			value = args[index]
		}
		if value == "" || value != strings.TrimSpace(value) {
			return nil, fmt.Errorf("--%s requires a canonical value", key)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("--%s may be provided only once", key)
		}
		values[key] = value
	}
	for _, key := range required {
		if values[key] == "" {
			return nil, fmt.Errorf("--%s is required", key)
		}
	}
	return values, nil
}

func parseOptionalWebhookBefore(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	values, err := parseWebhookOperatorArgs(args, "before")
	if err != nil {
		return "", err
	}
	return values["before"], nil
}

func webhookUintArgument(values map[string]string, key string) (uint, error) {
	parsed, err := strconv.ParseUint(values[key], 10, strconv.IntSize)
	if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != values[key] {
		return 0, fmt.Errorf("--%s must be a positive canonical integer", key)
	}
	return uint(parsed), nil
}

func webhookUint64Argument(values map[string]string, key string) (uint64, error) {
	parsed, err := strconv.ParseUint(values[key], 10, 64)
	if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != values[key] {
		return 0, fmt.Errorf("--%s must be a positive canonical integer", key)
	}
	return parsed, nil
}

func recordWebhookCLIAudit(
	ctx context.Context,
	application *app.Application,
	actorID *uint,
	action string,
	endpointID uint,
	metadata map[string]any,
) {
	if application == nil || application.AuditRecorder == nil {
		return
	}
	actorType := domain.AuditActorSystem
	if actorID != nil {
		actorType = domain.AuditActorUser
	}
	targetID := ""
	if endpointID != 0 {
		targetID = strconv.FormatUint(uint64(endpointID), 10)
	}
	err := application.AuditRecorder.Record(ctx, &domain.AuditLog{
		UserID:     actorID,
		ActorType:  actorType,
		ActorID:    actorID,
		Action:     action,
		Resource:   "webhook-endpoints",
		TargetType: "webhook_endpoint",
		TargetID:   targetID,
		Result:     domain.AuditResultSuccess,
		Method:     "CLI",
		Path:       "webhook:" + action,
		StatusCode: 200,
		Metadata:   metadata,
	})
	if err != nil {
		slog.WarnContext(ctx, "webhook.operator_audit_failed", "action", action)
	}
}
