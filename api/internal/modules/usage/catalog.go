package usage

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	maxUsageDefinitions     = 64
	maxUsageMetricKeyBytes  = 96
	maxUsageUnitBytes       = 32
	maxUsageDimensions      = 8
	maxUsageDimensionValues = 32
	maxUsageDimensionBytes  = 64
	maxUsageSourceBytes     = 64
	maxUsageEventIDBytes    = 128
	maxSafeUsageInteger     = int64(9_007_199_254_740_991)
)

var usageMetricKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*(?:\.[a-z][a-z0-9]*(?:_[a-z0-9]+)*)+$`)
var usageUnitPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
var usageDimensionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
var usageSourcePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*(?:\.[a-z][a-z0-9]*(?:_[a-z0-9]+)*)*$`)
var usageEventIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// DimensionDefinition is one finite code-owned event dimension.
type DimensionDefinition struct {
	Key    string
	Values []string
}

// Definition is one immutable scoped usage metric schema.
type Definition struct {
	Scope        domain.UsageScope
	Key          string
	Unit         string
	Period       domain.UsagePeriod
	DefaultLimit *int64
	Dimensions   []DimensionDefinition
}

// Catalog is a finite validated registry of scoped usage metrics.
type Catalog struct {
	ordered []Definition
	byScope map[domain.UsageScope]map[string]Definition
}

// NewDefaultCatalog returns provider-neutral starter metrics with unlimited safe defaults.
func NewDefaultCatalog() (*Catalog, error) {
	metrics := []struct {
		key  string
		unit string
	}{
		{key: "api.requests", unit: "request"},
		{key: "ai.input_tokens", unit: "token"},
		{key: "ai.output_tokens", unit: "token"},
		{key: "asset.transfer_bytes", unit: "byte"},
		{key: "workflow.runs", unit: "run"},
	}
	definitions := make([]Definition, 0, len(metrics)*2)
	for _, scope := range []domain.UsageScope{
		domain.UsageScopeUser,
		domain.UsageScopeOrganization,
	} {
		for _, metric := range metrics {
			definitions = append(definitions, NewDefinition(
				scope,
				metric.key,
				metric.unit,
				domain.UsagePeriodMonth,
				nil,
				nil,
			))
		}
	}
	return NewCatalog(definitions...)
}

// NewDefinition creates a usage metric candidate for catalog validation.
func NewDefinition(
	scope domain.UsageScope,
	key string,
	unit string,
	period domain.UsagePeriod,
	defaultLimit *int64,
	dimensions []DimensionDefinition,
) Definition {
	return Definition{
		Scope:        scope,
		Key:          key,
		Unit:         unit,
		Period:       period,
		DefaultLimit: cloneInt64(defaultLimit),
		Dimensions:   cloneDimensionDefinitions(dimensions),
	}
}

// NewCatalog validates one complete immutable usage metric snapshot.
func NewCatalog(definitions ...Definition) (*Catalog, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("at least one usage metric definition is required")
	}
	if len(definitions) > maxUsageDefinitions {
		return nil, fmt.Errorf("usage catalog exceeds %d definitions", maxUsageDefinitions)
	}
	catalog := &Catalog{
		ordered: make([]Definition, 0, len(definitions)),
		byScope: make(map[domain.UsageScope]map[string]Definition),
	}
	for _, candidate := range definitions {
		definition, err := validateDefinition(candidate)
		if err != nil {
			return nil, err
		}
		byKey := catalog.byScope[definition.Scope]
		if byKey == nil {
			byKey = make(map[string]Definition)
			catalog.byScope[definition.Scope] = byKey
		}
		if _, exists := byKey[definition.Key]; exists {
			return nil, fmt.Errorf("usage metric %s/%s is duplicated", definition.Scope, definition.Key)
		}
		byKey[definition.Key] = definition
		catalog.ordered = append(catalog.ordered, definition)
	}
	return catalog, nil
}

// Definition returns one metric without exposing mutable catalog state.
func (c *Catalog) Definition(scope domain.UsageScope, key string) (Definition, bool) {
	if c == nil {
		return Definition{}, false
	}
	definition, ok := c.byScope[scope][key]
	return cloneDefinition(definition), ok
}

// Definitions returns scoped definitions in registration order.
func (c *Catalog) Definitions(scope domain.UsageScope) []Definition {
	if c == nil {
		return nil
	}
	result := make([]Definition, 0, len(c.ordered))
	for _, definition := range c.ordered {
		if definition.Scope == scope {
			result = append(result, cloneDefinition(definition))
		}
	}
	return result
}

