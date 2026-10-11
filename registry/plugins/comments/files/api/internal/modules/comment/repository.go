package comment

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/domain"
	infradatabase "github.com/fastagi/fastbuild/api/internal/infra/database"
)

var commentColumns = []string{
	"id", "organization_id", "resource_type", "resource_id", "parent_id", "author_id", "body", "mentions",
	"version", "created_at", "updated_at", "edited_at", "deleted_at",
}

// CommentChanges is a partial edit; nil fields stay unchanged.
type CommentChanges struct {
	Body     *string
	Mentions *[]uint
}

type store interface {
	ListRoots(ctx context.Context, organizationID uint, resourceType, resourceID string, after *Cursor, limit int) ([]*Comment, *Cursor, error)
	// ListReplyPreviews returns up to limit replies per root, oldest first.
	ListReplyPreviews(ctx context.Context, organizationID uint, rootIDs []uint, limit int) (map[uint][]*Comment, error)
	ListReplies(ctx context.Context, organizationID, parentID uint, after *Cursor, limit int) ([]*Comment, *Cursor, error)
	// Get returns a comment of the organization, including a deleted one.
	Get(ctx context.Context, organizationID, id uint) (*Comment, error)
	// Create inserts a comment. A reply's ParentID must name a live root comment on the same
	// resource; the root is locked so a concurrent delete cannot slip between check and insert.
	Create(ctx context.Context, comment *Comment) error
	Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes CommentChanges) (*Comment, error)
	// SoftDelete erases the body and mentions and marks the comment deleted.
	SoftDelete(ctx context.Context, organizationID, id uint) (*Comment, error)
}

type repository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewRepository returns the PostgreSQL store for comments.
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

// ListRoots returns one keyset page of a resource's root comments, oldest first.
func (r *repository) ListRoots(
	ctx context.Context,
	organizationID uint,
	resourceType, resourceID string,
	after *Cursor,
	limit int,
) ([]*Comment, *Cursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := db.Model(&commentPO{}).Select(commentColumns).
		Where("organization_id = ? AND resource_type = ? AND resource_id = ? AND parent_id IS NULL",
			organizationID, resourceType, resourceID)
	return r.page(query, after, limit)
}

// ListReplies returns one keyset page of a comment's replies, oldest first.
func (r *repository) ListReplies(ctx context.Context, organizationID, parentID uint, after *Cursor, limit int) ([]*Comment, *Cursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := db.Model(&commentPO{}).Select(commentColumns).
		Where("organization_id = ? AND parent_id = ?", organizationID, parentID)
	return r.page(query, after, limit)
}

