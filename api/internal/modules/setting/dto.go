package setting

import (
	"slices"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

type setSettingRequest struct {
	Value any `json:"value"`
}

// SettingResponse is the stable effective setting representation.
type SettingResponse struct {
	Scope      domain.SettingScope      `json:"scope"`
	Key        string                   `json:"key"`
	Kind       domain.SettingKind       `json:"kind"`
	Visibility domain.SettingVisibility `json:"visibility"`
	Value      any                      `json:"value"`
	Version    uint64                   `json:"version"`
	Source     domain.SettingSource     `json:"source"`
	Options    []string                 `json:"options,omitempty"`
	UpdatedAt  *time.Time               `json:"updated_at"`
}

func toSettingResponse(value *domain.Setting) *SettingResponse {
	if value == nil {
		return nil
	}
	return &SettingResponse{
		Scope:      value.Scope,
		Key:        value.Key,
		Kind:       value.Kind,
		Visibility: value.Visibility,
		Value:      value.Value,
		Version:    value.Version,
		Source:     value.Source,
		Options:    slices.Clone(value.Options),
		UpdatedAt:  cloneSettingTime(value.UpdatedAt),
	}
}

func toSettingResponses(values []*domain.Setting) []*SettingResponse {
	result := make([]*SettingResponse, len(values))
	for index := range values {
		result[index] = toSettingResponse(values[index])
	}
	return result
}

func cloneSettingTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
