package setting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
	auditstarter "github.com/zgiai/luas/api/internal/modules/audit"
)

// Service owns effective value resolution and typed setting mutation.
type Service interface {
	domain.SettingReader
	domain.AppSettingWriter
	ListPublicAppSettings(context.Context) ([]*domain.Setting, error)
	SetSetting(context.Context, domain.SettingTarget, string, any, uint64) (*domain.Setting, error)
	ResetSetting(context.Context, domain.SettingTarget, string, uint64) (uint64, error)
}

type service struct {
	catalog *Catalog
	store   settingStore
	enabled bool
}

var (
	_ Service                 = (*service)(nil)
	_ domain.SettingReader    = (*service)(nil)
	_ domain.AppSettingWriter = (*service)(nil)
)

// NewService creates the optional typed setting service.
func NewService(catalog *Catalog, store settingStore, cfg *config.Config) *service {
	value := &service{catalog: catalog, store: store}
	if cfg != nil {
		value.enabled = slices.Contains(cfg.Starters.Optional, "setting")
	}
	return value
}

func (s *service) ListSettings(
	ctx context.Context,
	target domain.SettingTarget,
) ([]*domain.Setting, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	definitions := s.catalog.Definitions(target.Scope)
	if len(definitions) == 0 {
		return []*domain.Setting{}, nil
	}
	keys := make([]string, len(definitions))
	for index := range definitions {
		keys[index] = definitions[index].Key
	}
	records, err := s.store.List(ctx, target, keys)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Setting, 0, len(definitions))
	for _, definition := range definitions {
		effective, err := effectiveSetting(target, definition, records[definition.Key])
		if err != nil {
			return nil, err
		}
		result = append(result, effective)
	}
	return result, nil
}

func (s *service) GetSetting(
	ctx context.Context,
	target domain.SettingTarget,
	key string,
) (*domain.Setting, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	definition, ok := s.catalog.Definition(target.Scope, key)
	if !ok {
		return nil, domain.ErrSettingNotFound
	}
	records, err := s.store.List(ctx, target, []string{key})
	if err != nil {
		return nil, err
	}
	return effectiveSetting(target, definition, records[key])
}

func (s *service) ListPublicAppSettings(ctx context.Context) ([]*domain.Setting, error) {
	settings, err := s.ListSettings(ctx, domain.SettingTarget{Scope: domain.SettingScopeApp})
	if err != nil {
		return nil, err
	}
	public := make([]*domain.Setting, 0, len(settings))
	for _, value := range settings {
		if value.Visibility == domain.SettingVisibilityPublic {
			public = append(public, value)
		}
	}
	return public, nil
}

func (s *service) SetSetting(
	ctx context.Context,
	target domain.SettingTarget,
	key string,
	value any,
	expectedVersion uint64,
) (*domain.Setting, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if !target.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	definition, ok := s.catalog.Definition(target.Scope, key)
	if !ok {
		return nil, domain.ErrSettingNotFound
	}
	normalized, encoded, err := normalizeSettingValue(definition, value)
	if err != nil {
		return nil, err
	}
	record, changed, err := s.store.Set(ctx, target, key, encoded, expectedVersion)
	if err != nil {
		return nil, err
	}
	effective := settingFromRecord(target, definition, record, normalized)
	if changed {
		recordSettingAudit(ctx, "set", effective, expectedVersion)
	}
	return effective, nil
}

func (s *service) ResetSetting(
	ctx context.Context,
	target domain.SettingTarget,
	key string,
	expectedVersion uint64,
) (uint64, error) {
	if err := s.available(); err != nil {
		return 0, err
	}
	if !target.IsValid() {
		return 0, domain.ErrInvalidInput
	}
	definition, ok := s.catalog.Definition(target.Scope, key)
	if !ok {
		return 0, domain.ErrSettingNotFound
	}
	version, changed, err := s.store.Reset(ctx, target, key, expectedVersion)
	if err != nil {
		return 0, err
	}
	if changed {
		recordSettingAudit(ctx, "reset", &domain.Setting{
			Scope:     target.Scope,
			SubjectID: target.SubjectID,
			Key:       definition.Key,
			Version:   version,
			Source:    domain.SettingSourceDefault,
		}, expectedVersion)
	}
	return version, nil
}

