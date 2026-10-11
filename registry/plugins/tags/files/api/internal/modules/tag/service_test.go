package tag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fastagi/fastbuild/api/internal/domain"
	auditstarter "github.com/fastagi/fastbuild/api/internal/modules/audit"
)

func newRecordingService() (*service, *memoryStore, *[]auditstarter.Change) {
	store := newMemoryStore()
	svc := NewService(store, newTestResourceTypes())
	var changes []auditstarter.Change
	svc.record = func(_ context.Context, change auditstarter.Change) { changes = append(changes, change) }
	return svc, store, &changes
}

func TestResourceTypesRegisterOnceWithValidNames(t *testing.T) {
	types := NewResourceTypes()
	exists := func(context.Context, uint, string) (bool, error) { return true, nil }
	require.NoError(t, types.Register("knowledge_article", exists))
	assert.Error(t, types.Register("knowledge_article", exists), "a type registers once")
	assert.Error(t, types.Register("Knowledge", exists), "names are lowercase snake_case")
	assert.Error(t, types.Register("other", nil), "a checker is required")
	require.NoError(t, types.Register("asset", exists))
	assert.Equal(t, []string{"asset", "knowledge_article"}, types.Names())
	assert.True(t, types.Registered("asset"))

	ctx := context.Background()
	assert.NoError(t, types.Exists(ctx, 1, ResourceRef{Type: "asset", ID: "42"}))
	assert.ErrorIs(t, types.Exists(ctx, 1, ResourceRef{Type: "unknown", ID: "42"}), ErrResourceNotFound)
	assert.ErrorIs(t, types.Exists(ctx, 1, ResourceRef{Type: "asset", ID: "../42"}), ErrResourceNotFound)

	failure := errors.New("database down")
	require.NoError(t, types.Register("broken", func(context.Context, uint, string) (bool, error) { return false, failure }))
	assert.ErrorIs(t, types.Exists(ctx, 1, ResourceRef{Type: "broken", ID: "1"}), failure, "checker errors are not hidden as not found")
}

func TestServiceAuditsTagChangesWithoutDescriptions(t *testing.T) {
	svc, _, changes := newRecordingService()
	ctx := context.Background()
	tag, err := svc.Create(ctx, 1, 7, TagInput{Name: " Customer ", Color: "green", Description: "Private notes about the account"})
	require.NoError(t, err)
	assert.Equal(t, "Customer", tag.Name)

	description := "Even more private notes"
	_, err = svc.Update(ctx, 1, tag.ID, 1, TagChanges{Description: &description})
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, 1, tag.ID, 2))

	require.Len(t, *changes, 3)
	created, updated, deleted := (*changes)[0], (*changes)[1], (*changes)[2]
	assert.Equal(t, "create", created.Action)
	assert.Equal(t, "tags", created.Resource)
	assert.Equal(t, "1", created.TargetID)
	assert.Equal(t, "Customer", created.Changes["name"].After)
	assert.Equal(t, "[changed]", created.Changes["description"].After)
	assert.Equal(t, map[string]domain.AuditValueChange{"description": {After: "[changed]"}}, updated.Changes)
	assert.Equal(t, "delete", deleted.Action)
	assert.Nil(t, deleted.Changes)
}

func TestServiceRejectsInvalidTags(t *testing.T) {
	svc, _, changes := newRecordingService()
	ctx := context.Background()
	for _, input := range []TagInput{
		{Name: ""},
		{Name: "   "},
		{Name: strings.Repeat("x", 51)},
		{Name: "tab\tinside"},
		{Name: "ok", Color: "#ff0000"},
		{Name: "ok", Description: strings.Repeat("x", 201)},
		{Name: "ok", Description: "nul\x00"},
		{Name: "\xff"},
	} {
		_, err := svc.Create(ctx, 1, 7, input)
		assert.ErrorIs(t, err, ErrInvalid, "%q", input.Name)
	}
	_, err := svc.Create(ctx, 0, 7, TagInput{Name: "ok"})
	assert.ErrorIs(t, err, ErrInvalid, "an organization is required")
	_, err = svc.Update(ctx, 1, 1, 1, TagChanges{})
	assert.ErrorIs(t, err, ErrInvalid, "an update needs a field")
	assert.Empty(t, *changes, "rejected writes are not audited")

	name := "五十个字以内的中文标签"
	created, err := svc.Create(ctx, 1, 7, TagInput{Name: name})
	require.NoError(t, err, "names count characters, not bytes")
	assert.Equal(t, name, created.Name)
}

