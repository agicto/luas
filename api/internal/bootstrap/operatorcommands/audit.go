package operatorcommands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

const (
	defaultAuditPruneBatch = 500
	maxAuditPruneBatch     = 10_000
	auditPruneTimeout      = 30 * time.Second
)

var errAuditMaintainerUnavailable = errors.New("audit log maintainer is unavailable")

type auditPruneArguments struct {
	before time.Time
	batch  int
}

// AuditPruneCommand removes one bounded batch of audit history before an explicit cutoff.
type AuditPruneCommand struct {
	output *console.Output
}

func NewAuditPruneCommand() *AuditPruneCommand {
	return &AuditPruneCommand{output: console.NewOutput()}
}

func (c *AuditPruneCommand) Name() string { return "audit:prune" }
func (c *AuditPruneCommand) Description() string {
	return "Prune one bounded batch of audit logs before a cutoff"
}
func (c *AuditPruneCommand) Usage() string {
	return "audit:prune --before=<RFC3339> [--batch=500]"
}

func (c *AuditPruneCommand) Run(args []string) error {
	parsed, err := parseAuditPruneArguments(args)
	if err != nil {
		return err
	}
	if !parsed.before.Before(time.Now().UTC()) {
		return errors.New("--before must be in the past")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("initialize audit cleanup: %w", err)
	}
	if application.AuditMaintainer == nil || application.AuditRecorder == nil {
		return errAuditMaintainerUnavailable
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pruneCtx, cancelPrune := context.WithTimeout(ctx, auditPruneTimeout)
	count, err := application.AuditMaintainer.PruneAuditLogs(pruneCtx, parsed.before, parsed.batch)
	cancelPrune()
	if err != nil {
		return err
	}
	if auditErr := recordAuditPrune(ctx, application.AuditRecorder, parsed, count); auditErr != nil {
		c.output.Warning("Audit logs pruned, but the operator audit record failed")
	}
	c.output.Success("Pruned %d audit log(s) before %s", count, parsed.before.Format(time.RFC3339))
	return nil
}

func parseAuditPruneArguments(args []string) (*auditPruneArguments, error) {
	values := map[string]string{}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, fmt.Errorf("unknown audit prune argument %q", argument)
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		if !hasValue {
			if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
				return nil, fmt.Errorf("audit prune flag --%s requires a value", name)
			}
			index++
			value = args[index]
		}
		if name != "before" && name != "batch" {
			return nil, fmt.Errorf("unknown audit prune flag --%s", name)
		}
		if _, exists := values[name]; exists {
			return nil, fmt.Errorf("duplicate audit prune flag --%s", name)
		}
		values[name] = value
	}

	beforeValue, ok := values["before"]
	if !ok {
		return nil, errors.New("audit prune flag --before is required")
	}
	before, err := time.Parse(time.RFC3339, beforeValue)
	if err != nil {
		return nil, fmt.Errorf("invalid --before value %q: %w", beforeValue, err)
	}
	batch := defaultAuditPruneBatch
	if batchValue, ok := values["batch"]; ok {
		batch, err = strconv.Atoi(batchValue)
		if err != nil {
			return nil, fmt.Errorf("invalid --batch value %q: %w", batchValue, err)
		}
	}
	if batch < 1 || batch > maxAuditPruneBatch {
		return nil, fmt.Errorf("--batch must be between 1 and %d", maxAuditPruneBatch)
	}
	return &auditPruneArguments{before: before.UTC(), batch: batch}, nil
}

func recordAuditPrune(
	ctx context.Context,
	recorder domain.AuditLogRecorder,
	parsed *auditPruneArguments,
	count int64,
) error {
	if recorder == nil || parsed == nil {
		return errAuditMaintainerUnavailable
	}
	return recorder.Record(ctx, &domain.AuditLog{
		ActorType:  domain.AuditActorSystem,
		Action:     "prune",
		Resource:   "audit_logs",
		TargetType: "audit_retention_cutoff",
		TargetID:   parsed.before.Format(time.RFC3339),
		Result:     domain.AuditResultSuccess,
		Method:     "COMMAND",
		Path:       "audit:prune",
		RouteName:  "console.audit.prune",
		StatusCode: 200,
		Metadata: map[string]any{
			"batch":   parsed.batch,
			"deleted": count,
		},
	})
}
