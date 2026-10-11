package tag

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/fastagi/fastbuild/api/internal/domain"
	infradatabase "github.com/fastagi/fastbuild/api/internal/infra/database"
)

var tagColumns = []string{
	"id", "organization_id", "name", "color", "description", "created_by", "version", "created_at", "updated_at",
}

// sortKeyExpression orders tags by lowercase name with byte-wise collation, so keyset pages are
// identical on every PostgreSQL locale. The unique index uses the same expression.
const sortKeyExpression = `lower(name) COLLATE "C"`

type store interface {
	List(ctx context.Context, organizationID uint, after *TagCursor, limit int) ([]*Tag, *TagCursor, error)
	Get(ctx context.Context, organizationID, id uint) (*Tag, error)
	Create(ctx context.Context, tag *Tag) error
	Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes TagChanges) (*Tag, error)
	Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error
	ResourceTags(ctx context.Context, organizationID uint, ref ResourceRef) ([]*Tag, error)
	// ReplaceResourceTags makes tagIDs the resource's complete tag set and returns the previous tag
	// IDs and the new tags.
	ReplaceResourceTags(ctx context.Context, organizationID uint, ref ResourceRef, tagIDs []uint) ([]uint, []*Tag, error)
	TaggedResources(
		ctx context.Context, organizationID, tagID uint, resourceType string, after *AssignmentCursor, limit int,
	) ([]TaggedResource, *AssignmentCursor, error)
	// DetachResource removes the resource's assignments and returns the tag IDs it had.
	DetachResource(ctx context.Context, organizationID uint, ref ResourceRef) ([]uint, error)
}

type repository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewRepository returns the PostgreSQL store for tags and assignments.
func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db, now: time.Now}
}

func (r *repository) withContext(ctx context.Context) (*gorm.DB, error) {
	db := infradatabase.ResolveContextDB(ctx, r.db)
	if db == nil {
		return nil, domain.ErrServiceUnavailable
	}
	return db.WithContext(ctx), nil
}

