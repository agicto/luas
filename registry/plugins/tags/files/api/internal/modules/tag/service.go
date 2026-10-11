package tag

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fastagi/fastbuild/api/internal/domain"
	auditstarter "github.com/fastagi/fastbuild/api/internal/modules/audit"
)

// TagInput is a new tag. An empty color defaults to gray.
type TagInput struct {
	Name        string
	Color       string
	Description string
}

// TagChanges is a partial update; nil fields stay unchanged.
type TagChanges struct {
	Name        *string
	Color       *string
	Description *string
}

// Service owns tag and assignment rules. Callers pass the verified organization and actor; role
// checks happen in the handler, where the organization context is resolved. Other modules may
// inject Service to call DetachResource when they delete a tagged resource.
type Service interface {
	List(ctx context.Context, organizationID uint, after *TagCursor, limit int) ([]*Tag, *TagCursor, error)
	Get(ctx context.Context, organizationID, id uint) (*Tag, error)
	Create(ctx context.Context, organizationID, actorID uint, input TagInput) (*Tag, error)
	Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes TagChanges) (*Tag, error)
	Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error
	ResourceTags(ctx context.Context, organizationID uint, ref ResourceRef) ([]*Tag, error)
	ReplaceResourceTags(ctx context.Context, organizationID uint, ref ResourceRef, tagIDs []uint) ([]*Tag, error)
	TaggedResources(
		ctx context.Context, organizationID, tagID uint, resourceType string, after *AssignmentCursor, limit int,
	) ([]TaggedResource, *AssignmentCursor, error)
	DetachResource(ctx context.Context, organizationID uint, ref ResourceRef) error
}

type service struct {
	store  store
	types  *ResourceTypes
	record func(context.Context, auditstarter.Change)
}

// NewService returns the tag service.
func NewService(store store, types *ResourceTypes) *service {
	return &service{store: store, types: types, record: auditstarter.RecordChange}
}

func (s *service) List(ctx context.Context, organizationID uint, after *TagCursor, limit int) ([]*Tag, *TagCursor, error) {
	if organizationID == 0 || limit < 1 || limit > 100 {
		return nil, nil, domain.ErrInvalidInput
	}
	return s.store.List(ctx, organizationID, after, limit)
}

func (s *service) Get(ctx context.Context, organizationID, id uint) (*Tag, error) {
	if organizationID == 0 || id == 0 {
		return nil, ErrNotFound
	}
	return s.store.Get(ctx, organizationID, id)
}

func (s *service) Create(ctx context.Context, organizationID, actorID uint, input TagInput) (*Tag, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Color == "" {
		input.Color = defaultColor
	}
	if organizationID == 0 || !validName(input.Name) || !ValidColor(input.Color) || !validDescription(input.Description) {
		return nil, ErrInvalid
	}
	tag := &Tag{
		OrganizationID: organizationID, Name: input.Name, Color: input.Color,
		Description: input.Description, CreatedBy: &actorID,
	}
	if err := s.store.Create(ctx, tag); err != nil {
		return nil, err
	}
	changes := map[string]domain.AuditValueChange{
		"name":  {After: tag.Name},
		"color": {After: tag.Color},
	}
	if tag.Description != "" {
		changes["description"] = domain.AuditValueChange{After: "[changed]"}
	}
	s.recordTag(ctx, "create", tag.ID, changes)
	return tag, nil
}

func (s *service) Update(ctx context.Context, organizationID, id uint, expectedVersion uint64, changes TagChanges) (*Tag, error) {
	if changes.Name == nil && changes.Color == nil && changes.Description == nil {
		return nil, ErrInvalid
	}
	if changes.Name != nil {
		trimmed := strings.TrimSpace(*changes.Name)
		if !validName(trimmed) {
			return nil, ErrInvalid
		}
		changes.Name = &trimmed
	}
	if (changes.Color != nil && !ValidColor(*changes.Color)) || (changes.Description != nil && !validDescription(*changes.Description)) {
		return nil, ErrInvalid
	}
	if organizationID == 0 || id == 0 {
		return nil, ErrNotFound
	}
	tag, err := s.store.Update(ctx, organizationID, id, expectedVersion, changes)
	if err != nil {
		return nil, err
	}
	// Descriptions are free text: record that one changed, never its content.
	changed := map[string]domain.AuditValueChange{}
	if changes.Name != nil {
		changed["name"] = domain.AuditValueChange{After: tag.Name}
	}
	if changes.Color != nil {
		changed["color"] = domain.AuditValueChange{After: tag.Color}
	}
	if changes.Description != nil {
		changed["description"] = domain.AuditValueChange{After: "[changed]"}
	}
	s.recordTag(ctx, "update", tag.ID, changed)
	return tag, nil
}

