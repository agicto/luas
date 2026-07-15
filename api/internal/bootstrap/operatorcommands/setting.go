package operatorcommands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/zgiai/luas/api/internal/bootstrap"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	"github.com/zgiai/luas/api/internal/infra/console"
	"github.com/zgiai/luas/api/internal/wiring"
)

var errSettingCommandUnavailable = errors.New("setting command service is unavailable")

type settingCommandRuntime struct {
	reader domain.SettingReader
	writer domain.AppSettingWriter
	audit  domain.AuditLogRecorder
}

type settingSetArguments struct {
	key             string
	value           any
	expectedVersion uint64
}

type settingResetArguments struct {
	key             string
	expectedVersion uint64
}

// SettingListCommand lists app setting metadata without printing values.
type SettingListCommand struct {
	output *console.Output
}

func NewSettingListCommand() *SettingListCommand {
	return &SettingListCommand{output: console.NewOutput()}
}

func (c *SettingListCommand) Name() string        { return "setting:list" }
func (c *SettingListCommand) Description() string { return "List app setting metadata" }
func (c *SettingListCommand) Usage() string       { return "setting:list" }

func (c *SettingListCommand) Run(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("setting:list accepts no arguments")
	}
	runtime, err := loadSettingCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := settingCommandContext()
	defer stop()
	values, err := runtime.reader.ListSettings(ctx, domain.SettingTarget{Scope: domain.SettingScopeApp})
	if err != nil {
		return err
	}
	rows := make([][]string, len(values))
	for index, value := range values {
		rows[index] = []string{
			value.Key,
			string(value.Kind),
			string(value.Visibility),
			strconv.FormatUint(value.Version, 10),
			string(value.Source),
		}
	}
	c.output.Table([]string{"KEY", "KIND", "VISIBILITY", "VERSION", "SOURCE"}, rows)
	return nil
}

// SettingSetCommand sets one app override with an explicit expected version.
type SettingSetCommand struct {
	output *console.Output
}

func NewSettingSetCommand() *SettingSetCommand {
	return &SettingSetCommand{output: console.NewOutput()}
}

func (c *SettingSetCommand) Name() string        { return "setting:set" }
func (c *SettingSetCommand) Description() string { return "Set one app setting override" }
func (c *SettingSetCommand) Usage() string {
	return "setting:set --key=<key> --value=<json-scalar> --expected-version=<version>"
}

func (c *SettingSetCommand) Run(args []string) error {
	parsed, err := parseSettingSetArguments(args)
	if err != nil {
		return err
	}
	runtime, err := loadSettingCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := settingCommandContext()
	defer stop()
	value, err := runtime.writer.SetAppSetting(ctx, parsed.key, parsed.value, parsed.expectedVersion)
	if err != nil {
		return err
	}
	if auditErr := recordSettingCommandAudit(
		ctx,
		runtime.audit,
		"set",
		parsed.key,
		parsed.expectedVersion,
		value.Version,
		value.Source,
	); auditErr != nil {
		c.output.Warning("App setting updated, but audit persistence failed")
	}
	c.output.Success(
		"Updated app setting %s at version %d (%s)",
		value.Key,
		value.Version,
		value.Source,
	)
	return nil
}

// SettingResetCommand resets one app setting with an explicit expected version.
type SettingResetCommand struct {
	output *console.Output
}

func NewSettingResetCommand() *SettingResetCommand {
	return &SettingResetCommand{output: console.NewOutput()}
}

func (c *SettingResetCommand) Name() string { return "setting:reset" }
func (c *SettingResetCommand) Description() string {
	return "Reset one app setting to its code default"
}
func (c *SettingResetCommand) Usage() string {
	return "setting:reset --key=<key> --expected-version=<version>"
}

func (c *SettingResetCommand) Run(args []string) error {
	parsed, err := parseSettingResetArguments(args)
	if err != nil {
		return err
	}
	runtime, err := loadSettingCommandRuntime()
	if err != nil {
		return err
	}
	ctx, stop := settingCommandContext()
	defer stop()
	version, err := runtime.writer.ResetAppSetting(ctx, parsed.key, parsed.expectedVersion)
	if err != nil {
		return err
	}
	if auditErr := recordSettingCommandAudit(
		ctx,
		runtime.audit,
		"reset",
		parsed.key,
		parsed.expectedVersion,
		version,
		domain.SettingSourceDefault,
	); auditErr != nil {
		c.output.Warning("App setting reset, but audit persistence failed")
	}
	c.output.Success("Reset app setting %s at version %d (default)", parsed.key, version)
	return nil
}

