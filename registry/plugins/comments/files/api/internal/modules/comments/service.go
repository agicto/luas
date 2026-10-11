package comments

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fastagi/fastbuild/api/internal/domain"
	auditstarter "github.com/fastagi/fastbuild/api/internal/modules/audit"
)

// NewComment is a create request: a root comment, or a reply when ParentID is set.
type NewComment struct {
	ResourceType string
	ResourceID   string
	ParentID     *uint
	Body         string
	Mentions     []uint
}

// Service owns comment rules. Callers pass the verified organization context; resource access
// comes from the registered ResourceChecker and authorship from the stored comment.
type Service interface {
	List(ctx context.Context, actor domain.OrganizationContext, resourceType, resourceID string, after *Cursor, limit int) ([]Thread, *Cursor, error)
	ListReplies(ctx context.Context, actor domain.OrganizationContext, id uint, after *Cursor, limit int) ([]*Comment, *Cursor, error)
	Create(ctx context.Context, actor domain.OrganizationContext, input NewComment) (*Comment, error)
	Update(ctx context.Context, actor domain.OrganizationContext, id uint, expectedVersion uint64, changes CommentChanges) (*Comment, error)
	Delete(ctx context.Context, actor domain.OrganizationContext, id uint) error
}

// memberDirectory verifies that mentioned users belong to the organization. The organization
// starter's repository satisfies it.
type memberDirectory interface {
	FindForUser(ctx context.Context, organizationID, userID uint) (*domain.OrganizationMembership, error)
}

// mentionPublisher delivers mention notifications. It is nil when the notification starter is not
// selected, and mentions are then stored without notifications.
type mentionPublisher interface {
	Publish(ctx context.Context, publication domain.NotificationPublication) (*domain.Notification, error)
}

type service struct {
	store         store
	catalog       *ResourceCatalog
	members       memberDirectory
	notifications mentionPublisher
}

// NewService returns the comments service. notifications may be nil.
func NewService(store store, catalog *ResourceCatalog, members memberDirectory, notifications mentionPublisher) *service {
	return &service{store: store, catalog: catalog, members: members, notifications: notifications}
}

func (s *service) List(
	ctx context.Context,
	actor domain.OrganizationContext,
	resourceType, resourceID string,
	after *Cursor,
	limit int,
) ([]Thread, *Cursor, error) {
	if limit < 1 || limit > maxPageSize {
		return nil, nil, domain.ErrInvalidInput
	}
	if _, _, err := s.resourceAccess(ctx, actor, resourceType, resourceID); err != nil {
		return nil, nil, err
	}
	roots, next, err := s.store.ListRoots(ctx, actor.OrganizationID, resourceType, resourceID, after, limit)
	if err != nil {
		return nil, nil, err
	}
	rootIDs := make([]uint, len(roots))
	for index, root := range roots {
		rootIDs[index] = root.ID
	}
	// One extra reply per root tells whether more exist without counting them.
	previews, err := s.store.ListReplyPreviews(ctx, actor.OrganizationID, rootIDs, replyPreviewLimit+1)
	if err != nil {
		return nil, nil, err
	}
	threads := make([]Thread, len(roots))
	for index, root := range roots {
		replies := previews[root.ID]
		more := len(replies) > replyPreviewLimit
		if more {
			replies = replies[:replyPreviewLimit]
		}
		if replies == nil {
			replies = []*Comment{}
		}
		threads[index] = Thread{Root: root, Replies: replies, HasMoreReplies: more}
	}
	return threads, next, nil
}

func (s *service) ListReplies(
	ctx context.Context,
	actor domain.OrganizationContext,
	id uint,
	after *Cursor,
	limit int,
) ([]*Comment, *Cursor, error) {
	if limit < 1 || limit > maxPageSize {
		return nil, nil, domain.ErrInvalidInput
	}
	if _, _, _, err := s.readableComment(ctx, actor, id); err != nil {
		return nil, nil, err
	}
	return s.store.ListReplies(ctx, actor.OrganizationID, id, after, limit)
}

func (s *service) Create(ctx context.Context, actor domain.OrganizationContext, input NewComment) (*Comment, error) {
	body, ok := normalizeBody(input.Body)
	if !ok {
		return nil, ErrInvalidComment
	}
	resourceType, access, err := s.resourceAccess(ctx, actor, input.ResourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}
	if !access.Comment {
		return nil, domain.ErrPermissionDenied
	}
	mentions, err := s.validMentions(ctx, actor.OrganizationID, input.Mentions)
	if err != nil {
		return nil, err
	}
	comment := &Comment{
		OrganizationID: actor.OrganizationID, ResourceType: input.ResourceType, ResourceID: input.ResourceID,
		AuthorID: &actor.UserID, Body: body, Mentions: mentions,
	}
	if input.ParentID != nil {
		rootID, err := s.threadRoot(ctx, actor.OrganizationID, *input.ParentID, input.ResourceType, input.ResourceID)
		if err != nil {
			return nil, err
		}
		comment.ParentID = &rootID
	}
	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}
	changes := map[string]domain.AuditValueChange{
		"resource_type": {After: comment.ResourceType},
		"resource_id":   {After: comment.ResourceID},
		"mentions":      {After: len(comment.Mentions)},
	}
	if comment.ParentID != nil {
		changes["parent_id"] = domain.AuditValueChange{After: *comment.ParentID}
	}
	recordChange(ctx, "create", comment.ID, changes, nil)
	s.publishMentions(ctx, resourceType, comment, comment.Mentions)
	return comment, nil
}

