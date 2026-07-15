package setting

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	maxSettingDefinitions = 64
	maxSettingKeyBytes    = 96
	maxSettingValueBytes  = 4 * 1024
)

var settingKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*(?:\.[a-z][a-z0-9]*(?:_[a-z0-9]+)*)+$`)
var timezonePattern = regexp.MustCompile(`^[A-Za-z0-9_+-]+(?:/[A-Za-z0-9_+-]+)*$`)

// Definition is one immutable code-owned setting schema.
type Definition struct {
	Scope      domain.SettingScope
	Key        string
	Kind       domain.SettingKind
	Visibility domain.SettingVisibility
	Default    any
	Options    []string
	normalize  func(any) (any, error)
}

// Catalog is a finite validated registry of setting definitions.
type Catalog struct {
	ordered []Definition
	byScope map[domain.SettingScope]map[string]Definition
}

// NewDefaultCatalog returns the starter definitions. Downstream apps can replace this provider to
// compose product-owned definitions while preserving the finite typed boundary.
func NewDefaultCatalog() (*Catalog, error) {
	definitions := []Definition{
		NewStringDefinition(
			domain.SettingScopeApp,
			"branding.display_name",
			domain.SettingVisibilityPublic,
			"Luas",
			1,
			80,
		),
		NewEnumDefinition(
			domain.SettingScopeApp,
			"localization.locale",
			domain.SettingVisibilityPublic,
			"en-US",
			[]string{"en-US", "zh-Hans"},
		),
		NewEnumDefinition(
			domain.SettingScopeOrganization,
			"localization.locale",
			domain.SettingVisibilityPrivate,
			"en-US",
			[]string{"en-US", "zh-Hans"},
		),
		NewEnumDefinition(
			domain.SettingScopeUser,
			"localization.locale",
			domain.SettingVisibilityPrivate,
			"en-US",
			[]string{"en-US", "zh-Hans"},
		),
		NewTimezoneDefinition(
			domain.SettingScopeUser,
			"localization.timezone",
			domain.SettingVisibilityPrivate,
			"UTC",
		),
	}
	return NewCatalog(definitions...)
}

// NewCatalog validates a complete immutable definition snapshot.
func NewCatalog(definitions ...Definition) (*Catalog, error) {
	if len(definitions) == 0 {
		return nil, fmt.Errorf("at least one setting definition is required")
	}
	if len(definitions) > maxSettingDefinitions {
		return nil, fmt.Errorf("setting catalog exceeds %d definitions", maxSettingDefinitions)
	}

	catalog := &Catalog{
		ordered: make([]Definition, 0, len(definitions)),
		byScope: make(map[domain.SettingScope]map[string]Definition),
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
			return nil, fmt.Errorf("setting definition %s/%s is duplicated", definition.Scope, definition.Key)
		}
		byKey[definition.Key] = definition
		catalog.ordered = append(catalog.ordered, definition)
	}
	return catalog, nil
}

// Definition returns one definition without exposing mutable catalog state.
func (c *Catalog) Definition(scope domain.SettingScope, key string) (Definition, bool) {
	if c == nil {
		return Definition{}, false
	}
	definition, ok := c.byScope[scope][key]
	return cloneDefinition(definition), ok
}

// Definitions returns the bounded definitions for one scope in registration order.
func (c *Catalog) Definitions(scope domain.SettingScope) []Definition {
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

// NewStringDefinition creates a trimmed, control-free string schema.
func NewStringDefinition(
	scope domain.SettingScope,
	key string,
	visibility domain.SettingVisibility,
	defaultValue string,
	minRunes int,
	maxRunes int,
) Definition {
	return Definition{
		Scope:      scope,
		Key:        key,
		Kind:       domain.SettingKindString,
		Visibility: visibility,
		Default:    defaultValue,
		normalize: func(value any) (any, error) {
			text, ok := value.(string)
			if !ok {
				return nil, domain.ErrSettingInvalidValue
			}
			text = strings.TrimSpace(text)
			length := utf8.RuneCountInString(text)
			if !utf8.ValidString(text) || length < minRunes || length > maxRunes || containsControl(text) {
				return nil, domain.ErrSettingInvalidValue
			}
			return text, nil
		},
	}
}

// NewBooleanDefinition creates a strict JSON boolean schema.
func NewBooleanDefinition(
	scope domain.SettingScope,
	key string,
	visibility domain.SettingVisibility,
	defaultValue bool,
) Definition {
	return Definition{
		Scope:      scope,
		Key:        key,
		Kind:       domain.SettingKindBoolean,
		Visibility: visibility,
		Default:    defaultValue,
		normalize: func(value any) (any, error) {
			boolean, ok := value.(bool)
			if !ok {
				return nil, domain.ErrSettingInvalidValue
			}
			return boolean, nil
		},
	}
}

// NewIntegerDefinition creates a bounded JSON integer schema.
func NewIntegerDefinition(
	scope domain.SettingScope,
	key string,
	visibility domain.SettingVisibility,
	defaultValue int64,
	minimum int64,
	maximum int64,
) Definition {
	return Definition{
		Scope:      scope,
		Key:        key,
		Kind:       domain.SettingKindInteger,
		Visibility: visibility,
		Default:    defaultValue,
		normalize: func(value any) (any, error) {
			integer, ok := settingInteger(value)
			if !ok || integer < minimum || integer > maximum {
				return nil, domain.ErrSettingInvalidValue
			}
			return integer, nil
		},
	}
}

// NewEnumDefinition creates a fixed exact string vocabulary.
func NewEnumDefinition(
	scope domain.SettingScope,
	key string,
	visibility domain.SettingVisibility,
	defaultValue string,
	options []string,
) Definition {
	ownedOptions := slices.Clone(options)
	return Definition{
		Scope:      scope,
		Key:        key,
		Kind:       domain.SettingKindEnum,
		Visibility: visibility,
		Default:    defaultValue,
		Options:    ownedOptions,
		normalize: func(value any) (any, error) {
			text, ok := value.(string)
			if !ok || !slices.Contains(ownedOptions, text) {
				return nil, domain.ErrSettingInvalidValue
			}
			return text, nil
		},
	}
}

// NewTimezoneDefinition creates an IANA timezone schema.
func NewTimezoneDefinition(
	scope domain.SettingScope,
	key string,
	visibility domain.SettingVisibility,
	defaultValue string,
) Definition {
	return Definition{
		Scope:      scope,
		Key:        key,
		Kind:       domain.SettingKindTimezone,
		Visibility: visibility,
		Default:    defaultValue,
		normalize: func(value any) (any, error) {
			name, ok := value.(string)
			if !ok || len(name) == 0 || len(name) > 64 || name == "Local" || !timezonePattern.MatchString(name) {
				return nil, domain.ErrSettingInvalidValue
			}
			if _, err := time.LoadLocation(name); err != nil {
				return nil, domain.ErrSettingInvalidValue
			}
			return name, nil
		},
	}
}

func validateDefinition(candidate Definition) (Definition, error) {
	if !candidate.Scope.IsValid() {
		return Definition{}, fmt.Errorf("setting definition %q has invalid scope", candidate.Key)
	}
	if len(candidate.Key) == 0 || len(candidate.Key) > maxSettingKeyBytes || !settingKeyPattern.MatchString(candidate.Key) {
		return Definition{}, fmt.Errorf("setting definition key %q is invalid", candidate.Key)
	}
	if candidate.Visibility != domain.SettingVisibilityPublic && candidate.Visibility != domain.SettingVisibilityPrivate {
		return Definition{}, fmt.Errorf("setting definition %s/%s has invalid visibility", candidate.Scope, candidate.Key)
	}
	if candidate.Visibility == domain.SettingVisibilityPublic && candidate.Scope != domain.SettingScopeApp {
		return Definition{}, fmt.Errorf("only app setting definitions may be public")
	}
	if !candidate.Kind.IsValid() {
		return Definition{}, fmt.Errorf("setting definition %s/%s has invalid kind", candidate.Scope, candidate.Key)
	}
	if candidate.normalize == nil {
		return Definition{}, fmt.Errorf("setting definition %s/%s has no schema", candidate.Scope, candidate.Key)
	}
	normalized, err := candidate.normalize(candidate.Default)
	if err != nil {
		return Definition{}, fmt.Errorf("setting definition %s/%s has invalid default: %w", candidate.Scope, candidate.Key, err)
	}
	if err := validateEncodedSettingValue(normalized); err != nil {
		return Definition{}, fmt.Errorf("setting definition %s/%s default: %w", candidate.Scope, candidate.Key, err)
	}
	if candidate.Kind == domain.SettingKindEnum {
		if len(candidate.Options) == 0 || len(candidate.Options) > maxSettingDefinitions {
			return Definition{}, fmt.Errorf("setting definition %s/%s has invalid enum options", candidate.Scope, candidate.Key)
		}
		seen := make(map[string]struct{}, len(candidate.Options))
		for _, option := range candidate.Options {
			if option == "" || len(option) > 128 || containsControl(option) {
				return Definition{}, fmt.Errorf("setting definition %s/%s has invalid enum option", candidate.Scope, candidate.Key)
			}
			if _, exists := seen[option]; exists {
				return Definition{}, fmt.Errorf("setting definition %s/%s has duplicate enum option", candidate.Scope, candidate.Key)
			}
			seen[option] = struct{}{}
		}
	} else if len(candidate.Options) != 0 {
		return Definition{}, fmt.Errorf("only enum setting definitions may expose options")
	}
	candidate.Default = normalized
	candidate.Options = slices.Clone(candidate.Options)
	return candidate, nil
}

func normalizeSettingValue(definition Definition, value any) (any, string, error) {
	normalized, err := definition.normalize(value)
	if err != nil {
		return nil, "", domain.ErrSettingInvalidValue
	}
	encoded, err := json.Marshal(normalized)
	if err != nil || len(encoded) > maxSettingValueBytes {
		return nil, "", domain.ErrSettingInvalidValue
	}
	return normalized, string(encoded), nil
}

func validateEncodedSettingValue(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return domain.ErrSettingInvalidValue
	}
	if len(encoded) == 0 || len(encoded) > maxSettingValueBytes {
		return domain.ErrSettingInvalidValue
	}
	return nil
}

func settingInteger(value any) (int64, bool) {
	switch candidate := value.(type) {
	case json.Number:
		parsed, err := candidate.Int64()
		return parsed, err == nil
	case int:
		return int64(candidate), true
	case int8:
		return int64(candidate), true
	case int16:
		return int64(candidate), true
	case int32:
		return int64(candidate), true
	case int64:
		return candidate, true
	case uint:
		if uint64(candidate) > math.MaxInt64 {
			return 0, false
		}
		return int64(candidate), true
	case uint8:
		return int64(candidate), true
	case uint16:
		return int64(candidate), true
	case uint32:
		return int64(candidate), true
	case uint64:
		if candidate > math.MaxInt64 {
			return 0, false
		}
		return int64(candidate), true
	case float64:
		if math.IsNaN(candidate) || math.IsInf(candidate, 0) || math.Trunc(candidate) != candidate || candidate < math.MinInt64 || candidate > math.MaxInt64 {
			return 0, false
		}
		return int64(candidate), true
	default:
		return 0, false
	}
}

func containsControl(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func cloneDefinition(definition Definition) Definition {
	definition.Options = slices.Clone(definition.Options)
	return definition
}