func (r *repository) page(query *gorm.DB, after *Cursor, limit int) ([]*Comment, *Cursor, error) {
	if after != nil {
		query = query.Where("id > ?", after.ID)
	}
	var rows []commentPO
	if err := query.Order("id ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *Cursor
	if len(rows) > limit {
		rows = rows[:limit]
		next = &Cursor{ID: rows[limit-1].ID}
	}
	comments, err := toDomainList(rows)
	return comments, next, err
}

// ListReplyPreviews reads the first replies of every root in one bounded query.
func (r *repository) ListReplyPreviews(ctx context.Context, organizationID uint, rootIDs []uint, limit int) (map[uint][]*Comment, error) {
	previews := make(map[uint][]*Comment, len(rootIDs))
	if len(rootIDs) == 0 {
		return previews, nil
	}
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var rows []commentPO
	err = db.Raw(`
		SELECT reply.id, reply.organization_id, reply.resource_type, reply.resource_id, reply.parent_id,
		       reply.author_id, reply.body, reply.mentions, reply.version, reply.created_at, reply.updated_at,
		       reply.edited_at, reply.deleted_at
		FROM comments AS root
		CROSS JOIN LATERAL (
		    SELECT * FROM comments AS child
		    WHERE child.organization_id = root.organization_id AND child.parent_id = root.id
		    ORDER BY child.id ASC
		    LIMIT ?
		) AS reply
		WHERE root.organization_id = ? AND root.id IN ?
		ORDER BY reply.parent_id, reply.id`,
		limit, organizationID, rootIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	replies, err := toDomainList(rows)
	if err != nil {
		return nil, err
	}
	for _, reply := range replies {
		previews[*reply.ParentID] = append(previews[*reply.ParentID], reply)
	}
	return previews, nil
}

// Get returns one comment of the organization.
func (r *repository) Get(ctx context.Context, organizationID, id uint) (*Comment, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var row commentPO
	err = db.Select(commentColumns).Where("organization_id = ? AND id = ?", organizationID, id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain()
}

// Create inserts a comment at version 1.
func (r *repository) Create(ctx context.Context, comment *Comment) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	body := comment.Body
	row := commentPO{
		OrganizationID: comment.OrganizationID, ResourceType: comment.ResourceType, ResourceID: comment.ResourceID,
		ParentID: comment.ParentID, AuthorID: comment.AuthorID, Body: &body, Mentions: encodeMentions(comment.Mentions),
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if row.ParentID != nil {
			var roots []uint
			// FOR SHARE conflicts with the row lock SoftDelete takes, so a reply never lands on a
			// root that was deleted after the check.
			if lockErr := tx.Raw(`
				SELECT id FROM comments
				WHERE organization_id = ? AND id = ? AND resource_type = ? AND resource_id = ?
				  AND parent_id IS NULL AND deleted_at IS NULL
				FOR SHARE`,
				row.OrganizationID, *row.ParentID, row.ResourceType, row.ResourceID,
			).Scan(&roots).Error; lockErr != nil {
				return lockErr
			}
			if len(roots) == 0 {
				return ErrInvalidParent
			}
		}
		return tx.Select("organization_id", "resource_type", "resource_id", "parent_id", "author_id", "body",
			"mentions", "version", "created_at", "updated_at").Create(&row).Error
	})
	if err != nil {
		return err
	}
	created, err := row.toDomain()
	if err != nil {
		return err
	}
	*comment = *created
	return nil
}

// Update edits a live comment when the stored version still matches expectedVersion.
func (r *repository) Update(
	ctx context.Context,
	organizationID, id uint,
	expectedVersion uint64,
	changes CommentChanges,
) (*Comment, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	values := map[string]any{
		"updated_at": now,
		"edited_at":  now,
		"version":    gorm.Expr("version + 1"),
	}
	if changes.Body != nil {
		values["body"] = *changes.Body
	}
	if changes.Mentions != nil {
		values["mentions"] = encodeMentions(*changes.Mentions)
	}
	result := db.Model(&commentPO{}).
		Where("organization_id = ? AND id = ? AND version = ? AND deleted_at IS NULL", organizationID, id, expectedVersion).
		Updates(values)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		current, err := r.Get(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
		if current.IsDeleted() {
			return nil, ErrCommentNotFound
		}
		return nil, ErrVersionConflict
	}
	return r.Get(ctx, organizationID, id)
}

// SoftDelete erases a live comment's content and keeps its place in the thread.
func (r *repository) SoftDelete(ctx context.Context, organizationID, id uint) (*Comment, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	now := r.now().UTC()
	result := db.Model(&commentPO{}).
		Where("organization_id = ? AND id = ? AND deleted_at IS NULL", organizationID, id).
		Updates(map[string]any{
			"body":       nil,
			"mentions":   encodeMentions(nil),
			"deleted_at": now,
			"updated_at": now,
			"version":    gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrCommentNotFound
	}
	return r.Get(ctx, organizationID, id)
}

func toDomainList(rows []commentPO) ([]*Comment, error) {
	comments := make([]*Comment, len(rows))
	for index := range rows {
		comment, err := rows[index].toDomain()
		if err != nil {
			return nil, err
		}
		comments[index] = comment
	}
	return comments, nil
}
