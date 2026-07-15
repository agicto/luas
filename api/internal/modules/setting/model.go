package setting

import (
	"time"

	"github.com/zgiai/luas/api/internal/modules/organization"
	"github.com/zgiai/luas/api/internal/modules/user"
)

// SettingPO stores one durable override or reset tombstone for a code-owned definition.
type SettingPO struct {
	ID             uint   `gorm:"primaryKey"`
	Scope          string `gorm:"size:24;not null;uniqueIndex:idx_settings_scope_subject_key,priority:1;check:settings_scope_check,scope IN ('app','organization','user')"`
	SubjectID      uint   `gorm:"not null;uniqueIndex:idx_settings_scope_subject_key,priority:2;index:idx_settings_user,priority:2;index:idx_settings_organization,priority:2;check:settings_subject_check,(scope = 'app' AND subject_id = 0 AND user_id IS NULL AND organization_id IS NULL) OR (scope = 'user' AND subject_id = user_id AND user_id IS NOT NULL AND organization_id IS NULL) OR (scope = 'organization' AND subject_id = organization_id AND organization_id IS NOT NULL AND user_id IS NULL)"`
	UserID         *uint  `gorm:"index:idx_settings_user,priority:1"`
	OrganizationID *uint  `gorm:"index:idx_settings_organization,priority:1"`
	Key            string `gorm:"size:96;not null;uniqueIndex:idx_settings_scope_subject_key,priority:3;check:settings_key_check,length(key) BETWEEN 1 AND 96"`
	ValueJSON      string `gorm:"type:text;not null;default:'';check:settings_value_state_check,length(value_json) <= 4096 AND ((is_overridden = TRUE AND value_json <> '') OR (is_overridden = FALSE AND value_json = ''))"`
	IsOverridden   bool   `gorm:"not null;default:false"`
	Version        uint64 `gorm:"not null;check:settings_version_check,version > 0"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	User         *user.UserPO                 `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Organization *organization.OrganizationPO `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (SettingPO) TableName() string { return "settings" }

type settingRecord struct {
	ID           uint
	ValueJSON    string
	IsOverridden bool
	Version      uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func recordFromPO(po *SettingPO) *settingRecord {
	if po == nil {
		return nil
	}
	return &settingRecord{
		ID:           po.ID,
		ValueJSON:    po.ValueJSON,
		IsOverridden: po.IsOverridden,
		Version:      po.Version,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}
