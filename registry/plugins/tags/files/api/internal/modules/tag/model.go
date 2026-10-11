package tag

import (
	"slices"
	"time"
)

const (
	maxNameLength          = 50
	maxDescriptionLength   = 200
	maxTagsPerResource     = 20
	maxResourceTypeLength  = 50
	defaultColor           = "gray"
	maxTagCursorSortKeyLen = 4 * maxNameLength
)

// Colors is the fixed palette of design-token names a tag may use. Browsers map each name to their
// own theme tokens, so the API never stores raw color values.
var Colors = []string{"gray", "red", "orange", "yellow", "green", "blue", "purple", "pink"}

// ValidColor reports whether color is a palette token name.
func ValidColor(color string) bool {
	return slices.Contains(Colors, color)
}

// Tag is an organization-owned label that resources of registered types can carry.
type Tag struct {
	ID             uint
	OrganizationID uint
	Name           string
	Color          string
	Description    string
	CreatedBy      *uint
	Version        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ResourceRef names one resource of a registered type inside an organization.
type ResourceRef struct {
	Type string
	ID   string
}

// TaggedResource is one resource that carries a tag.
type TaggedResource struct {
	AssignmentID uint
	ResourceType string
	ResourceID   string
	TaggedAt     time.Time
}

// TagCursor marks the last tag of a keyset page. SortKey is the database's lowercase name, which
// is unique per organization.
type TagCursor struct {
	SortKey string
}

// AssignmentCursor marks the last assignment of a keyset page of tagged resources.
type AssignmentCursor struct {
	ID uint
}

// tagPO maps tags.
type tagPO struct {
	ID             uint      `gorm:"primaryKey"`
	OrganizationID uint      `gorm:"not null"`
	Name           string    `gorm:"size:50;not null"`
	Color          string    `gorm:"size:16;not null"`
	Description    string    `gorm:"size:200;not null"`
	CreatedBy      *uint     `gorm:"column:created_by"`
	Version        uint64    `gorm:"not null;default:1"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (tagPO) TableName() string { return "tags" }

func (po *tagPO) toDomain() *Tag {
	return &Tag{
		ID: po.ID, OrganizationID: po.OrganizationID, Name: po.Name, Color: po.Color,
		Description: po.Description, CreatedBy: po.CreatedBy, Version: po.Version,
		CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}

// assignmentPO maps tag_assignments.
type assignmentPO struct {
	ID             uint      `gorm:"primaryKey"`
	OrganizationID uint      `gorm:"not null"`
	TagID          uint      `gorm:"not null"`
	ResourceType   string    `gorm:"size:50;not null"`
	ResourceID     string    `gorm:"size:64;not null"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (assignmentPO) TableName() string { return "tag_assignments" }
