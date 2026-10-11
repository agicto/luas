package tag

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// tagTables mirror the plugin migration without the organization and user foreign keys, which
// would need those tables in this isolated schema.
var tagTables = []string{
	`CREATE TABLE tags (
	    id bigserial PRIMARY KEY,
	    organization_id bigint NOT NULL,
	    name varchar(50) NOT NULL,
	    color varchar(16) NOT NULL DEFAULT 'gray',
	    description varchar(200) NOT NULL DEFAULT '',
	    created_by bigint,
	    version bigint NOT NULL DEFAULT 1,
	    created_at timestamptz NOT NULL,
	    updated_at timestamptz NOT NULL,
	    CONSTRAINT tags_organization_id_id_key UNIQUE (organization_id, id),
	    CONSTRAINT tags_name_check CHECK (char_length(name) BETWEEN 1 AND 50 AND name = btrim(name)),
	    CONSTRAINT tags_color_check CHECK (color IN ('gray', 'red', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink'))
	)`,
	`CREATE UNIQUE INDEX idx_tags_organization_name ON tags (organization_id, (lower(name) COLLATE "C"))`,
	`CREATE TABLE tag_assignments (
	    id bigserial PRIMARY KEY,
	    organization_id bigint NOT NULL,
	    tag_id bigint NOT NULL,
	    resource_type varchar(50) NOT NULL,
	    resource_id varchar(64) NOT NULL,
	    created_at timestamptz NOT NULL,
	    CONSTRAINT tag_assignments_tag_fkey FOREIGN KEY (organization_id, tag_id)
	        REFERENCES tags (organization_id, id) ON DELETE CASCADE
	)`,
	`CREATE UNIQUE INDEX idx_tag_assignments_resource_tag ON tag_assignments (organization_id, resource_type, resource_id, tag_id)`,
	`CREATE INDEX idx_tag_assignments_tag ON tag_assignments (organization_id, tag_id, id)`,
}

func newPostgresRepository(t *testing.T) *repository {
	t.Helper()
	db := testplatform.OpenPostgres(t, nil)
	for _, statement := range tagTables {
		require.NoError(t, db.Exec(statement).Error)
	}
	return NewRepository(db)
}

func createTag(t *testing.T, repo *repository, organization uint, name string) *Tag {
	t.Helper()
	actor := uint(1)
	tag := &Tag{OrganizationID: organization, Name: name, Color: "gray", CreatedBy: &actor}
	require.NoError(t, repo.Create(context.Background(), tag))
	return tag
}

func names(tags []*Tag) []string {
	out := make([]string, len(tags))
	for index, tag := range tags {
		out[index] = tag.Name
	}
	return out
}

func TestRepositoryNamesAreUniquePerOrganizationIgnoringCasePostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	bug := createTag(t, repo, 1, "Bug")
	assert.Equal(t, uint64(1), bug.Version)

	assert.ErrorIs(t, repo.Create(ctx, &Tag{OrganizationID: 1, Name: "BUG", Color: "red"}), ErrNameTaken)
	createTag(t, repo, 2, "bug")

	feature := createTag(t, repo, 1, "Feature")
	name := "bug"
	_, err := repo.Update(ctx, 1, feature.ID, 1, TagChanges{Name: &name})
	assert.ErrorIs(t, err, ErrNameTaken)

	recased := "BUG"
	updated, err := repo.Update(ctx, 1, bug.ID, 1, TagChanges{Name: &recased})
	require.NoError(t, err, "a tag may change the case of its own name")
	assert.Equal(t, "BUG", updated.Name)
	assert.Equal(t, uint64(2), updated.Version)
}

func TestRepositoryKeepsOrganizationsApartPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	mine := createTag(t, repo, 1, "mine")
	theirs := createTag(t, repo, 2, "theirs")
	ref := ResourceRef{Type: testResourceType, ID: "1"}

	_, err := repo.Get(ctx, 2, mine.ID)
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = repo.Update(ctx, 2, mine.ID, 1, TagChanges{Name: &theirs.Name})
	assert.ErrorIs(t, err, ErrNotFound, "another organization sees not found, not a conflict")
	assert.ErrorIs(t, repo.Delete(ctx, 2, mine.ID, 1), ErrNotFound)
	_, _, err = repo.TaggedResources(ctx, 2, mine.ID, "", nil, 10)
	assert.ErrorIs(t, err, ErrNotFound)

	_, _, err = repo.ReplaceResourceTags(ctx, 1, ref, []uint{mine.ID, theirs.ID})
	assert.ErrorIs(t, err, ErrInvalidAssignment, "another organization's tag cannot be attached")
	_, _, err = repo.ReplaceResourceTags(ctx, 1, ref, []uint{mine.ID})
	require.NoError(t, err)
	_, _, err = repo.ReplaceResourceTags(ctx, 2, ref, []uint{theirs.ID})
	require.NoError(t, err)

	tags, err := repo.ResourceTags(ctx, 1, ref)
	require.NoError(t, err)
	assert.Equal(t, []string{"mine"}, names(tags), "the same resource ID in two organizations has separate tag sets")

	err = repo.db.Exec(`INSERT INTO tag_assignments (organization_id, tag_id, resource_type, resource_id, created_at)
		VALUES (1, ?, 'test_resource', '2', now())`, theirs.ID).Error
	assert.Error(t, err, "the composite foreign key rejects a cross-organization assignment")
}

func TestRepositoryListsTagsByNameInKeysetPagesPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	for _, name := range []string{"delta", "Alpha", "charlie", "Bravo", "echo", "Ärger"} {
		createTag(t, repo, 1, name)
	}
	createTag(t, repo, 2, "aaa-other")

	var seen []string
	var after *TagCursor
	for pages := 0; pages < 10; pages++ {
		tags, next, err := repo.List(ctx, 1, after, 2)
		require.NoError(t, err)
		seen = append(seen, names(tags)...)
		if next == nil {
			break
		}
		after = next
	}
	assert.Equal(t, []string{"Alpha", "Bravo", "charlie", "delta", "echo", "Ärger"}, seen,
		"lowercase byte order, identical on every locale")
}