func (s *service) Update(
	ctx context.Context,
	actor domain.OrganizationContext,
	id uint,
	expectedVersion uint64,
	changes CommentChanges,
) (*Comment, error) {
	if changes.Body == nil && changes.Mentions == nil {
		return nil, ErrInvalidComment
	}
	if changes.Body != nil {
		body, ok := normalizeBody(*changes.Body)
		if !ok {
			return nil, ErrInvalidComment
		}
		changes.Body = &body
	}
	current, resourceType, access, err := s.readableComment(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if current.IsDeleted() {
		return nil, ErrCommentNotFound
	}
	if !isAuthor(current, actor.UserID) || !access.Comment {
		return nil, domain.ErrPermissionDenied
	}
	if changes.Mentions != nil {
		mentions, mentionErr := s.validMentions(ctx, actor.OrganizationID, *changes.Mentions)
		if mentionErr != nil {
			return nil, mentionErr
		}
		changes.Mentions = &mentions
	}
	updated, err := s.store.Update(ctx, actor.OrganizationID, id, expectedVersion, changes)
	if err != nil {
		return nil, err
	}
	// The body is business content: record that it changed, never its text.
	audited := map[string]domain.AuditValueChange{}
	if changes.Body != nil {
		audited["body"] = domain.AuditValueChange{After: "[changed]"}
	}
	if changes.Mentions != nil {
		audited["mentions"] = domain.AuditValueChange{Before: len(current.Mentions), After: len(updated.Mentions)}
	}
	recordChange(ctx, "update", updated.ID, audited, nil)
	var added []uint
	for _, userID := range updated.Mentions {
		if !slices.Contains(current.Mentions, userID) {
			added = append(added, userID)
		}
	}
	s.publishMentions(ctx, resourceType, updated, added)
	return updated, nil
}

func (s *service) Delete(ctx context.Context, actor domain.OrganizationContext, id uint) error {
	current, _, _, err := s.readableComment(ctx, actor, id)
	if err != nil {
		return err
	}
	if current.IsDeleted() {
		return ErrCommentNotFound
	}
	author := isAuthor(current, actor.UserID)
	if !author && !actor.Role.CanManageOrganization() {
		return domain.ErrPermissionDenied
	}
	if _, err := s.store.SoftDelete(ctx, actor.OrganizationID, id); err != nil {
		return err
	}
	var metadata map[string]any
	if !author {
		// A moderator removed someone else's comment: keep who wrote it, never what it said.
		metadata = map[string]any{"moderated": true}
		if current.AuthorID != nil {
			metadata["author_id"] = *current.AuthorID
		}
	}
	recordChange(ctx, "delete", id, map[string]domain.AuditValueChange{
		"body": {Before: "[removed]"},
	}, metadata)
	return nil
}

// resourceAccess resolves a registered type and requires read access to the resource.
func (s *service) resourceAccess(
	ctx context.Context,
	actor domain.OrganizationContext,
	resourceType, resourceID string,
) (ResourceType, ResourceAccess, error) {
	if actor.OrganizationID == 0 || actor.UserID == 0 {
		return ResourceType{}, ResourceAccess{}, domain.ErrOrganizationContextRequired
	}
	registered, ok := s.catalog.Lookup(resourceType)
	if !ok {
		return ResourceType{}, ResourceAccess{}, ErrUnsupportedResourceType
	}
	if !validResourceID(resourceID) {
		return ResourceType{}, ResourceAccess{}, domain.ErrInvalidInput
	}
	access, err := registered.Checker.CheckCommentAccess(ctx, actor, resourceID)
	if err != nil {
		return ResourceType{}, ResourceAccess{}, err
	}
	if !access.Read {
		return ResourceType{}, ResourceAccess{}, ErrResourceNotFound
	}
	return registered, access, nil
}

// readableComment loads a comment and requires read access to its resource; anything the caller
// may not see is not found.
func (s *service) readableComment(
	ctx context.Context,
	actor domain.OrganizationContext,
	id uint,
) (*Comment, ResourceType, ResourceAccess, error) {
	if actor.OrganizationID == 0 || actor.UserID == 0 {
		return nil, ResourceType{}, ResourceAccess{}, domain.ErrOrganizationContextRequired
	}
	if id == 0 {
		return nil, ResourceType{}, ResourceAccess{}, ErrCommentNotFound
	}
	comment, err := s.store.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, ResourceType{}, ResourceAccess{}, err
	}
	resourceType, access, err := s.resourceAccess(ctx, actor, comment.ResourceType, comment.ResourceID)
	if errors.Is(err, ErrResourceNotFound) || errors.Is(err, ErrUnsupportedResourceType) || errors.Is(err, domain.ErrInvalidInput) {
		return nil, ResourceType{}, ResourceAccess{}, ErrCommentNotFound
	}
	if err != nil {
		return nil, ResourceType{}, ResourceAccess{}, err
	}
	return comment, resourceType, access, nil
}