func validateDefinition(candidate Definition) (Definition, error) {
	if !candidate.Scope.IsValid() {
		return Definition{}, fmt.Errorf("usage metric %q has invalid scope", candidate.Key)
	}
	if len(candidate.Key) == 0 || len(candidate.Key) > maxUsageMetricKeyBytes || !usageMetricKeyPattern.MatchString(candidate.Key) {
		return Definition{}, fmt.Errorf("usage metric key %q is invalid", candidate.Key)
	}
	if len(candidate.Unit) == 0 || len(candidate.Unit) > maxUsageUnitBytes || !usageUnitPattern.MatchString(candidate.Unit) {
		return Definition{}, fmt.Errorf("usage metric %s/%s has invalid unit", candidate.Scope, candidate.Key)
	}
	if !candidate.Period.IsValid() {
		return Definition{}, fmt.Errorf("usage metric %s/%s has invalid period", candidate.Scope, candidate.Key)
	}
	if candidate.DefaultLimit != nil && (*candidate.DefaultLimit < 0 || *candidate.DefaultLimit > maxSafeUsageInteger) {
		return Definition{}, fmt.Errorf("usage metric %s/%s has invalid default limit", candidate.Scope, candidate.Key)
	}
	if len(candidate.Dimensions) > maxUsageDimensions {
		return Definition{}, fmt.Errorf("usage metric %s/%s exceeds %d dimensions", candidate.Scope, candidate.Key, maxUsageDimensions)
	}
	seen := make(map[string]struct{}, len(candidate.Dimensions))
	for index := range candidate.Dimensions {
		dimension := &candidate.Dimensions[index]
		if len(dimension.Key) == 0 || len(dimension.Key) > maxUsageDimensionBytes || !usageDimensionKeyPattern.MatchString(dimension.Key) {
			return Definition{}, fmt.Errorf("usage metric %s/%s has invalid dimension key", candidate.Scope, candidate.Key)
		}
		if _, exists := seen[dimension.Key]; exists {
			return Definition{}, fmt.Errorf("usage metric %s/%s has duplicate dimension %s", candidate.Scope, candidate.Key, dimension.Key)
		}
		seen[dimension.Key] = struct{}{}
		if len(dimension.Values) == 0 || len(dimension.Values) > maxUsageDimensionValues {
			return Definition{}, fmt.Errorf("usage metric %s/%s dimension %s has invalid values", candidate.Scope, candidate.Key, dimension.Key)
		}
		valueSeen := make(map[string]struct{}, len(dimension.Values))
		for _, value := range dimension.Values {
			if value == "" || len(value) > maxUsageDimensionBytes || strings.TrimSpace(value) != value || containsControl(value) {
				return Definition{}, fmt.Errorf("usage metric %s/%s dimension %s has invalid value", candidate.Scope, candidate.Key, dimension.Key)
			}
			if _, exists := valueSeen[value]; exists {
				return Definition{}, fmt.Errorf("usage metric %s/%s dimension %s has duplicate value", candidate.Scope, candidate.Key, dimension.Key)
			}
			valueSeen[value] = struct{}{}
		}
		dimension.Values = slices.Clone(dimension.Values)
	}
	candidate.DefaultLimit = cloneInt64(candidate.DefaultLimit)
	candidate.Dimensions = cloneDimensionDefinitions(candidate.Dimensions)
	return candidate, nil
}

func normalizeUsageDimensions(
	definition Definition,
	dimensions map[string]string,
) (map[string]string, string, error) {
	if len(dimensions) != len(definition.Dimensions) {
		return nil, "", domain.ErrUsageInvalidEvent
	}
	result := make(map[string]string, len(dimensions))
	for _, expected := range definition.Dimensions {
		value, ok := dimensions[expected.Key]
		if !ok || !slices.Contains(expected.Values, value) {
			return nil, "", domain.ErrUsageInvalidEvent
		}
		result[expected.Key] = value
	}
	for key := range dimensions {
		if _, ok := result[key]; !ok {
			return nil, "", domain.ErrUsageInvalidEvent
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > 4096 {
		return nil, "", domain.ErrUsageInvalidEvent
	}
	return result, string(encoded), nil
}

func validUsageSource(value string) bool {
	return len(value) > 0 && len(value) <= maxUsageSourceBytes && usageSourcePattern.MatchString(value)
}

func validUsageEventID(value string) bool {
	return len(value) > 0 && len(value) <= maxUsageEventIDBytes && usageEventIDPattern.MatchString(value)
}

func cloneDefinition(definition Definition) Definition {
	definition.DefaultLimit = cloneInt64(definition.DefaultLimit)
	definition.Dimensions = cloneDimensionDefinitions(definition.Dimensions)
	return definition
}

func cloneDimensionDefinitions(values []DimensionDefinition) []DimensionDefinition {
	if len(values) == 0 {
		return nil
	}
	result := make([]DimensionDefinition, len(values))
	for index := range values {
		result[index] = DimensionDefinition{
			Key:    values[index].Key,
			Values: slices.Clone(values[index].Values),
		}
	}
	return result
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func containsControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}