// List returns one keyset page of an organization's tags ordered by lowercase name.
func (r *repository) List(ctx context.Context, organizationID uint, after *TagCursor, limit int) ([]*Tag, *TagCursor, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := db.Model(&tagPO{}).
		Select(append(slices.Clone(tagColumns), sortKeyExpression+" AS sort_key")).
		Where("organization_id = ?", organizationID)
	if after != nil {
		query = query.Where(sortKeyExpression+" > ?", after.SortKey)
	}
	var rows []tagListRow
	if err := query.Order(sortKeyExpression).Limit(limit + 1).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *TagCursor
	if len(rows) > limit {
		rows = rows[:limit]
		next = &TagCursor{SortKey: rows[limit-1].SortKey}
	}
	tags := make([]*Tag, len(rows))
	for index, row := range rows {
		tags[index] = &Tag{
			ID: row.ID, OrganizationID: row.OrganizationID, Name: row.Name, Color: row.Color,
			Description: row.Description, CreatedBy: row.CreatedBy, Version: row.Version,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return tags, next, nil
}

// tagListRow is a tag with its sort key. The cursor carries the database's own lowercase form,
// which Go's case mapping may not match.
type tagListRow struct {
	ID             uint
	OrganizationID uint
	Name           string
	Color          string
	Description    string
	CreatedBy      *uint
	Version        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	SortKey        string
}

// Get returns one tag of the organization.
func (r *repository) Get(ctx context.Context, organizationID, id uint) (*Tag, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var row tagPO
	err = db.Select(tagColumns).Where("organization_id = ? AND id = ?", organizationID, id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

// Create inserts a new tag at version 1.
func (r *repository) Create(ctx context.Context, tag *Tag) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	row := tagPO{
		OrganizationID: tag.OrganizationID, Name: tag.Name, Color: tag.Color, Description: tag.Description,
		CreatedBy: tag.CreatedBy, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	err = db.Select("organization_id", "name", "color", "description", "created_by", "version", "created_at", "updated_at").
		Create(&row).Error
	if isUniqueViolation(err) {
		return ErrNameTaken
	}
	if err != nil {
		return err
	}
	*tag = *row.toDomain()
	return nil
}

// Update applies changes when the stored version still matches expectedVersion.
func (r *repository) Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes TagChanges) (*Tag, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"updated_at": r.now().UTC(),
		"version":    gorm.Expr("version + 1"),
	}
	if changes.Name != nil {
		values["name"] = *changes.Name
	}
	if changes.Color != nil {
		values["color"] = *changes.Color
	}
	if changes.Description != nil {
		values["description"] = *changes.Description
	}
	result := db.Model(&tagPO{}).
		Where("organization_id = ? AND id = ? AND version = ?", organizationID, id, expectedVersion).
		Updates(values)
	if isUniqueViolation(result.Error) {
		return nil, ErrNameTaken
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, r.missOrConflict(ctx, organizationID, id)
	}
	return r.Get(ctx, organizationID, id)
}

// Delete removes the tag and, through the foreign key, its assignments when the stored version
// still matches expectedVersion.
func (r *repository) Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error {
	db, err := r.withContext(ctx)
	if err != nil {
		return err
	}
	result := db.Where("organization_id = ? AND id = ? AND version = ?", organizationID, id, expectedVersion).
		Delete(&tagPO{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.missOrConflict(ctx, organizationID, id)
	}
	return nil
}

// missOrConflict tells a stale version apart from a tag that is not in the organization.
func (r *repository) missOrConflict(ctx context.Context, organizationID, id uint) error {
	if _, err := r.Get(ctx, organizationID, id); err != nil {
		return err
	}
	return ErrVersionConflict
}

// ResourceTags returns the tags a resource carries, ordered by lowercase name.
func (r *repository) ResourceTags(ctx context.Context, organizationID uint, ref ResourceRef) ([]*Tag, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	return resourceTags(db, organizationID, ref)
}

func resourceTags(db *gorm.DB, organizationID uint, ref ResourceRef) ([]*Tag, error) {
	var rows []tagPO
	err := db.Model(&tagPO{}).
		Select(prefixed("tags", tagColumns)).
		Joins("JOIN tag_assignments ON tag_assignments.organization_id = tags.organization_id AND tag_assignments.tag_id = tags.id").
		Where("tags.organization_id = ? AND tag_assignments.resource_type = ? AND tag_assignments.resource_id = ?",
			organizationID, ref.Type, ref.ID).
		Order(`lower(tags.name) COLLATE "C"`).
		Limit(maxTagsPerResource + 1).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tags := make([]*Tag, len(rows))
	for index := range rows {
		tags[index] = rows[index].toDomain()
	}
	return tags, nil
}

// ReplaceResourceTags serializes writers of one resource with a transaction-scoped advisory lock,
// so concurrent replacements end with exactly one writer's set instead of a union. The tags are
// read FOR SHARE, so a concurrent tag delete waits instead of racing the insert.
func (r *repository) ReplaceResourceTags(
	ctx context.Context,
	organizationID uint,
	ref ResourceRef,
	tagIDs []uint,
) ([]uint, []*Tag, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	var previous []uint
	var tags []*Tag
	err = db.Transaction(func(tx *gorm.DB) error {
		if lockErr := lockResource(tx, organizationID, ref); lockErr != nil {
			return lockErr
		}
		if len(tagIDs) > 0 {
			var found []uint
			if findErr := tx.Raw(`SELECT id FROM tags WHERE organization_id = ? AND id IN ? FOR SHARE`, organizationID, tagIDs).
				Scan(&found).Error; findErr != nil {
				return findErr
			}
			if len(found) != len(tagIDs) {
				return ErrInvalidAssignment
			}
		}
		if scanErr := tx.Model(&assignmentPO{}).
			Where("organization_id = ? AND resource_type = ? AND resource_id = ?", organizationID, ref.Type, ref.ID).
			Pluck("tag_id", &previous).Error; scanErr != nil {
			return scanErr
		}
		stale := tx.Where("organization_id = ? AND resource_type = ? AND resource_id = ?", organizationID, ref.Type, ref.ID)
		if len(tagIDs) > 0 {
			stale = stale.Where("tag_id NOT IN ?", tagIDs)
		}
		if deleteErr := stale.Delete(&assignmentPO{}).Error; deleteErr != nil {
			return deleteErr
		}
		now := r.now().UTC()
		for _, tagID := range tagIDs {
			if slices.Contains(previous, tagID) {
				continue
			}
			row := assignmentPO{OrganizationID: organizationID, TagID: tagID, ResourceType: ref.Type, ResourceID: ref.ID, CreatedAt: now}
			if createErr := tx.Select("organization_id", "tag_id", "resource_type", "resource_id", "created_at").
				Create(&row).Error; createErr != nil {
				return createErr
			}
		}
		var listErr error
		tags, listErr = resourceTags(tx, organizationID, ref)
		return listErr
	})
	if err != nil {
		return nil, nil, err
	}
	if previous == nil {
		previous = []uint{}
	}
	return previous, tags, nil
}

// TaggedResources returns one keyset page of the resources carrying a tag, newest assignment first.
func (r *repository) TaggedResources(
	ctx context.Context,
	organizationID, tagID uint,
	resourceType string,
	after *AssignmentCursor,
	limit int,
) ([]TaggedResource, *AssignmentCursor, error) {
	if _, err := r.Get(ctx, organizationID, tagID); err != nil {
		return nil, nil, err
	}
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	query := db.Model(&assignmentPO{}).Where("organization_id = ? AND tag_id = ?", organizationID, tagID)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if after != nil {
		query = query.Where("id < ?", after.ID)
	}
	var rows []assignmentPO
	if err := query.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *AssignmentCursor
	if len(rows) > limit {
		rows = rows[:limit]
		next = &AssignmentCursor{ID: rows[limit-1].ID}
	}
	resources := make([]TaggedResource, len(rows))
	for index, row := range rows {
		resources[index] = TaggedResource{
			AssignmentID: row.ID, ResourceType: row.ResourceType, ResourceID: row.ResourceID, TaggedAt: row.CreatedAt,
		}
	}
	return resources, next, nil
}

// DetachResource removes every assignment of the resource.
func (r *repository) DetachResource(ctx context.Context, organizationID uint, ref ResourceRef) ([]uint, error) {
	db, err := r.withContext(ctx)
	if err != nil {
		return nil, err
	}
	var previous []uint
	err = db.Transaction(func(tx *gorm.DB) error {
		if lockErr := lockResource(tx, organizationID, ref); lockErr != nil {
			return lockErr
		}
		return tx.Raw(`DELETE FROM tag_assignments
			WHERE organization_id = ? AND resource_type = ? AND resource_id = ?
			RETURNING tag_id`, organizationID, ref.Type, ref.ID).Scan(&previous).Error
	})
	if err != nil {
		return nil, err
	}
	return previous, nil
}

// lockResource takes a transaction-scoped advisory lock for one resource's tag set. Hash
// collisions only serialize unrelated resources; they never change results.
func lockResource(tx *gorm.DB, organizationID uint, ref ResourceRef) error {
	key := "tag_assignments:" + strconv.FormatUint(uint64(organizationID), 10) + ":" + ref.Type + ":" + ref.ID
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, key).Error
}

func prefixed(table string, columns []string) []string {
	out := make([]string, len(columns))
	for index, column := range columns {
		out[index] = table + "." + column
	}
	return out
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var stateError interface{ SQLState() string }
	if errors.As(err, &stateError) && stateError.SQLState() == "23505" {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}
