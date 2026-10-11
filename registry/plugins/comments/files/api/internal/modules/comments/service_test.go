package comments

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/domain"
	"github.com/fastagi/fastbuild/api/internal/infra/config"
)

func newTestService(t *testing.T, publisher mentionPublisher) (*service, *memoryStore) {
	t.Helper()
	catalog, err := NewResourceCatalog(testResourceType())
	require.NoError(t, err)
	store := newMemoryStore()
	return NewService(store, catalog, memberSet{1: {7, 8, 9}}, publisher), store
}

func actor(user uint) domain.OrganizationContext {
	return domain.OrganizationContext{OrganizationID: 1, UserID: user, Role: domain.OrganizationRoleMember}
}

func TestMentionsPublishOneIdempotentNotificationPerUser(t *testing.T) {
	publisher := &fakePublisher{}
	svc, _ := newTestService(t, publisher)
	ctx := context.Background()

	comment, err := svc.Create(ctx, actor(7), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", Body: "secret plan", Mentions: []uint{8, 7, 9}})
	require.NoError(t, err)
	require.Len(t, publisher.publications, 2, "the author is never notified of their own mention")
	first := publisher.publications[0]
	assert.Equal(t, uint(8), first.UserID)
	assert.Equal(t, "comments.mention:1:8", first.IdempotencyKey)
	assert.Equal(t, mentionKind, first.Kind)
	assert.Equal(t, "/console/docs/doc-1", first.ActionURL)
	assert.NotContains(t, first.Body, "secret plan", "notifications never copy the comment body")
	assert.Equal(t, []uint{8, 7, 9}, comment.Mentions)

	// Editing publishes only to newly mentioned users; a repeated key is replayed, not duplicated.
	mentions := []uint{9, 8}
	_, err = svc.Update(ctx, actor(7), comment.ID, comment.Version, CommentChanges{Mentions: &mentions})
	require.NoError(t, err)
	assert.Len(t, publisher.publications, 2)

	only7 := []uint{7}
	updated, err := svc.Update(ctx, actor(7), comment.ID, comment.Version+1, CommentChanges{Mentions: &only7})
	require.NoError(t, err)
	readd := []uint{8}
	_, err = svc.Update(ctx, actor(7), comment.ID, updated.Version, CommentChanges{Mentions: &readd})
	require.NoError(t, err)
	assert.Len(t, publisher.publications, 2, "re-mentioning a user reuses the same idempotency key")
}

func TestMentionsAreStoredSilentlyWithoutTheNotificationStarter(t *testing.T) {
	svc, store := newTestService(t, nil)
	comment, err := svc.Create(context.Background(), actor(7), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", Body: "hi", Mentions: []uint{8}})
	require.NoError(t, err)
	assert.Equal(t, []uint{8}, store.comments[comment.ID].Mentions)
}

func TestMentionPublicationFailuresDoNotFailTheComment(t *testing.T) {
	publisher := &fakePublisher{err: domain.ErrServiceUnavailable}
	svc, store := newTestService(t, publisher)
	comment, err := svc.Create(context.Background(), actor(7), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", Body: "hi", Mentions: []uint{8}})
	require.NoError(t, err)
	assert.Contains(t, store.comments, comment.ID)
}

func TestMentionPublisherFollowsStarterSelection(t *testing.T) {
	publisher := &fakePublisher{}
	disabled := &config.Config{}
	assert.Nil(t, ProvideMentionPublisher(disabled, publisher))
	assert.Nil(t, ProvideMentionPublisher(nil, publisher))

	enabled := &config.Config{}
	enabled.Starters.Optional = []string{config.StarterOrganization, config.StarterNotification}
	assert.NotNil(t, ProvideMentionPublisher(enabled, publisher))
}

func TestResourceCatalogRejectsInvalidTypes(t *testing.T) {
	checker := testResourceType().Checker
	_, err := NewResourceCatalog(ResourceType{Name: "Task", Checker: checker})
	assert.Error(t, err)
	_, err = NewResourceCatalog(ResourceType{Name: "task"})
	assert.Error(t, err, "a checker is required")
	_, err = NewResourceCatalog(ResourceType{Name: "task", Checker: checker}, ResourceType{Name: "task", Checker: checker})
	assert.Error(t, err)
	empty, err := NewResourceCatalog()
	require.NoError(t, err)
	_, ok := empty.Lookup("task")
	assert.False(t, ok)
}

func TestReplyingToAReplyJoinsTheRootThread(t *testing.T) {
	svc, _ := newTestService(t, nil)
	ctx := context.Background()
	root, err := svc.Create(ctx, actor(7), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", Body: "root"})
	require.NoError(t, err)
	reply, err := svc.Create(ctx, actor(8), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &root.ID, Body: "reply"})
	require.NoError(t, err)
	nested, err := svc.Create(ctx, actor(9), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &reply.ID, Body: "nested"})
	require.NoError(t, err)
	require.NotNil(t, nested.ParentID)
	assert.Equal(t, root.ID, *nested.ParentID)

	require.NoError(t, svc.Delete(ctx, actor(8), reply.ID))
	_, err = svc.Create(ctx, actor(9), NewComment{ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &reply.ID, Body: "late"})
	assert.ErrorIs(t, err, ErrInvalidParent, "a deleted comment takes no replies")
}
