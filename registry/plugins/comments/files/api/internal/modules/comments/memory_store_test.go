package comments

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/fastagi/fastbuild/api/internal/domain"
)

// memoryStore is an in-memory store that enforces the same organization scope, threading, and
// versions as the PostgreSQL repository.
type memoryStore struct {
	mu       sync.Mutex
	comments map[uint]*Comment
	nextID   uint
}

func newMemoryStore() *memoryStore { return &memoryStore{comments: map[uint]*Comment{}} }

func (s *memoryStore) page(match func(*Comment) bool, after *Cursor, limit int) ([]*Comment, *Cursor) {
	var items []*Comment
	for id := uint(1); id <= s.nextID; id++ {
		comment, ok := s.comments[id]
		if !ok || !match(comment) || (after != nil && id <= after.ID) {
			continue
		}
		if len(items) == limit {
			return items, &Cursor{ID: items[len(items)-1].ID}
		}
		copied := *comment
		items = append(items, &copied)
	}
	return items, nil
}

func (s *memoryStore) ListRoots(_ context.Context, organizationID uint, resourceType, resourceID string, after *Cursor, limit int) ([]*Comment, *Cursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, next := s.page(func(c *Comment) bool {
		return c.OrganizationID == organizationID && c.ResourceType == resourceType && c.ResourceID == resourceID && c.ParentID == nil
	}, after, limit)
	return items, next, nil
}

func (s *memoryStore) ListReplyPreviews(_ context.Context, organizationID uint, rootIDs []uint, limit int) (map[uint][]*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previews := map[uint][]*Comment{}
	for _, rootID := range rootIDs {
		previews[rootID], _ = s.page(func(c *Comment) bool {
			return c.OrganizationID == organizationID && c.ParentID != nil && *c.ParentID == rootID
		}, nil, limit)
	}
	return previews, nil
}

func (s *memoryStore) ListReplies(_ context.Context, organizationID, parentID uint, after *Cursor, limit int) ([]*Comment, *Cursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, next := s.page(func(c *Comment) bool {
		return c.OrganizationID == organizationID && c.ParentID != nil && *c.ParentID == parentID
	}, after, limit)
	return items, next, nil
}

func (s *memoryStore) Get(_ context.Context, organizationID, id uint) (*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.get(organizationID, id)
}

func (s *memoryStore) get(organizationID, id uint) (*Comment, error) {
	comment, ok := s.comments[id]
	if !ok || comment.OrganizationID != organizationID {
		return nil, ErrCommentNotFound
	}
	copied := *comment
	copied.Mentions = slices.Clone(comment.Mentions)
	return &copied, nil
}

func (s *memoryStore) Create(_ context.Context, comment *Comment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if comment.ParentID != nil {
		root, ok := s.comments[*comment.ParentID]
		if !ok || root.OrganizationID != comment.OrganizationID || root.ParentID != nil || root.IsDeleted() ||
			root.ResourceType != comment.ResourceType || root.ResourceID != comment.ResourceID {
			return ErrInvalidParent
		}
	}
	s.nextID++
	now := time.Now().UTC()
	comment.ID, comment.Version, comment.CreatedAt, comment.UpdatedAt = s.nextID, 1, now, now
	if comment.Mentions == nil {
		comment.Mentions = []uint{}
	}
	stored := *comment
	stored.Mentions = slices.Clone(comment.Mentions)
	s.comments[comment.ID] = &stored
	return nil
}

func (s *memoryStore) Update(_ context.Context, organizationID, id uint, expected uint64, changes CommentChanges) (*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	comment, err := s.get(organizationID, id)
	if err != nil {
		return nil, err
	}
	if comment.IsDeleted() {
		return nil, ErrCommentNotFound
	}
	if comment.Version != expected {
		return nil, ErrVersionConflict
	}
	now := time.Now().UTC()
	if changes.Body != nil {
		comment.Body = *changes.Body
	}
	if changes.Mentions != nil {
		comment.Mentions = slices.Clone(*changes.Mentions)
	}
	comment.Version++
	comment.UpdatedAt, comment.EditedAt = now, &now
	s.comments[id] = comment
	return s.get(organizationID, id)
}

func (s *memoryStore) SoftDelete(_ context.Context, organizationID, id uint) (*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	comment, err := s.get(organizationID, id)
	if err != nil || comment.IsDeleted() {
		return nil, ErrCommentNotFound
	}
	now := time.Now().UTC()
	comment.Body, comment.Mentions, comment.DeletedAt, comment.UpdatedAt = "", []uint{}, &now, now
	comment.Version++
	s.comments[id] = comment
	return s.get(organizationID, id)
}

// memberSet is a member directory of fixed organization memberships.
type memberSet map[uint][]uint

func (m memberSet) FindForUser(_ context.Context, organizationID, userID uint) (*domain.OrganizationMembership, error) {
	if slices.Contains(m[organizationID], userID) {
		return &domain.OrganizationMembership{OrganizationID: organizationID, UserID: userID, Role: domain.OrganizationRoleMember}, nil
	}
	return nil, domain.ErrOrganizationNotFound
}

// fakePublisher records publications and replays by idempotency key like the notification starter.
type fakePublisher struct {
	mu           sync.Mutex
	publications []domain.NotificationPublication
	keys         map[string]bool
	err          error
}

func (p *fakePublisher) Publish(_ context.Context, publication domain.NotificationPublication) (*domain.Notification, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.keys == nil {
		p.keys = map[string]bool{}
	}
	if !p.keys[publication.IdempotencyKey] {
		p.keys[publication.IdempotencyKey] = true
		p.publications = append(p.publications, publication)
	}
	return &domain.Notification{UserID: publication.UserID, Kind: publication.Kind}, nil
}

// testResourceType is the test-only commentable resource: "doc" IDs exist in organization 1;
// "locked-*" documents are readable but closed to new comments; "hidden-*" documents are not
// visible to members.
func testResourceType() ResourceType {
	return ResourceType{
		Name: "test.doc",
		Checker: ResourceCheckerFunc(func(_ context.Context, organization domain.OrganizationContext, id string) (ResourceAccess, error) {
			switch {
			case organization.OrganizationID != 1:
				return ResourceAccess{}, nil
			case len(id) > 7 && id[:7] == "hidden-":
				return ResourceAccess{Read: organization.Role.CanManageOrganization(), Comment: organization.Role.CanManageOrganization()}, nil
			case len(id) > 7 && id[:7] == "locked-":
				return ResourceAccess{Read: true}, nil
			default:
				return ResourceAccess{Read: true, Comment: true}, nil
			}
		}),
		ActionURL: func(id string, commentID uint) string { return "/console/docs/" + id },
	}
}