func (s *service) SetAppSetting(
	ctx context.Context,
	key string,
	value any,
	expectedVersion uint64,
) (*domain.Setting, error) {
	return s.SetSetting(
		ctx,
		domain.SettingTarget{Scope: domain.SettingScopeApp},
		key,
		value,
		expectedVersion,
	)
}

func (s *service) ResetAppSetting(
	ctx context.Context,
	key string,
	expectedVersion uint64,
) (uint64, error) {
	return s.ResetSetting(
		ctx,
		domain.SettingTarget{Scope: domain.SettingScopeApp},
		key,
		expectedVersion,
	)
}

func (s *service) AccountDeletionCleanerName() string { return "setting" }

func (s *service) CleanAccountDeletion(ctx context.Context, userID uint) error {
	if err := s.available(); err != nil {
		return err
	}
	return s.store.DeleteForUser(ctx, userID)
}

func (s *service) available() error {
	if s == nil || !s.enabled || s.catalog == nil || s.store == nil {
		return domain.ErrServiceUnavailable
	}
	return nil
}

func effectiveSetting(
	target domain.SettingTarget,
	definition Definition,
	record *settingRecord,
) (*domain.Setting, error) {
	if record == nil || !record.IsOverridden {
		setting := settingFromRecord(target, definition, record, definition.Default)
		setting.Source = domain.SettingSourceDefault
		return setting, nil
	}
	value, err := decodeStoredSettingValue(definition, record.ValueJSON)
	if err != nil {
		return nil, err
	}
	return settingFromRecord(target, definition, record, value), nil
}

func settingFromRecord(
	target domain.SettingTarget,
	definition Definition,
	record *settingRecord,
	value any,
) *domain.Setting {
	setting := &domain.Setting{
		Scope:      target.Scope,
		SubjectID:  target.SubjectID,
		Key:        definition.Key,
		Kind:       definition.Kind,
		Visibility: definition.Visibility,
		Value:      value,
		Source:     domain.SettingSourceOverride,
		Options:    slices.Clone(definition.Options),
	}
	if record != nil {
		setting.Version = record.Version
		updated := record.UpdatedAt
		setting.UpdatedAt = &updated
	}
	return setting
}

func decodeStoredSettingValue(definition Definition, raw string) (any, error) {
	if raw == "" || len(raw) > maxSettingValueBytes {
		return nil, domain.ErrServiceUnavailable
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode stored setting: %w", domain.ErrServiceUnavailable)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode stored setting: %w", domain.ErrServiceUnavailable)
	}
	normalized, encoded, err := normalizeSettingValue(definition, value)
	if err != nil || encoded != raw {
		return nil, fmt.Errorf("validate stored setting: %w", domain.ErrServiceUnavailable)
	}
	return normalized, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return domain.ErrInvalidInput
}

func recordSettingAudit(
	ctx context.Context,
	operation string,
	setting *domain.Setting,
	beforeVersion uint64,
) {
	if setting == nil {
		return
	}
	metadata := map[string]any{
		"operation":      operation,
		"scope":          setting.Scope,
		"key":            setting.Key,
		"before_version": beforeVersion,
		"after_version":  setting.Version,
		"source":         setting.Source,
	}
	if setting.Scope == domain.SettingScopeUser {
		metadata["user_id"] = setting.SubjectID
	}
	if setting.Scope == domain.SettingScopeOrganization {
		metadata["organization_id"] = setting.SubjectID
	}
	auditstarter.RecordChange(ctx, auditstarter.Change{
		TargetType: "setting",
		TargetID:   string(setting.Scope) + ":" + strconv.FormatUint(uint64(setting.SubjectID), 10) + ":" + setting.Key,
		Result:     domain.AuditResultSuccess,
		Metadata:   metadata,
	})
}
