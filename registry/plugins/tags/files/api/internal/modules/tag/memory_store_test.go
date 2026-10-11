package tag

import (
	"context"
	"slices"
	"strings"
	"time"
)

// memoryStore is an in-memory store that enforces the same organization scope, name uniqueness,
// versions, and ordering as the PostgreSQL repository.
type memoryStore struct {
	tags        map[uint]*Tag
	assignments []assignmentPO
	nextID      uint
	nextAssign  uint
}

func newMemoryStore() *memoryStore { return &memoryStore{tags: map[uint]*Tag{}} }

func (s *memoryStore) sorted(organizationID uint) []*Tag {
	var tags []*Tag
	for _, tag := range s.tags {
		if tag.OrganizationID == organizationID {
			copied := *tag
			tags = append(tags, &copied)
		}
	}
	slices.SortFunc(tags, func(a, b *Tag) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return tags
}

func (s *memoryStore) List(_ context.Context, organizationID uint, after *TagCursor, limit int) ([]*Tag, *TagCursor, error) {
	var items []*Tag
	for _, tag := range s.sorted(organizationID) {
		if after != nil && strings.ToLower(tag.Name) <= after.SortKey {
			continue
		}
		if len(items) == limit {
			return items, &TagCursor{SortKey: strings.ToLower(items[len(items)-1].Name)}, nil
		}
		items = append(items, tag)
	}
	return items, nil, nil
}

func (s *memoryStore) Get(_ context.Context, organizationID, id uint) (*Tag, error) {
	tag, ok := s.tags[id]
	if !ok || tag.OrganizationID != organizationID {
		return nil, ErrNotFound
	}
	copied := *tag
	return &copied, nil
}

func (s *memoryStore) nameTaken(organizationID, exceptID uint, name string) bool {
	for _, tag := range s.tags {
		if tag.OrganizationID == organizationID && tag.ID != exceptID && strings.EqualFold(tag.Name, name) {
			return true
		}
	}
	return false
}

func (s *memoryStore) Create(_ context.Context, tag *Tag) error {
	if s.nameTaken(tag.OrganizationID, 0, tag.Name) {
		return ErrNameTaken
	}
	s.nextID++
	tag.ID, tag.Version, tag.CreatedAt, tag.UpdatedAt = s.nextID, 1, time.Now(), time.Now()
	stored := *tag
	s.tags[tag.ID] = &stored
	return nil
}

func (s *memoryStore) Update(ctx context.Context, organizationID, id uint, expected uint64, changes TagChanges) (*Tag, error) {
	tag, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if tag.Version != expected {
		return nil, ErrVersionConflict
	}
	if changes.Name != nil {
		if s.nameTaken(organizationID, id, *changes.Name) {
			return nil, ErrNameTaken
		}
		tag.Name = *changes.Name
	}
	if changes.Color != nil {
		tag.Color = *changes.Color
	}
	if changes.Description != nil {
		tag.Description = *changes.Description
	}
	tag.Version++
	s.tags[id] = tag
	return s.Get(ctx, organizationID, id)
}

func (s *memoryStore) Delete(ctx context.Context, organizationID, id uint, expected uint64) error {
	tag, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if tag.Version != expected {
		return ErrVersionConflict
	}
	delete(s.tags, id)
	s.assignments = slices.DeleteFunc(s.assignments, func(row assignmentPO) bool { return row.TagID == id })
	return nil
}

func (s *memoryStore) isResource(row assignmentPO, organizationID uint, ref ResourceRef) bool {
	return row.OrganizationID == organizationID && row.ResourceType == ref.Type && row.ResourceID == ref.ID
}

func (s *memoryStore) ResourceTags(_ context.Context, organizationID uint, ref ResourceRef) ([]*Tag, error) {
	var tags []*Tag
	for _, tag := range s.sorted(organizationID) {
		if slices.ContainsFunc(s.assignments, func(row assignmentPO) bool {
			return s.isResource(row, organizationID, ref) && row.TagID == tag.ID
		}) {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func (s *memoryStore) ReplaceResourceTags(ctx context.Context, organizationID uint, ref ResourceRef, tagIDs []uint) ([]uint, []*Tag, error) {
	for _, id := range tagIDs {
		if _, err := s.Get(ctx, organizationID, id); err != nil {
			return nil, nil, ErrInvalidAssignment
		}
	}
	previous := []uint{}
	kept := s.assignments[:0]
	for _, row := range s.assignments {
		if s.isResource(row, organizationID, ref) {
			previous = append(previous, row.TagID)
			if !slices.Contains(tagIDs, row.TagID) {
				continue
			}
		}
		kept = append(kept, row)
	}
	s.assignments = kept
	for _, id := range tagIDs {
		if !slices.Contains(previous, id) {
			s.nextAssign++
			s.assignments = append(s.assignments, assignmentPO{
				ID: s.nextAssign, OrganizationID: organizationID, TagID: id,
				ResourceType: ref.Type, ResourceID: ref.ID, CreatedAt: time.Now(),
			})
		}
	}
	tags, err := s.ResourceTags(ctx, organizationID, ref)
	return previous, tags, err
}

func (s *memoryStore) TaggedResources(
	ctx context.Context, organizationID, tagID uint, resourceType string, after *AssignmentCursor, limit int,
) ([]TaggedResource, *AssignmentCursor, error) {
	if _, err := s.Get(ctx, organizationID, tagID); err != nil {
		return nil, nil, err
	}
	var items []TaggedResource
	for index := len(s.assignments) - 1; index >= 0; index-- {
		row := s.assignments[index]
		if row.OrganizationID != organizationID || row.TagID != tagID ||
			(resourceType != "" && row.ResourceType != resourceType) || (after != nil && row.ID >= after.ID) {
			continue
		}
		if len(items) == limit {
			return items, &AssignmentCursor{ID: items[len(items)-1].AssignmentID}, nil
		}
		items = append(items, TaggedResource{row.ID, row.ResourceType, row.ResourceID, row.CreatedAt})
	}
	return items, nil, nil
}

func (s *memoryStore) DetachResource(_ context.Context, organizationID uint, ref ResourceRef) ([]uint, error) {
	var previous []uint
	s.assignments = slices.DeleteFunc(s.assignments, func(row assignmentPO) bool {
		if s.isResource(row, organizationID, ref) {
			previous = append(previous, row.TagID)
			return true
		}
		return false
	})
	return previous, nil
}

// testResourceType is the only resource type the tests register: resources "1" to "9" exist in
// organization 1, and nothing exists elsewhere.
const testResourceType = "test_resource"

func newTestResourceTypes() *ResourceTypes {
	types := NewResourceTypes()
	if err := types.Register(testResourceType, func(_ context.Context, organizationID uint, resourceID string) (bool, error) {
		return organizationID == 1 && len(resourceID) == 1 && resourceID >= "1" && resourceID <= "9", nil
	}); err != nil {
		panic(err)
	}
	return types
}