func TestRepositoryReplacesAndPagesAssignmentsPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	a, b, c := createTag(t, repo, 1, "a"), createTag(t, repo, 1, "b"), createTag(t, repo, 1, "c")
	ref := ResourceRef{Type: testResourceType, ID: "7"}

	previous, tags, err := repo.ReplaceResourceTags(ctx, 1, ref, []uint{b.ID, a.ID})
	require.NoError(t, err)
	assert.Empty(t, previous)
	assert.Equal(t, []string{"a", "b"}, names(tags))

	previous, tags, err = repo.ReplaceResourceTags(ctx, 1, ref, []uint{c.ID, b.ID})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{a.ID, b.ID}, previous)
	assert.Equal(t, []string{"b", "c"}, names(tags))

	for _, id := range []string{"1", "2", "3", "4"} {
		_, _, err = repo.ReplaceResourceTags(ctx, 1, ResourceRef{Type: testResourceType, ID: id}, []uint{b.ID})
		require.NoError(t, err)
	}
	_, _, err = repo.ReplaceResourceTags(ctx, 1, ResourceRef{Type: "other_type", ID: "1"}, []uint{b.ID})
	require.NoError(t, err)

	var seen []string
	var after *AssignmentCursor
	for pages := 0; pages < 10; pages++ {
		page, next, pageErr := repo.TaggedResources(ctx, 1, b.ID, testResourceType, after, 2)
		require.NoError(t, pageErr)
		for _, resource := range page {
			seen = append(seen, resource.ResourceID)
		}
		if next == nil {
			break
		}
		after = next
	}
	assert.Equal(t, []string{"4", "3", "2", "1", "7"}, seen, "newest assignment first, filtered by type")

	require.NoError(t, repo.Delete(ctx, 1, b.ID, 1))
	tags, err = repo.ResourceTags(ctx, 1, ref)
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, names(tags), "deleting a tag removes its assignments")

	detached, err := repo.DetachResource(ctx, 1, ref)
	require.NoError(t, err)
	assert.Equal(t, []uint{c.ID}, detached)
	tags, err = repo.ResourceTags(ctx, 1, ref)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestRepositoryConcurrentWritersPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()

	t.Run("one of many equal names wins", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, 8)
		for index := range results {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[index] = repo.Create(ctx, &Tag{OrganizationID: 1, Name: []string{"Race", "RACE"}[index%2], Color: "gray"})
			}()
		}
		wg.Wait()
		succeeded := slices.IndexFunc(results, func(err error) bool { return err == nil })
		require.GreaterOrEqual(t, succeeded, 0)
		for index, err := range results {
			if index != succeeded {
				assert.ErrorIs(t, err, ErrNameTaken)
			}
		}
	})

	t.Run("one of many updates of a version wins", func(t *testing.T) {
		tag := createTag(t, repo, 1, "contended")
		var wg sync.WaitGroup
		results := make([]error, 8)
		for index := range results {
			wg.Add(1)
			go func() {
				defer wg.Done()
				color := Colors[index]
				_, results[index] = repo.Update(ctx, 1, tag.ID, 1, TagChanges{Color: &color})
			}()
		}
		wg.Wait()
		var won int
		for _, err := range results {
			if err == nil {
				won++
				continue
			}
			assert.ErrorIs(t, err, ErrVersionConflict)
		}
		assert.Equal(t, 1, won)
	})

	t.Run("concurrent replacements end with one writer's set", func(t *testing.T) {
		var ids []uint
		for index := 0; index < 6; index++ {
			ids = append(ids, createTag(t, repo, 1, fmt.Sprintf("set-%d", index)).ID)
		}
		ref := ResourceRef{Type: testResourceType, ID: "9"}
		sets := [][]uint{{ids[0], ids[1]}, {ids[2], ids[3]}, {ids[4], ids[5]}, {ids[0], ids[5]}}
		for round := 0; round < 5; round++ {
			var wg sync.WaitGroup
			for _, set := range sets {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, _, err := repo.ReplaceResourceTags(ctx, 1, ref, set)
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
			tags, err := repo.ResourceTags(ctx, 1, ref)
			require.NoError(t, err)
			current := make([]uint, len(tags))
			for index, tag := range tags {
				current[index] = tag.ID
			}
			slices.Sort(current)
			assert.True(t, slices.ContainsFunc(sets, func(set []uint) bool {
				sorted := slices.Clone(set)
				slices.Sort(sorted)
				return slices.Equal(sorted, current)
			}), "round %d ended with %v, not a union of writers", round, current)
		}
	})

	t.Run("a tag deleted during a replacement is never half attached", func(t *testing.T) {
		victim := createTag(t, repo, 1, "victim")
		ref := ResourceRef{Type: testResourceType, ID: "8"}
		var wg sync.WaitGroup
		var replaceErr, deleteErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, replaceErr = repo.ReplaceResourceTags(ctx, 1, ref, []uint{victim.ID})
		}()
		go func() {
			defer wg.Done()
			deleteErr = repo.Delete(ctx, 1, victim.ID, 1)
		}()
		wg.Wait()
		require.NoError(t, deleteErr)
		if replaceErr != nil {
			assert.True(t, errors.Is(replaceErr, ErrInvalidAssignment), "replace lost the race cleanly: %v", replaceErr)
		}
		tags, err := repo.ResourceTags(ctx, 1, ref)
		require.NoError(t, err)
		assert.Empty(t, tags)
	})
}