func TestServiceAuditsOnlyEffectiveAssignmentChanges(t *testing.T) {
	svc, store, changes := newRecordingService()
	ctx := context.Background()
	first, err := svc.Create(ctx, 1, 7, TagInput{Name: "a"})
	require.NoError(t, err)
	second, err := svc.Create(ctx, 1, 7, TagInput{Name: "b"})
	require.NoError(t, err)
	*changes = nil

	ref := ResourceRef{Type: testResourceType, ID: "4"}
	tags, err := svc.ReplaceResourceTags(ctx, 1, ref, []uint{second.ID, first.ID})
	require.NoError(t, err)
	assert.Len(t, tags, 2)
	_, err = svc.ReplaceResourceTags(ctx, 1, ref, []uint{first.ID, second.ID})
	require.NoError(t, err)
	require.Len(t, *changes, 1, "an unchanged set records nothing")
	assignment := (*changes)[0]
	assert.Equal(t, "tag_assignments", assignment.Resource)
	assert.Equal(t, testResourceType, assignment.TargetType)
	assert.Equal(t, "4", assignment.TargetID)
	assert.Equal(t, domain.AuditValueChange{Before: []uint{}, After: []uint{first.ID, second.ID}}, assignment.Changes["tag_ids"])

	require.NoError(t, svc.DetachResource(ctx, 1, ref))
	require.Len(t, *changes, 2)
	assert.Equal(t, domain.AuditValueChange{Before: []uint{first.ID, second.ID}, After: []uint{}}, (*changes)[1].Changes["tag_ids"])
	assert.Empty(t, store.assignments)
	require.NoError(t, svc.DetachResource(ctx, 1, ref), "detaching an untagged resource is a no-op")
	assert.Len(t, *changes, 2)
	assert.ErrorIs(t, svc.DetachResource(ctx, 1, ResourceRef{Type: "Bad", ID: "1"}), domain.ErrInvalidInput)
}

func TestServiceChecksResourcesBeforeTouchingAssignments(t *testing.T) {
	svc, store, _ := newRecordingService()
	ctx := context.Background()
	tag, err := svc.Create(ctx, 1, 7, TagInput{Name: "a"})
	require.NoError(t, err)

	_, err = svc.ReplaceResourceTags(ctx, 1, ResourceRef{Type: "unregistered", ID: "1"}, []uint{tag.ID})
	assert.ErrorIs(t, err, ErrResourceNotFound)
	_, err = svc.ReplaceResourceTags(ctx, 2, ResourceRef{Type: testResourceType, ID: "1"}, nil)
	assert.ErrorIs(t, err, ErrResourceNotFound, "the checker scopes resources to the organization")
	_, err = svc.ResourceTags(ctx, 2, ResourceRef{Type: testResourceType, ID: "1"})
	assert.ErrorIs(t, err, ErrResourceNotFound)
	_, err = svc.ReplaceResourceTags(ctx, 1, ResourceRef{Type: testResourceType, ID: "1"}, []uint{tag.ID, 0})
	assert.ErrorIs(t, err, ErrInvalidAssignment)
	assert.Empty(t, store.assignments)

	_, _, err = svc.TaggedResources(ctx, 1, tag.ID, "", nil, 0)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	_, _, err = svc.TaggedResources(ctx, 2, tag.ID, "", nil, 10)
	assert.ErrorIs(t, err, ErrNotFound)
}