func (s *service) Delete(ctx context.Context, organizationID, id uint, expectedVersion uint64) error {
	if organizationID == 0 || id == 0 {
		return ErrNotFound
	}
	if err := s.store.Delete(ctx, organizationID, id, expectedVersion); err != nil {
		return err
	}
	s.recordTag(ctx, "delete", id, nil)
	return nil
}

func (s *service) ResourceTags(ctx context.Context, organizationID uint, ref ResourceRef) ([]*Tag, error) {
	if err := s.checkResource(ctx, organizationID, ref); err != nil {
		return nil, err
	}
	return s.store.ResourceTags(ctx, organizationID, ref)
}

func (s *service) ReplaceResourceTags(ctx context.Context, organizationID uint, ref ResourceRef, tagIDs []uint) ([]*Tag, error) {
	if len(tagIDs) > maxTagsPerResource || slices.Contains(tagIDs, 0) || hasDuplicates(tagIDs) {
		return nil, ErrInvalidAssignment
	}
	if err := s.checkResource(ctx, organizationID, ref); err != nil {
		return nil, err
	}
	previous, tags, err := s.store.ReplaceResourceTags(ctx, organizationID, ref, tagIDs)
	if err != nil {
		return nil, err
	}
	current := make([]uint, len(tags))
	for index, tag := range tags {
		current[index] = tag.ID
	}
	slices.Sort(previous)
	slices.Sort(current)
	if !slices.Equal(previous, current) {
		s.recordAssignment(ctx, ref, previous, current)
	}
	return tags, nil
}

func (s *service) TaggedResources(
	ctx context.Context,
	organizationID, tagID uint,
	resourceType string,
	after *AssignmentCursor,
	limit int,
) ([]TaggedResource, *AssignmentCursor, error) {
	if limit < 1 || limit > 100 {
		return nil, nil, domain.ErrInvalidInput
	}
	if resourceType != "" && !resourceTypePattern.MatchString(resourceType) {
		return nil, nil, domain.ErrInvalidInput
	}
	if organizationID == 0 || tagID == 0 {
		return nil, nil, ErrNotFound
	}
	return s.store.TaggedResources(ctx, organizationID, tagID, resourceType, after, limit)
}

// DetachResource removes every tag from a resource. Owning modules call it when they delete a
// resource; it does not consult the checker because the resource may already be gone.
func (s *service) DetachResource(ctx context.Context, organizationID uint, ref ResourceRef) error {
	if organizationID == 0 || !ValidResourceRef(ref) {
		return domain.ErrInvalidInput
	}
	previous, err := s.store.DetachResource(ctx, organizationID, ref)
	if err != nil {
		return err
	}
	if len(previous) > 0 {
		slices.Sort(previous)
		s.recordAssignment(ctx, ref, previous, []uint{})
	}
	return nil
}

func (s *service) checkResource(ctx context.Context, organizationID uint, ref ResourceRef) error {
	if organizationID == 0 {
		return ErrResourceNotFound
	}
	return s.types.Exists(ctx, organizationID, ref)
}

func (s *service) recordTag(ctx context.Context, action string, id uint, changes map[string]domain.AuditValueChange) {
	s.record(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "tags",
		TargetType: "tag",
		TargetID:   strconv.FormatUint(uint64(id), 10),
		Changes:    changes,
	})
}

// recordAssignment records which tag IDs a resource gained or lost; IDs carry no business text.
func (s *service) recordAssignment(ctx context.Context, ref ResourceRef, previous, current []uint) {
	s.record(ctx, auditstarter.Change{
		Action:     "update",
		Resource:   "tag_assignments",
		TargetType: ref.Type,
		TargetID:   ref.ID,
		Changes: map[string]domain.AuditValueChange{
			"tag_ids": {Before: previous, After: current},
		},
	})
}

// validName accepts 1-50 characters of single-line text.
func validName(name string) bool {
	length := utf8.RuneCountInString(name)
	return utf8.ValidString(name) && length >= 1 && length <= maxNameLength && !strings.ContainsFunc(name, unicode.IsControl)
}

// validDescription accepts up to 200 characters; PostgreSQL text cannot hold NUL.
func validDescription(description string) bool {
	return utf8.ValidString(description) && utf8.RuneCountInString(description) <= maxDescriptionLength &&
		!strings.ContainsRune(description, 0)
}

func hasDuplicates(ids []uint) bool {
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}
