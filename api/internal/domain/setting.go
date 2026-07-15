package domain

import (
	"context"
	"time"
)

// SettingScope identifies the owner boundary for one code-owned setting definition.
type SettingScope string

const (
	SettingScopeApp          SettingScope = "app"
	SettingScopeOrganization SettingScope = "organization"
	SettingScopeUser         SettingScope = "user"
)

// IsValid reports whether the scope belongs to the stable setting vocabulary.
func (s SettingScope) IsValid() bool {
	switch s {
	case SettingScopeApp, SettingScopeOrganization, SettingScopeUser:
		return true
	default:
		return false
	}
}

// SettingKind is the scalar schema owned by a setting definition.
type SettingKind string

const (
	SettingKindString   SettingKind = "string"
	SettingKindBoolean  SettingKind = "boolean"
	SettingKindInteger  SettingKind = "integer"
	SettingKindEnum     SettingKind = "enum"
	SettingKindTimezone SettingKind = "timezone"
)

// IsValid reports whether the kind belongs to the stable scalar setting vocabulary.
func (k SettingKind) IsValid() bool {
	switch k {
	case SettingKindString, SettingKindBoolean, SettingKindInteger, SettingKindEnum, SettingKindTimezone:
		return true
	default:
		return false
	}
}

// SettingVisibility controls whether an app setting may enter the public contract.
type SettingVisibility string

const (
	SettingVisibilityPublic  SettingVisibility = "public"
	SettingVisibilityPrivate SettingVisibility = "private"
)

// SettingSource says whether the effective value comes from code or durable state.
type SettingSource string

const (
	SettingSourceDefault  SettingSource = "default"
	SettingSourceOverride SettingSource = "override"
)

// SettingTarget identifies one app, organization, or user setting owner.
type SettingTarget struct {
	Scope     SettingScope
	SubjectID uint
}

// IsValid enforces the canonical subject identity for each scope.
func (t SettingTarget) IsValid() bool {
	if !t.Scope.IsValid() {
		return false
	}
	if t.Scope == SettingScopeApp {
		return t.SubjectID == 0
	}
	return t.SubjectID > 0
}

// Setting is one effective code-defined value and its durable version history.
type Setting struct {
	Scope      SettingScope
	SubjectID  uint
	Key        string
	Kind       SettingKind
	Visibility SettingVisibility
	Value      any
	Version    uint64
	Source     SettingSource
	Options    []string
	UpdatedAt  *time.Time
}

// SettingReader is the downstream read seam for code-owned settings.
type SettingReader interface {
	ListSettings(ctx context.Context, target SettingTarget) ([]*Setting, error)
	GetSetting(ctx context.Context, target SettingTarget, key string) (*Setting, error)
}

// AppSettingWriter is the operator/internal mutation seam for app-scoped settings.
type AppSettingWriter interface {
	SetAppSetting(ctx context.Context, key string, value any, expectedVersion uint64) (*Setting, error)
	ResetAppSetting(ctx context.Context, key string, expectedVersion uint64) (uint64, error)
}