// threadRoot resolves the root a reply attaches to: replying to a reply joins its root's thread.
func (s *service) threadRoot(ctx context.Context, organizationID, parentID uint, resourceType, resourceID string) (uint, error) {
	parent, err := s.store.Get(ctx, organizationID, parentID)
	if errors.Is(err, ErrCommentNotFound) {
		return 0, ErrInvalidParent
	}
	if err != nil {
		return 0, err
	}
	if parent.IsDeleted() || parent.ResourceType != resourceType || parent.ResourceID != resourceID {
		return 0, ErrInvalidParent
	}
	if parent.ParentID != nil {
		return *parent.ParentID, nil
	}
	return parent.ID, nil
}

func (s *service) validMentions(ctx context.Context, organizationID uint, mentions []uint) ([]uint, error) {
	if len(mentions) > maxMentions {
		return nil, ErrInvalidMentions
	}
	unique := make([]uint, 0, len(mentions))
	for _, userID := range mentions {
		if userID == 0 || slices.Contains(unique, userID) {
			return nil, ErrInvalidMentions
		}
		unique = append(unique, userID)
	}
	for _, userID := range unique {
		membership, err := s.members.FindForUser(ctx, organizationID, userID)
		if errors.Is(err, domain.ErrOrganizationNotFound) || (err == nil && membership == nil) {
			return nil, ErrInvalidMentions
		}
		if err != nil {
			return nil, err
		}
	}
	return unique, nil
}

// publishMentions sends one idempotent notification per newly mentioned user. Publication runs
// after the comment is stored and never fails the request: a failure is logged without content.
func (s *service) publishMentions(ctx context.Context, resourceType ResourceType, comment *Comment, userIDs []uint) {
	if s.notifications == nil || len(userIDs) == 0 {
		return
	}
	actionURL := ""
	if resourceType.ActionURL != nil {
		actionURL = resourceType.ActionURL(comment.ResourceID, comment.ID)
	}
	for _, userID := range userIDs {
		if comment.AuthorID != nil && userID == *comment.AuthorID {
			continue
		}
		_, err := s.notifications.Publish(ctx, domain.NotificationPublication{
			UserID:         userID,
			IdempotencyKey: mentionIdempotencyKey(comment.ID, userID),
			Kind:           mentionKind,
			Title:          "You were mentioned in a comment",
			Body:           "You were mentioned in a comment on " + comment.ResourceType + " " + comment.ResourceID + ".",
			ActionURL:      actionURL,
			Channels:       []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelEmail},
		})
		if err != nil {
			slog.WarnContext(ctx, "comments.mention_notification_failed",
				"comment_id", comment.ID, "user_id", userID, "error_class", errorClass(err))
		}
	}
}

func mentionIdempotencyKey(commentID, userID uint) string {
	return "comments.mention:" + strconv.FormatUint(uint64(commentID), 10) + ":" + strconv.FormatUint(uint64(userID), 10)
}

func errorClass(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, domain.ErrServiceUnavailable):
		return "unavailable"
	default:
		return "internal"
	}
}

// normalizeBody trims surrounding whitespace and enforces 1-10,000 characters of UTF-8 text.
func normalizeBody(body string) (string, bool) {
	body = strings.TrimSpace(body)
	length := utf8.RuneCountInString(body)
	return body, length >= 1 && length <= maxBodyLength && utf8.ValidString(body) && !strings.ContainsRune(body, 0)
}

func isAuthor(comment *Comment, userID uint) bool {
	return comment.AuthorID != nil && *comment.AuthorID == userID
}

func recordChange(ctx context.Context, action string, id uint, changes map[string]domain.AuditValueChange, metadata map[string]any) {
	auditstarter.RecordChange(ctx, auditstarter.Change{
		Action:     action,
		Resource:   "comments",
		TargetType: "comment",
		TargetID:   strconv.FormatUint(uint64(id), 10),
		Changes:    changes,
		Metadata:   metadata,
	})
}