func loadSettingCommandRuntime() (*settingCommandRuntime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if !slices.Contains(cfg.Starters.Optional, "setting") {
		return nil, fmt.Errorf("setting starter is not selected in OPTIONAL_STARTERS")
	}
	if loggerErr := bootstrap.InitLogger(cfg); loggerErr != nil {
		return nil, loggerErr
	}
	application, err := wiring.InitApplicationWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize setting command: %w", err)
	}
	if application.SettingReader == nil || application.AppSettingWriter == nil || application.AuditRecorder == nil {
		return nil, errSettingCommandUnavailable
	}
	return &settingCommandRuntime{
		reader: application.SettingReader,
		writer: application.AppSettingWriter,
		audit:  application.AuditRecorder,
	}, nil
}

func recordSettingCommandAudit(
	ctx context.Context,
	recorder domain.AuditLogRecorder,
	operation string,
	key string,
	beforeVersion uint64,
	afterVersion uint64,
	source domain.SettingSource,
) error {
	if recorder == nil {
		return errSettingCommandUnavailable
	}
	command := "setting:" + operation
	return recorder.Record(ctx, &domain.AuditLog{
		ActorType:  domain.AuditActorSystem,
		Action:     operation,
		Resource:   "settings",
		TargetType: "setting",
		TargetID:   "app:0:" + key,
		Result:     domain.AuditResultSuccess,
		Method:     "CLI",
		Path:       command,
		RouteName:  "console.setting." + operation,
		StatusCode: 200,
		Metadata: map[string]any{
			"operation":      operation,
			"scope":          domain.SettingScopeApp,
			"key":            key,
			"before_version": beforeVersion,
			"after_version":  afterVersion,
			"source":         source,
		},
	})
}

func settingCommandContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func parseSettingSetArguments(args []string) (*settingSetArguments, error) {
	values, err := parseSettingFlags(args, map[string]bool{
		"key":              true,
		"value":            true,
		"expected-version": true,
	})
	if err != nil {
		return nil, err
	}
	value, err := decodeSettingCommandValue(values["value"])
	if err != nil {
		return nil, err
	}
	version, err := parseSettingCommandVersion(values["expected-version"])
	if err != nil {
		return nil, err
	}
	return &settingSetArguments{
		key:             values["key"],
		value:           value,
		expectedVersion: version,
	}, nil
}

func parseSettingResetArguments(args []string) (*settingResetArguments, error) {
	values, err := parseSettingFlags(args, map[string]bool{
		"key":              true,
		"expected-version": true,
	})
	if err != nil {
		return nil, err
	}
	version, err := parseSettingCommandVersion(values["expected-version"])
	if err != nil {
		return nil, err
	}
	return &settingResetArguments{key: values["key"], expectedVersion: version}, nil
}

func parseSettingFlags(args []string, expected map[string]bool) (map[string]string, error) {
	values := make(map[string]string, len(expected))
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if !strings.HasPrefix(argument, "--") {
			return nil, fmt.Errorf("unknown setting argument %q", argument)
		}
		nameValue := strings.SplitN(strings.TrimPrefix(argument, "--"), "=", 2)
		name := nameValue[0]
		if !expected[name] {
			return nil, fmt.Errorf("unknown setting flag --%s", name)
		}
		if _, exists := values[name]; exists {
			return nil, fmt.Errorf("duplicate setting flag --%s", name)
		}
		value := ""
		if len(nameValue) == 2 {
			value = nameValue[1]
		} else if index+1 < len(args) {
			index++
			value = args[index]
		}
		if value == "" {
			return nil, fmt.Errorf("setting flag --%s requires a value", name)
		}
		values[name] = value
	}
	for name := range expected {
		if values[name] == "" {
			return nil, fmt.Errorf("setting flag --%s is required", name)
		}
	}
	return values, nil
}

func decodeSettingCommandValue(raw string) (any, error) {
	if len(raw) == 0 || len(raw) > 4*1024 {
		return nil, fmt.Errorf("--value must contain at most 4096 bytes of JSON")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode --value: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("--value must contain exactly one JSON value")
	}
	if value == nil {
		return nil, fmt.Errorf("--value cannot be null")
	}
	switch candidate := value.(type) {
	case string, bool:
		return value, nil
	case json.Number:
		if _, err := candidate.Int64(); err != nil {
			return nil, fmt.Errorf("--value JSON number must be an integer")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("--value must be a JSON string, boolean, or integer")
	}
}

func parseSettingCommandVersion(raw string) (uint64, error) {
	if raw != "0" && (len(raw) == 0 || raw[0] == '0') {
		return 0, fmt.Errorf("--expected-version must be a canonical unsigned integer")
	}
	version, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid --expected-version: %w", err)
	}
	return version, nil
}
