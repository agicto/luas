package commands

import (
	"bytes"
	"sort"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The fragment must describe exactly the routes routesTemplate registers, so the merged contract
// passes the two-way route check without edits.
func TestOpenAPIFragmentDescribesTheGeneratedRoutes(t *testing.T) {
	data := moduleScaffoldData("BlogPost")
	var rendered bytes.Buffer
	require.NoError(t, template.Must(template.New("").Parse(openAPIFragmentTemplate)).Execute(&rendered, data))

	var fragment struct {
		Tags       []map[string]string                  `yaml:"tags"`
		Paths      map[string]map[string]map[string]any `yaml:"paths"`
		Components map[string]map[string]map[string]any `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(rendered.Bytes(), &fragment))

	var operations []string
	for path, item := range fragment.Paths {
		for method, operation := range item {
			operations = append(operations, method+" "+path+" "+operation["operationId"].(string))
		}
	}
	sort.Strings(operations)
	assert.Equal(t, []string{
		"delete /v1/blog_posts/{blog_post_id} deleteBlogPost",
		"get /v1/blog_posts listBlogPosts",
		"get /v1/blog_posts/{blog_post_id} getBlogPost",
		"post /v1/blog_posts createBlogPost",
		"put /v1/blog_posts/{blog_post_id} updateBlogPost",
	}, operations)
	assert.Equal(t, "BlogPost", fragment.Tags[0]["name"])
	assert.Contains(t, fragment.Components["parameters"], "BlogPostID")
	for _, schema := range []string{"BlogPost", "CreateBlogPostRequest", "UpdateBlogPostRequest", "BlogPostResponse", "BlogPostPageResponse"} {
		assert.Contains(t, fragment.Components["schemas"], schema)
	}
}
