package knowledge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testplatform "github.com/fastagi/fastbuild/api/internal/infra/testing"
)

// knowledgeTable mirrors the plugin migration without the foreign keys, which would need the
// organization and user tables in this isolated schema.
const knowledgeTable = `CREATE TABLE knowledge_articles (
    id bigserial PRIMARY KEY,
    organization_id bigint NOT NULL,
    title varchar(200) NOT NULL,
    body text NOT NULL DEFAULT '',
    created_by bigint,
    updated_by bigint,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', title), 'A') || setweight(to_tsvector('simple', body), 'B')
    ) STORED,
    CONSTRAINT knowledge_articles_title_check CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT knowledge_articles_body_check CHECK (char_length(body) <= 200000)
)`

func newPostgresRepository(t *testing.T) *repository {
	t.Helper()
	db := testplatform.OpenPostgres(t, nil)
	require.NoError(t, db.Exec(knowledgeTable).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_knowledge_articles_search ON knowledge_articles USING gin (search_vector)`).Error)
	return NewRepository(db)
}

func create(t *testing.T, repo *repository, organization uint, title, body string) *Article {
	t.Helper()
	actor := uint(1)
	article := &Article{OrganizationID: organization, Title: title, Body: body, CreatedBy: &actor}
	require.NoError(t, repo.Create(context.Background(), article))
	return article
}

func titles(hits []SearchHit) []string {
	out := make([]string, len(hits))
	for index, hit := range hits {
		out[index] = hit.Title
	}
	return out
}

func TestRepositorySearchRanksWordsAndFindsChineseSubstringsPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	create(t, repo, 1, "Deployment guide", "How we deploy the API with Kubernetes.")
	create(t, repo, 1, "Onboarding", "Read the deployment guide on your first day.")
	create(t, repo, 1, "公司知识库使用说明", "在这里记录团队的流程和常见问题。")
	create(t, repo, 2, "Deployment secrets", "Another organization's article.")

	words, err := repo.Search(ctx, 1, "deployment", maxSearchResults)
	require.NoError(t, err)
	assert.Equal(t, []string{"Deployment guide", "Onboarding"}, titles(words), "title matches rank above body matches")

	chinese, err := repo.Search(ctx, 1, "知识库", maxSearchResults)
	require.NoError(t, err)
	assert.Equal(t, []string{"公司知识库使用说明"}, titles(chinese), "substrings match text without word spacing")

	bodyOnly, err := repo.Search(ctx, 1, "常见问题", maxSearchResults)
	require.NoError(t, err)
	assert.Equal(t, []string{"公司知识库使用说明"}, titles(bodyOnly))

	wildcard, err := repo.Search(ctx, 1, "%", maxSearchResults)
	require.NoError(t, err)
	assert.Empty(t, wildcard, "LIKE wildcards are matched literally")

	other, err := repo.Search(ctx, 2, "deployment", maxSearchResults)
	require.NoError(t, err)
	assert.Equal(t, []string{"Deployment secrets"}, titles(other), "search never crosses organizations")
}

func TestRepositoryUpdatesAndDeletesOnlyTheReadVersionPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	article := create(t, repo, 1, "Draft", "v1")
	assert.Equal(t, uint64(1), article.Version)

	title := "Final"
	updated, err := repo.Update(ctx, 1, article.ID, 1, ArticleChanges{Title: &title}, 9)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), updated.Version)
	assert.Equal(t, "Final", updated.Title)
	assert.Equal(t, "v1", updated.Body, "unchanged fields stay")
	require.NotNil(t, updated.UpdatedBy)
	assert.Equal(t, uint(9), *updated.UpdatedBy)

	_, err = repo.Update(ctx, 1, article.ID, 1, ArticleChanges{Title: &title}, 9)
	assert.ErrorIs(t, err, ErrVersionConflict)
	_, err = repo.Update(ctx, 2, article.ID, 2, ArticleChanges{Title: &title}, 9)
	assert.ErrorIs(t, err, ErrArticleNotFound, "another organization sees not found, not a conflict")

	assert.ErrorIs(t, repo.Delete(ctx, 1, article.ID, 1), ErrVersionConflict)
	require.NoError(t, repo.Delete(ctx, 1, article.ID, 2))
	_, err = repo.Get(ctx, 1, article.ID)
	assert.ErrorIs(t, err, ErrArticleNotFound)
}

func TestRepositoryListPagesNewestFirstPostgres(t *testing.T) {
	repo := newPostgresRepository(t)
	ctx := context.Background()
	for _, title := range []string{"a", "b", "c", "d", "e"} {
		create(t, repo, 1, title, "")
	}
	create(t, repo, 2, "other", "")

	var seen []string
	var after *ArticleCursor
	for pages := 0; pages < 5; pages++ {
		items, next, err := repo.List(ctx, 1, after, 2)
		require.NoError(t, err)
		for _, item := range items {
			seen = append(seen, item.Title)
		}
		if next == nil {
			break
		}
		after = next
	}
	assert.Equal(t, []string{"e", "d", "c", "b", "a"}, seen)
}
