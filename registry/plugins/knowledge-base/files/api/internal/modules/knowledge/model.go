package knowledge

import "time"

const (
	maxTitleLength   = 200
	maxBodyLength    = 200_000
	maxQueryLength   = 200
	maxSearchResults = 20
	excerptLength    = 240
)

// Article is an organization-owned document with a Markdown body.
type Article struct {
	ID             uint
	OrganizationID uint
	Title          string
	Body           string
	CreatedBy      *uint
	UpdatedBy      *uint
	Version        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SearchHit is one ranked search result: the article without its full body.
type SearchHit struct {
	ID        uint
	Title     string
	Excerpt   string
	Version   uint64
	UpdatedAt time.Time
}

// ArticleCursor marks the last article of a keyset page.
type ArticleCursor struct {
	ID uint
}

// articlePO maps knowledge_articles. The generated search_vector column is maintained by
// PostgreSQL and never read or written here.
type articlePO struct {
	ID             uint      `gorm:"primaryKey"`
	OrganizationID uint      `gorm:"not null"`
	Title          string    `gorm:"size:200;not null"`
	Body           string    `gorm:"type:text;not null"`
	CreatedBy      *uint     `gorm:"column:created_by"`
	UpdatedBy      *uint     `gorm:"column:updated_by"`
	Version        uint64    `gorm:"not null;default:1"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (articlePO) TableName() string { return "knowledge_articles" }

func (po *articlePO) toDomain() *Article {
	return &Article{
		ID: po.ID, OrganizationID: po.OrganizationID, Title: po.Title, Body: po.Body,
		CreatedBy: po.CreatedBy, UpdatedBy: po.UpdatedBy, Version: po.Version,
		CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}
