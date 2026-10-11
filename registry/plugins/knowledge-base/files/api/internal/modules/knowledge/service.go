package knowledge

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fastagi/fastbuild/api/internal/domain"
	auditstarter "github.com/fastagi/fastbuild/api/internal/modules/audit"
)

// ArticleChanges is a partial update; nil fields stay unchanged.
type ArticleChanges struct {
	Title *string
	Body  *string
}

// Service owns article rules. Callers pass the verified organization and actor; role checks happen
// in the handler, where the organization context is resolved.
type Service interface {
	List(ctx context.Context, organizationID uint, after *ArticleCursor, limit int) ([]*Article, *ArticleCursor, error)
	Get(ctx context.Context, organizationID, id uint) (*Article, error)
	Create(ctx context.Context, organizationID, actorID uint, title, body string) (*Article, error)
	Update(ctx context.Context, organizationID, actorID, id uint, expectedVersion uint64, changes ArticleChanges) (*Article, error)
	Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error
	Search(ctx context.Context, organizationID uint, query string) ([]SearchHit, error)
}

type service struct {
	store store
}

// NewService returns the knowledge service.
func NewService(store store) *service {
	return &service{store: store}
}

func (s *service) List(ctx context.Context, organizationID uint, after *ArticleCursor, limit int) ([]*Article, *ArticleCursor, error) {
	if organizationID == 0 || limit < 1 || limit > 100 {
		return nil, nil, domain.ErrInvalidInput
	}
	return s.store.List(ctx, organizationID, after, limit)
}

func (s *service) Get(ctx context.Context, organizationID, id uint) (*Article, error) {
	if organizationID == 0 || id == 0 {
		return nil, ErrArticleNotFound
	}
	return s.store.Get(ctx, organizationID, id)
}

func (s *service) Create(ctx context.Context, organizationID, actorID uint, title, body string) (*Article, error) {
	title = strings.TrimSpace(title)
	if organizationID == 0 || !validTitle(title) || !validBody(body) {
		return nil, ErrInvalidArticle
	}
	article := &Article{OrganizationID: organizationID, Title: title, Body: body, CreatedBy: &actorID}
	if err := s.store.Create(ctx, article); err != nil {
		return nil, err
	}
	recordChange(ctx, "create", article.ID, map[string]domain.AuditValueChange{
		"title": {After: article.Title},
	})
	return article, nil
}

func (s *service) Update(
	ctx context.Context,
	organizationID, actorID, id uint,
	expectedVersion uint64,
	changes ArticleChanges,
) (*Article, error) {
	if changes.Title == nil && changes.Body == nil {
		return nil, ErrInvalidArticle
	}
	if changes.Title != nil {
		trimmed := strings.TrimSpace(*changes.Title)
		if !validTitle(trimmed) {
			return nil, ErrInvalidArticle
		}
		changes.Title = &trimmed
	}
	if changes.Body != nil && !validBody(*changes.Body) {
		return nil, ErrInvalidArticle
	}
	if organizationID == 0 || id == 0 {
		return nil, ErrArticleNotFound
	}
	article, err := s.store.Update(ctx, organizationID, id, expectedVersion, changes, actorID)
	if err != nil {
		return nil, err
	}
	// The body is business content: record that it changed, never its text.
	changed := map[string]domain.AuditValueChange{}
	if changes.Title != nil {
		changed["title"] = domain.AuditValueChange{After: article.Title}
	}
	if changes.Body != nil {
		changed["body"] = domain.AuditValueChange{After: "[changed]"}
	}
	recordChange(ctx, "update", article.ID, changed)
	return article, nil
}

func (s *service) Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error {
	if organizationID == 0 || id == 0 {
		return ErrArticleNotFound
	}
	if err := s.store.Delete(ctx, organizationID, id, expectedVersion); err != nil {
		return err
	}
	recordChange(ctx, "delete", id, nil)
	return nil
}

func (s *service) Search(ctx context.Context, organizationID uint, query string) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if organizationID == 0 || query == "" || utf8.RuneCountInString(query) > maxQueryLength {
		return nil, domain.ErrInvalidInput
	}
	return s.store.Search(ctx, organizationID, query, maxSearchResults)
}

func validTitle(title string) bool {
	length := utf8.RuneCountInString(title)
	return length >= 1 && length <= maxTitleLength
}

func validBody(body string) bool {
	return utf8.ValidString(body) && utf8.RuneCountInString(body) <= maxBodyLength
}

func recordChange(ctx context.Context, action string, id uint, changes map[string]domain.AuditValueChange) {
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "knowledge_articles",
		TargetType: "knowledge_article",
		TargetID:   strconv.FormatUint(uint64(id), 10),
		Changes:    changes,
	})
}
