package operatorcommands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

var errUsageCommandUnavailable = errors.New("usage command service is unavailable")

type usageCommandRuntime struct {
	reader     domain.UsageReader
	recorder   domain.UsageRecorder
	consumer   domain.UsageConsumer
	quota      domain.UsageQuotaWriter
	maintainer domain.UsageMaintainer
	audit      domain.AuditLogRecorder
}

// UsageListCommand lists finite current-period usage for one subject.
type UsageListCommand struct{ output *console.Output }

func NewUsageListCommand() *UsageListCommand {
	return &UsageListCommand{output: console.NewOutput()}
}

func (c *UsageListCommand) Name() string        { return "usage:list" }
func (c *UsageListCommand) Description() string { return "List current usage for one subject" }
func (c *UsageListCommand) Usage() string {
	return "usage:list --scope=<user|organization> --subject-id=<id>"
}

func (c *UsageListCommand) Run(args []string) error {
	parsed, err := parseUsageTargetArguments(c.Name(), args)
	if err != nil {
		return err
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	values, err := runtime.reader.ListUsage(ctx, parsed.target)
	if err != nil {
		return err
	}
	rows := make([][]string, len(values))
	for index, value := range values {
		rows[index] = []string{
			value.Metric,
			value.Unit,
			string(value.Period),
			strconv.FormatInt(value.Used, 10),
			formatUsageLimit(value.Limit),
			formatUsageLimit(value.Remaining),
			string(value.QuotaSource),
			strconv.FormatUint(value.QuotaVersion, 10),
		}
	}
	c.output.Table(
		[]string{"METRIC", "UNIT", "PERIOD", "USED", "LIMIT", "REMAINING", "QUOTA_SOURCE", "QUOTA_VERSION"},
		rows,
	)
	return nil
}

// UsageRecordCommand records one occurred fact or correction.
type UsageRecordCommand struct{ output *console.Output }

func NewUsageRecordCommand() *UsageRecordCommand {
	return &UsageRecordCommand{output: console.NewOutput()}
}

func (c *UsageRecordCommand) Name() string        { return "usage:record" }
func (c *UsageRecordCommand) Description() string { return "Record one idempotent usage fact" }
func (c *UsageRecordCommand) Usage() string {
	return "usage:record --scope=<scope> --subject-id=<id> --metric=<key> --quantity=<n> --source=<source> --event-id=<id> --occurred-at=<RFC3339> [--dimensions=<json>]"
}

func (c *UsageRecordCommand) Run(args []string) error {
	parsed, err := parseUsageMutationArguments(c.Name(), args, true)
	if err != nil {
		return err
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	receipt, err := runtime.recorder.RecordUsage(ctx, domain.UsageEvent{
		Source:     parsed.source,
		EventID:    parsed.eventID,
		Target:     parsed.target,
		Metric:     parsed.metric,
		Quantity:   parsed.quantity,
		Dimensions: parsed.dimensions,
		OccurredAt: parsed.occurredAt,
	})
	if err != nil {
		return err
	}
	if !receipt.Replayed {
		c.warnAuditFailure(recordUsageOperatorAudit(ctx, runtime.audit, c.Name(), receipt, domain.AuditResultSuccess))
	}
	c.output.Success(
		"Recorded %s quantity %d; counter %d (%s)",
		receipt.Metric,
		receipt.Quantity,
		receipt.CounterAfter,
		usageReplayLabel(receipt.Replayed),
	)
	return nil
}

// UsageConsumeCommand performs one server-timed atomic hard-quota decision.
type UsageConsumeCommand struct{ output *console.Output }

func NewUsageConsumeCommand() *UsageConsumeCommand {
	return &UsageConsumeCommand{output: console.NewOutput()}
}

func (c *UsageConsumeCommand) Name() string        { return "usage:consume" }
func (c *UsageConsumeCommand) Description() string { return "Consume usage against a hard quota" }
func (c *UsageConsumeCommand) Usage() string {
	return "usage:consume --scope=<scope> --subject-id=<id> --metric=<key> --quantity=<n> --source=<source> --event-id=<id> [--dimensions=<json>]"
}

func (c *UsageConsumeCommand) Run(args []string) error {
	parsed, err := parseUsageMutationArguments(c.Name(), args, false)
	if err != nil {
		return err
	}
	if parsed.quantity < 0 {
		return fmt.Errorf("--quantity must be positive for usage:consume")
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	receipt, consumeErr := runtime.consumer.ConsumeUsage(ctx, domain.UsageConsumption{
		Source:     parsed.source,
		EventID:    parsed.eventID,
		Target:     parsed.target,
		Metric:     parsed.metric,
		Quantity:   parsed.quantity,
		Dimensions: parsed.dimensions,
	})
	if receipt != nil && !receipt.Replayed {
		result := domain.AuditResultSuccess
		if consumeErr != nil {
			result = domain.AuditResultFailure
		}
		c.warnAuditFailure(recordUsageOperatorAudit(ctx, runtime.audit, c.Name(), receipt, result))
	}
	if consumeErr != nil {
		if receipt != nil && errors.Is(consumeErr, domain.ErrUsageQuotaExceeded) {
			c.output.Warning(
				"Denied %s quantity %d at counter %d with limit %s",
				receipt.Metric,
				receipt.Quantity,
				receipt.CounterAfter,
				formatUsageLimit(receipt.Limit),
			)
		}
		return consumeErr
	}
	c.output.Success(
		"Consumed %s quantity %d; counter %d (%s)",
		receipt.Metric,
		receipt.Quantity,
		receipt.CounterAfter,
		usageReplayLabel(receipt.Replayed),
	)
	return nil
}

// UsageQuotaSetCommand sets one subject hard-limit override with CAS.
type UsageQuotaSetCommand struct{ output *console.Output }

func NewUsageQuotaSetCommand() *UsageQuotaSetCommand {
	return &UsageQuotaSetCommand{output: console.NewOutput()}
}

func (c *UsageQuotaSetCommand) Name() string        { return "usage:quota:set" }
func (c *UsageQuotaSetCommand) Description() string { return "Set one subject usage quota" }
func (c *UsageQuotaSetCommand) Usage() string {
	return "usage:quota:set --scope=<scope> --subject-id=<id> --metric=<key> --limit=<n> --expected-version=<version>"
}

func (c *UsageQuotaSetCommand) Run(args []string) error {
	parsed, err := parseUsageQuotaArguments(c.Name(), args, true)
	if err != nil {
		return err
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	quota, err := runtime.quota.SetUsageQuota(
		ctx,
		parsed.target,
		parsed.metric,
		parsed.limit,
		parsed.expectedVersion,
	)
	if err != nil {
		return err
	}
	if quota.Version != parsed.expectedVersion {
		c.warnAuditFailure(recordUsageQuotaOperatorAudit(ctx, runtime.audit, c.Name(), quota, parsed.expectedVersion))
	}
	c.output.Success(
		"Set %s quota to %s at version %d",
		quota.Metric,
		formatUsageLimit(quota.Limit),
		quota.Version,
	)
	return nil
}

// UsageQuotaResetCommand resets one subject hard limit to its code default.
type UsageQuotaResetCommand struct{ output *console.Output }

func NewUsageQuotaResetCommand() *UsageQuotaResetCommand {
	return &UsageQuotaResetCommand{output: console.NewOutput()}
}

func (c *UsageQuotaResetCommand) Name() string        { return "usage:quota:reset" }
func (c *UsageQuotaResetCommand) Description() string { return "Reset one subject usage quota" }
func (c *UsageQuotaResetCommand) Usage() string {
	return "usage:quota:reset --scope=<scope> --subject-id=<id> --metric=<key> --expected-version=<version>"
}

func (c *UsageQuotaResetCommand) Run(args []string) error {
	parsed, err := parseUsageQuotaArguments(c.Name(), args, false)
	if err != nil {
		return err
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	quota, err := runtime.quota.ResetUsageQuota(
		ctx,
		parsed.target,
		parsed.metric,
		parsed.expectedVersion,
	)
	if err != nil {
		return err
	}
	if quota.Version != parsed.expectedVersion {
		c.warnAuditFailure(recordUsageQuotaOperatorAudit(ctx, runtime.audit, c.Name(), quota, parsed.expectedVersion))
	}
	c.output.Success("Reset %s quota at version %d (%s)", quota.Metric, quota.Version, quota.Source)
	return nil
}

// UsagePruneCommand removes finalized receipts older than the minimum retention horizon.
type UsagePruneCommand struct{ output *console.Output }

func NewUsagePruneCommand() *UsagePruneCommand {
	return &UsagePruneCommand{output: console.NewOutput()}
}

func (c *UsagePruneCommand) Name() string        { return "usage:prune" }
func (c *UsagePruneCommand) Description() string { return "Prune old finalized usage receipts" }
func (c *UsagePruneCommand) Usage() string {
	return "usage:prune --before=<RFC3339>"
}

func (c *UsagePruneCommand) Run(args []string) error {
	before, err := parseUsagePruneBefore(args)
	if err != nil {
		return err
	}
	runtime, err := loadUsageCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := usageCommandContext()
	defer stop()
	count, err := runtime.maintainer.PruneUsageReceipts(ctx, before)
	if err != nil {
		return err
	}
	c.warnAuditFailure(recordUsagePruneAudit(ctx, runtime.audit, before.UTC().Format("2006-01-02T15:04:05Z"), count))
	c.output.Success("Pruned %d finalized usage receipt(s)", count)
	return nil
}

func loadUsageCommandRuntime() (*usageCommandRuntime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if !slices.Contains(cfg.Starters.Optional, "usage") {
		return nil, fmt.Errorf("usage starter is not selected in OPTIONAL_STARTERS")
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return nil, loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize usage command: %w", err)
	}
	if application.UsageReader == nil || application.UsageRecorder == nil ||
		application.UsageConsumer == nil || application.UsageQuotaWriter == nil ||
		application.UsageMaintainer == nil || application.AuditRecorder == nil {
		return nil, errUsageCommandUnavailable
	}
	return &usageCommandRuntime{
		reader:     application.UsageReader,
		recorder:   application.UsageRecorder,
		consumer:   application.UsageConsumer,
		quota:      application.UsageQuotaWriter,
		maintainer: application.UsageMaintainer,
		audit:      application.AuditRecorder,
	}, nil
}

func usageCommandContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func recordUsageOperatorAudit(
	ctx context.Context,
	recorder domain.AuditLogRecorder,
	command string,
	receipt *domain.UsageReceipt,
	result string,
) error {
	if recorder == nil || receipt == nil {
		return errUsageCommandUnavailable
	}
	statusCode := 200
	if result == domain.AuditResultFailure {
		statusCode = 429
	}
	return recorder.Record(ctx, &domain.AuditLog{
		ActorType:  domain.AuditActorSystem,
		Action:     command,
		Resource:   "usage_events",
		TargetType: "usage_metric",
		TargetID:   usageTargetID(receipt.Target, receipt.Metric),
		Result:     result,
		Method:     "CLI",
		Path:       command,
		RouteName:  "console." + command,
		StatusCode: statusCode,
		Metadata: map[string]any{
			"scope":          receipt.Target.Scope,
			"metric":         receipt.Metric,
			"quantity":       receipt.Quantity,
			"decision":       receipt.Decision,
			"counter_before": receipt.CounterBefore,
			"counter_after":  receipt.CounterAfter,
		},
	})
}

func recordUsageQuotaOperatorAudit(
	ctx context.Context,
	recorder domain.AuditLogRecorder,
	command string,
	quota *domain.UsageQuota,
	beforeVersion uint64,
) error {
	if recorder == nil || quota == nil {
		return errUsageCommandUnavailable
	}
	return recorder.Record(ctx, &domain.AuditLog{
		ActorType:  domain.AuditActorSystem,
		Action:     command,
		Resource:   "usage_quotas",
		TargetType: "usage_quota",
		TargetID:   usageTargetID(quota.Target, quota.Metric),
		Result:     domain.AuditResultSuccess,
		Method:     "CLI",
		Path:       command,
		RouteName:  "console." + command,
		StatusCode: 200,
		Metadata: map[string]any{
			"scope":          quota.Target.Scope,
			"metric":         quota.Metric,
			"before_version": beforeVersion,
			"after_version":  quota.Version,
			"source":         quota.Source,
		},
	})
}

func recordUsagePruneAudit(
	ctx context.Context,
	recorder domain.AuditLogRecorder,
	before string,
	count int64,
) error {
	if recorder == nil {
		return errUsageCommandUnavailable
	}
	return recorder.Record(ctx, &domain.AuditLog{
		ActorType:  domain.AuditActorSystem,
		Action:     "usage:prune",
		Resource:   "usage_events",
		TargetType: "usage_receipts",
		TargetID:   "retention",
		Result:     domain.AuditResultSuccess,
		Method:     "CLI",
		Path:       "usage:prune",
		RouteName:  "console.usage:prune",
		StatusCode: 200,
		Metadata: map[string]any{
			"before": before,
			"count":  count,
		},
	})
}

func usageTargetID(target domain.UsageTarget, metric string) string {
	return string(target.Scope) + ":" + strconv.FormatUint(uint64(target.SubjectID), 10) + ":" + metric
}

func formatUsageLimit(value *int64) string {
	if value == nil {
		return "unlimited"
	}
	return strconv.FormatInt(*value, 10)
}

func usageReplayLabel(replayed bool) string {
	if replayed {
		return "replayed"
	}
	return "applied"
}

func (c *UsageRecordCommand) warnAuditFailure(err error) {
	if err != nil {
		c.output.Warning("Usage committed, but audit persistence failed")
	}
}

func (c *UsageConsumeCommand) warnAuditFailure(err error) {
	if err != nil {
		c.output.Warning("Usage decision committed, but audit persistence failed")
	}
}

func (c *UsageQuotaSetCommand) warnAuditFailure(err error) {
	if err != nil {
		c.output.Warning("Usage quota updated, but audit persistence failed")
	}
}

func (c *UsageQuotaResetCommand) warnAuditFailure(err error) {
	if err != nil {
		c.output.Warning("Usage quota reset, but audit persistence failed")
	}
}

func (c *UsagePruneCommand) warnAuditFailure(err error) {
	if err != nil {
		c.output.Warning("Usage receipts pruned, but audit persistence failed")
	}
}
