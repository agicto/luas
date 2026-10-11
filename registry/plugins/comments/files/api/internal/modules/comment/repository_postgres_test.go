package comment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	infradatabase "github.com/fastagi/fastbuild/api/internal/infra/database"
	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// commentsTable mirrors the plugin migration without the organization and user foreign keys,
// which would need those tables in this isolated schema.
const commentsTable = `CREATE TABLE comments (
    id bigserial PRIMARY KEY,
    organization_id bigint NOT NULL,
    resource_type varchar(40) NOT NULL,
    resource_id varchar(64) NOT NULL,
    parent_id bigint REFERENCES comments (id) ON DELETE CASCADE,
    author_id bigint,
    body text,
    mentions jsonb NOT NULL DEFAULT '[]'::jsonb,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    edited_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT comments_body_check CHECK (
        (deleted_at IS NULL AND body IS NOT NULL AND char_length(body) BETWEEN 1 AND 10000)
        OR (deleted_at IS NOT NULL AND body IS NULL)
    ),
    CONSTRAINT comments_mentions_check CHECK (
        CASE WHEN jsonb_typeof(mentions) = 'array' THEN jsonb_array_length(mentions) <= 20 ELSE false END
    )
)`

func newPostgresRepository(t *testing.T) (*repository, *gorm.DB) {
	t.Helper()
	db := testplatform.OpenPostgres(t, nil)
	require.NoError(t, db.Exec(commentsTable).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_comments_resource ON comments (organization_id, resource_type, resource_id, id)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_comments_parent_id ON comments (parent_id, id) WHERE parent_id IS NOT NULL`).Error)
	return NewRepository(db), db
}

func insert(t *testing.T, repo *repository, organization uint, resourceID string, parent *uint, body string, mentions ...uint) *Comment {
	t.Helper()
	author := uint(1)
	comment := &Comment{
		OrganizationID: organization, ResourceType: "test.doc", ResourceID: resourceID,
		ParentID: parent, AuthorID: &author, Body: body, Mentions: mentions,
	}
	require.NoError(t, repo.Create(context.Background(), comment))
	return comment
}

func bodies(comments []*Comment) []string {
	out := make([]string, len(comments))
	for index, comment := range comments {
		out[index] = comment.Body
	}
	return out
}

func TestRepositoryKeepsOrganizationsAndResourcesApartPostgres(t *testing.T) {
	repo, _ := newPostgresRepository(t)
	ctx := context.Background()
	mine := insert(t, repo, 1, "doc-1", nil, "mine", 7, 8)
	insert(t, repo, 2, "doc-1", nil, "theirs")
	insert(t, repo, 1, "doc-2", nil, "elsewhere")

	roots, next, err := repo.ListRoots(ctx, 1, "test.doc", "doc-1", nil, 10)
	require.NoError(t, err)
	assert.Nil(t, next)
	assert.Equal(t, []string{"mine"}, bodies(roots))
	assert.Equal(t, []uint{7, 8}, roots[0].Mentions)

	_, err = repo.Get(ctx, 2, mine.ID)
	assert.ErrorIs(t, err, ErrCommentNotFound)
	_, err = repo.SoftDelete(ctx, 2, mine.ID)
	assert.ErrorIs(t, err, ErrCommentNotFound)
	body := "hijack"
	_, err = repo.Update(ctx, 2, mine.ID, 1, CommentChanges{Body: &body})
	assert.ErrorIs(t, err, ErrCommentNotFound)

	foreignReply := &Comment{OrganizationID: 2, ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &mine.ID, Body: "x"}
	assert.ErrorIs(t, repo.Create(ctx, foreignReply), ErrInvalidParent, "a reply never crosses organizations")
	previews, err := repo.ListReplyPreviews(ctx, 2, []uint{mine.ID}, 5)
	require.NoError(t, err)
	assert.Empty(t, previews[mine.ID])
}

func TestRepositoryThreadsAndBoundedReplyPreviewsPostgres(t *testing.T) {
	repo, _ := newPostgresRepository(t)
	ctx := context.Background()
	first := insert(t, repo, 1, "doc-1", nil, "first")
	second := insert(t, repo, 1, "doc-1", nil, "second")
	for _, body := range []string{"a", "b", "c", "d"} {
		insert(t, repo, 1, "doc-1", &first.ID, body)
	}
	insert(t, repo, 1, "doc-1", &second.ID, "only")

	wrongResource := &Comment{OrganizationID: 1, ResourceType: "test.doc", ResourceID: "doc-2", ParentID: &first.ID, Body: "x"}
	assert.ErrorIs(t, repo.Create(ctx, wrongResource), ErrInvalidParent)
	reply := insert(t, repo, 1, "doc-1", &second.ID, "reply")
	nested := &Comment{OrganizationID: 1, ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &reply.ID, Body: "x"}
	assert.ErrorIs(t, repo.Create(ctx, nested), ErrInvalidParent, "the store only attaches replies to roots")

	roots, _, err := repo.ListRoots(ctx, 1, "test.doc", "doc-1", nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, bodies(roots), "replies are not roots")

	previews, err := repo.ListReplyPreviews(ctx, 1, []uint{first.ID, second.ID}, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, bodies(previews[first.ID]))
	assert.Equal(t, []string{"only", "reply"}, bodies(previews[second.ID]))
}

func TestRepositoryKeysetPagesOldestFirstPostgres(t *testing.T) {
	repo, _ := newPostgresRepository(t)
	ctx := context.Background()
	root := insert(t, repo, 1, "doc-1", nil, "root")
	for _, body := range []string{"r1", "r2", "r3", "r4", "r5"} {
		insert(t, repo, 1, "doc-1", &root.ID, body)
	}
	var seen []string
	var after *Cursor
	for pages := 0; pages < 5; pages++ {
		items, next, err := repo.ListReplies(ctx, 1, root.ID, after, 2)
		require.NoError(t, err)
		seen = append(seen, bodies(items)...)
		if next == nil {
			break
		}
		after = next
	}
	assert.Equal(t, []string{"r1", "r2", "r3", "r4", "r5"}, seen)
}

func TestRepositorySoftDeleteErasesContentAndKeepsTheThreadPostgres(t *testing.T) {
	repo, db := newPostgresRepository(t)
	ctx := context.Background()
	root := insert(t, repo, 1, "doc-1", nil, "private words", 8)
	insert(t, repo, 1, "doc-1", &root.ID, "reply")

	deleted, err := repo.SoftDelete(ctx, 1, root.ID)
	require.NoError(t, err)
	assert.True(t, deleted.IsDeleted())
	assert.Empty(t, deleted.Body)
	assert.Empty(t, deleted.Mentions)
	assert.Equal(t, uint64(2), deleted.Version)

	var stored struct {
		Body     *string
		Mentions string
	}
	require.NoError(t, db.Raw(`SELECT body, mentions::text AS mentions FROM comments WHERE id = ?`, root.ID).Scan(&stored).Error)
	assert.Nil(t, stored.Body, "the body is erased, not hidden")
	assert.Equal(t, "[]", stored.Mentions)

	_, err = repo.SoftDelete(ctx, 1, root.ID)
	assert.ErrorIs(t, err, ErrCommentNotFound)
	body := "edit"
	_, err = repo.Update(ctx, 1, root.ID, 2, CommentChanges{Body: &body})
	assert.ErrorIs(t, err, ErrCommentNotFound)
	late := &Comment{OrganizationID: 1, ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &root.ID, Body: "late"}
	assert.ErrorIs(t, repo.Create(ctx, late), ErrInvalidParent)

	previews, err := repo.ListReplyPreviews(ctx, 1, []uint{root.ID}, 3)
	require.NoError(t, err)
	assert.Equal(t, []string{"reply"}, bodies(previews[root.ID]), "replies survive their root's deletion")
}

func TestRepositoryConcurrentEditsLetExactlyOneWinPostgres(t *testing.T) {
	repo, _ := newPostgresRepository(t)
	comment := insert(t, repo, 1, "doc-1", nil, "v1")
	const writers = 8
	var wg sync.WaitGroup
	results := make(chan error, writers)
	for index := 0; index < writers; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := "edit"
			_, err := repo.Update(context.Background(), 1, comment.ID, 1, CommentChanges{Body: &body})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			wins++
		case assert.ErrorIs(t, err, ErrVersionConflict):
			conflicts++
		}
	}
	assert.Equal(t, 1, wins)
	assert.Equal(t, writers-1, conflicts)
}

func TestRepositoryReplyWaitsForAConcurrentRootDeletePostgres(t *testing.T) {
	repo, db := newPostgresRepository(t)
	root := insert(t, repo, 1, "doc-1", nil, "root")

	tx := db.Begin()
	require.NoError(t, tx.Error)
	deleting := infradatabase.ContextWithTransaction(context.Background(), tx)
	_, err := repo.SoftDelete(deleting, 1, root.ID)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		reply := &Comment{OrganizationID: 1, ResourceType: "test.doc", ResourceID: "doc-1", ParentID: &root.ID, Body: "racing"}
		done <- repo.Create(context.Background(), reply)
	}()
	select {
	case err := <-done:
		t.Fatalf("reply finished while the delete held the root: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	require.NoError(t, tx.Commit().Error)
	assert.ErrorIs(t, <-done, ErrInvalidParent, "the reply sees the committed delete")
}
