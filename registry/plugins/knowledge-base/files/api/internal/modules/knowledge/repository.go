package knowledge

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/domain"
	infradatabase "github.com/fastagi/fastbuild/api/internal/infra/database"
)

// articleColumns excludes the generated search_vector column.
var articleColumns = []string{
	"id", "organization_id", "title", "body", "created_by", "updated_by", "version", "created_at", "updated_at",
}

type store interface {
	List(ctx context.Context, organizationID uint, after *ArticleCursor, limit int) ([]*Article, *ArticleCursor, error)
	Get(ctx context.Context, organizationID, id uint) (*Article, error)
	Create(ctx context.Context, article *Article) error
	Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes ArticleChanges, actorID uint) (*Article, error)
	Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error
	Search(ctx context.Context, organizationID uint, query string, limit int) ([]SearchHit, error)
}

type repository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewRepository returns the PostgreSQL store for knowledge articles.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db, now: time.Now}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	db := infradatabase.ResolveContextDB(ctx, r.db)
	if db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return db, nil
}

// List returns one keyset page of an organization's articles, newest first.
func (r *repository) List(ctx context.Context, organizationID uint, after *ArticleCursor, limit int) ([]*Article, *ArticleCursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := db.Model(&articlePO{}).Select(articleColumns).Where("organization_id = ?", organizationID)
	if after != nil {
		query = query.Where("id < ?", after.ID)
	}
	var rows []articlePO
	if err := query.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *ArticleCursor
	if len(rows) > limit {
		rows = rows[:limit]
		next = &ArticleCursor{ID: rows[limit-1].ID}
	}
	articles := make([]*Article, len(rows))
	for index := range rows {
		articles[index] = rows[index].toDomain()
	}
	return articles, next, nil
}

// Get returns one article of the organization.
func (r *repository) Get(ctx context.Context, organizationID, id uint) (*Article, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var row articlePO
	err = db.Select(articleColumns).Where("organization_id = ? AND id = ?", organizationID, id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

// Create inserts a new article at version 1.
func (r *repository) Create(ctx context.Context, article *Article) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	row := articlePO{
		OrganizationID: article.OrganizationID, Title: article.Title, Body: article.Body,
		CreatedBy: article.CreatedBy, UpdatedBy: article.CreatedBy, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Select("organization_id", "title", "body", "created_by", "updated_by", "version", "created_at", "updated_at").
		Create(&row).Error; err != nil {
		return err
	}
	*article = *row.toDomain()
	return nil
}

// Update applies changes when the stored version still matches expectedVersion.
func (r *repository) Update(
	ctx context.Context,
	organizationID, id uint,
	expectedVersion uint64,
	changes ArticleChanges,
	actorID uint,
) (*Article, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"updated_by": actorID,
		"updated_at": r.now().UTC(),
		"version":    gorm.Expr("version + 1"),
	}
	if changes.Title != nil {
		values["title"] = *changes.Title
	}
	if changes.Body != nil {
		values["body"] = *changes.Body
	}
	result := db.Model(&articlePO{}).
		Where("organization_id = ? AND id = ? AND version = ?", organizationID, id, expectedVersion).
		Updates(values)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, r.missOrConflict(ctx, organizationID, id)
	}
	return r.Get(ctx, organizationID, id)
}

// Delete removes the article when the stored version still matches expectedVersion.
func (r *repository) Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	result := db.Where("organization_id = ? AND id = ? AND version = ?", organizationID, id, expectedVersion).
		Delete(&articlePO{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.missOrConflict(ctx, organizationID, id)
	}
	return nil
}

// missOrConflict tells a stale version apart from an article that is not in the organization.
func (r *repository) missOrConflict(ctx context.Context, organizationID, id uint) error {
	if _, err := r.Get(ctx, organizationID, id); err != nil {
		return err
	}
	return ErrVersionConflict
}

// Search ranks the organization's articles by full-text match and also finds plain substrings,
// which covers languages written without spaces between words, such as Chinese.
func (r *repository) Search(ctx context.Context, organizationID uint, query string, limit int) ([]SearchHit, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	pattern := "%" + escapeLike(query) + "%"
	var rows []struct {
		ID        uint
		Title     string
		Excerpt   string
		Version   uint64
		UpdatedAt time.Time
	}
	err = db.Raw(`
		SELECT id, title, left(body, ?) AS excerpt, version, updated_at
		FROM knowledge_articles
		WHERE organization_id = ?
		  AND (search_vector @@ websearch_to_tsquery('simple', ?)
		       OR title ILIKE ? ESCAPE '\'
		       OR body ILIKE ? ESCAPE '\')
		ORDER BY ts_rank_cd(search_vector, websearch_to_tsquery('simple', ?)) DESC, id DESC
		LIMIT ?`,
		excerptLength, organizationID, query, pattern, pattern, query, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	hits := make([]SearchHit, len(rows))
	for index, row := range rows {
		hits[index] = SearchHit{ID: row.ID, Title: row.Title, Excerpt: row.Excerpt, Version: row.Version, UpdatedAt: row.UpdatedAt}
	}
	return hits, nil
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
