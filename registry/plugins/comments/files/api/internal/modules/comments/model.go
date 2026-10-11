package comments

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	maxBodyLength     = 10_000
	maxMentions       = 20
	maxPageSize       = 100
	replyPreviewLimit = 3
	maxResourceIDLen  = 64
	maxResourceTypes  = 64
	mentionKind       = "comments.mention"
)

// Comment is one comment on an organization resource. A root comment has no parent; a reply's
// parent is always a root, so threads are one level deep. A deleted comment keeps its place in
// the thread with its body and mentions erased.
type Comment struct {
	ID             uint
	OrganizationID uint
	ResourceType   string
	ResourceID     string
	ParentID       *uint
	AuthorID       *uint
	Body           string
	Mentions       []uint
	Version        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	EditedAt       *time.Time
	DeletedAt      *time.Time
}

// IsDeleted reports whether the comment was soft-deleted.
func (c *Comment) IsDeleted() bool { return c != nil && c.DeletedAt != nil }

// Thread is a root comment with the first replies and whether more exist.
type Thread struct {
	Root           *Comment
	Replies        []*Comment
	HasMoreReplies bool
}

// Cursor marks the last comment of a keyset page.
type Cursor struct {
	ID uint
}

// commentPO maps the comments table.
type commentPO struct {
	ID             uint       `gorm:"primaryKey"`
	OrganizationID uint       `gorm:"not null"`
	ResourceType   string     `gorm:"size:40;not null"`
	ResourceID     string     `gorm:"size:64;not null"`
	ParentID       *uint      `gorm:"column:parent_id"`
	AuthorID       *uint      `gorm:"column:author_id"`
	Body           *string    `gorm:"type:text"`
	Mentions       string     `gorm:"type:jsonb;not null"`
	Version        uint64     `gorm:"not null;default:1"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	EditedAt       *time.Time `gorm:"column:edited_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at"`
}

func (commentPO) TableName() string { return "comments" }

func (po *commentPO) toDomain() (*Comment, error) {
	comment := &Comment{
		ID: po.ID, OrganizationID: po.OrganizationID, ResourceType: po.ResourceType, ResourceID: po.ResourceID,
		ParentID: po.ParentID, AuthorID: po.AuthorID, Version: po.Version,
		CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt, EditedAt: po.EditedAt, DeletedAt: po.DeletedAt,
		Mentions: []uint{},
	}
	if po.Body != nil {
		comment.Body = *po.Body
	}
	if len(po.Mentions) > 0 {
		if err := json.Unmarshal([]byte(po.Mentions), &comment.Mentions); err != nil {
			return nil, err
		}
	}
	return comment, nil
}

func encodeMentions(mentions []uint) string {
	if len(mentions) == 0 {
		return "[]"
	}
	parts := make([]string, len(mentions))
	for index, userID := range mentions {
		parts[index] = strconv.FormatUint(uint64(userID), 10)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
