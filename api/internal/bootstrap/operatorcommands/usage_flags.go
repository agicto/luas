package operatorcommands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

type usageTargetArguments struct {
	target domain.UsageTarget
}

type usageMutationArguments struct {
	target     domain.UsageTarget
	metric     string
	quantity   int64
	source     string
	eventID    string
	dimensions map[string]string
	occurredAt time.Time
}

type usageQuotaArguments struct {
	target          domain.UsageTarget
	metric          string
	limit           int64
	expectedVersion uint64
}

func parseUsageTargetArguments(command string, args []string) (*usageTargetArguments, error) {
	flags := newUsageFlagSet(command)
	scope := flags.String("scope", "", "usage scope")
	subjectID := flags.Uint64("subject-id", 0, "usage subject ID")
	if err := parseUsageFlags(flags, args); err != nil {
		return nil, err
	}
	target, err := parseUsageTarget(*scope, *subjectID)
	if err != nil {
		return nil, err
	}
	return &usageTargetArguments{target: target}, nil
}

func parseUsageMutationArguments(
	command string,
	args []string,
	requireOccurredAt bool,
) (*usageMutationArguments, error) {
	flags := newUsageFlagSet(command)
	scope := flags.String("scope", "", "usage scope")
	subjectID := flags.Uint64("subject-id", 0, "usage subject ID")
	metric := flags.String("metric", "", "usage metric")
	quantity := flags.Int64("quantity", 0, "usage quantity")
	source := flags.String("source", "", "usage event source")
	eventID := flags.String("event-id", "", "usage event ID")
	dimensionsJSON := flags.String("dimensions", "{}", "finite usage dimensions JSON")
	var occurredAt *string
	if requireOccurredAt {
		occurredAt = flags.String("occurred-at", "", "RFC3339 occurrence time")
	}
	if err := parseUsageFlags(flags, args); err != nil {
		return nil, err
	}
	target, err := parseUsageTarget(*scope, *subjectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(*metric) == "" || strings.TrimSpace(*source) == "" || strings.TrimSpace(*eventID) == "" {
		return nil, fmt.Errorf("--metric, --source, and --event-id are required")
	}
	if *quantity == 0 {
		return nil, fmt.Errorf("--quantity must be non-zero")
	}
	dimensions, err := decodeUsageDimensions(*dimensionsJSON)
	if err != nil {
		return nil, err
	}
	parsed := &usageMutationArguments{
		target:     target,
		metric:     *metric,
		quantity:   *quantity,
		source:     *source,
		eventID:    *eventID,
		dimensions: dimensions,
	}
	if requireOccurredAt {
		if occurredAt == nil || *occurredAt == "" {
			return nil, fmt.Errorf("--occurred-at is required")
		}
		parsed.occurredAt, err = time.Parse(time.RFC3339Nano, *occurredAt)
		if err != nil {
			return nil, fmt.Errorf("invalid --occurred-at: %w", err)
		}
	}
	return parsed, nil
}

func parseUsageQuotaArguments(
	command string,
	args []string,
	requireLimit bool,
) (*usageQuotaArguments, error) {
	flags := newUsageFlagSet(command)
	scope := flags.String("scope", "", "usage scope")
	subjectID := flags.Uint64("subject-id", 0, "usage subject ID")
	metric := flags.String("metric", "", "usage metric")
	limit := flags.Int64("limit", -1, "hard usage limit")
	expectedVersion := flags.Uint64("expected-version", math.MaxUint64, "expected quota version")
	if err := parseUsageFlags(flags, args); err != nil {
		return nil, err
	}
	target, err := parseUsageTarget(*scope, *subjectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(*metric) == "" {
		return nil, fmt.Errorf("--metric is required")
	}
	if *expectedVersion == math.MaxUint64 {
		return nil, fmt.Errorf("--expected-version is required")
	}
	if requireLimit && *limit < 0 {
		return nil, fmt.Errorf("--limit must be a non-negative integer")
	}
	return &usageQuotaArguments{
		target:          target,
		metric:          *metric,
		limit:           *limit,
		expectedVersion: *expectedVersion,
	}, nil
}

func parseUsagePruneBefore(args []string) (time.Time, error) {
	flags := newUsageFlagSet("usage:prune")
	before := flags.String("before", "", "exclusive RFC3339 receipt cutoff")
	if err := parseUsageFlags(flags, args); err != nil {
		return time.Time{}, err
	}
	if *before == "" {
		return time.Time{}, fmt.Errorf("--before is required")
	}
	value, err := time.Parse(time.RFC3339Nano, *before)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --before: %w", err)
	}
	return value, nil
}

func parseUsageTarget(scope string, subjectID uint64) (domain.UsageTarget, error) {
	if subjectID == 0 || subjectID > uint64(^uint(0)) {
		return domain.UsageTarget{}, fmt.Errorf("--subject-id must identify an existing owner")
	}
	target := domain.UsageTarget{Scope: domain.UsageScope(scope), SubjectID: uint(subjectID)}
	if !target.IsValid() {
		return domain.UsageTarget{}, fmt.Errorf("--scope must be user or organization")
	}
	return target, nil
}

func decodeUsageDimensions(raw string) (map[string]string, error) {
	if len(raw) < 2 || len(raw) > 4096 {
		return nil, fmt.Errorf("--dimensions must be a JSON object of at most 4096 bytes")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var values map[string]string
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("decode --dimensions: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("--dimensions must contain exactly one JSON object")
	}
	if values == nil {
		return nil, fmt.Errorf("--dimensions must be a JSON object")
	}
	return values, nil
}

func newUsageFlagSet(command string) *flag.FlagSet {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parseUsageFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse %s flags: %w", flags.Name(), err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("%s does not accept positional arguments: %s", flags.Name(), strconv.Quote(strings.Join(flags.Args(), " ")))
	}
	return nil
}
