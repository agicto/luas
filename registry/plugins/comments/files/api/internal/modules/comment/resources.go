package comment

import (
	"context"
	"fmt"
	"regexp"

	"github.com/fastagi/fastbuild/api/internal/domain"
)

var (
	resourceTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)*$`)
	resourceIDPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
)

const maxResourceTypeLength = 40

// ResourceAccess is what the caller may do with one resource's comments. A zero value means the
// resource does not exist in the organization or the caller may not see it; both answer 404.
type ResourceAccess struct {
	// Read allows listing the resource's comments and deleting the caller's own comments.
	Read bool
	// Comment allows creating comments and replies and editing the caller's own comments.
	Comment bool
}

// ResourceChecker answers whether a resource exists in the verified organization and what the
// caller may do with its comments. It is called on every comment request, so it should be one
// indexed lookup. Return an error only for infrastructure failures.
type ResourceChecker interface {
	CheckCommentAccess(ctx context.Context, organization domain.OrganizationContext, resourceID string) (ResourceAccess, error)
}

// ResourceCheckerFunc adapts a function to ResourceChecker.
type ResourceCheckerFunc func(ctx context.Context, organization domain.OrganizationContext, resourceID string) (ResourceAccess, error)

// CheckCommentAccess calls f.
func (f ResourceCheckerFunc) CheckCommentAccess(
	ctx context.Context,
	organization domain.OrganizationContext,
	resourceID string,
) (ResourceAccess, error) {
	return f(ctx, organization, resourceID)
}

// ResourceType registers one commentable kind of resource, for example "task".
type ResourceType struct {
	// Name is the public resource_type: lowercase dotted words, at most 40 characters.
	Name string
	// Checker verifies existence and access in the organization.
	Checker ResourceChecker
	// ActionURL optionally returns the same-origin path a mention notification links to, for
	// example "/console/tasks/42#comment-7". Return "" for no link.
	ActionURL func(resourceID string, commentID uint) string
}

// ResourceCatalog is the frozen, code-owned set of commentable resource types.
type ResourceCatalog struct {
	types map[string]ResourceType
}

// NewResourceCatalog validates and freezes resource types. An empty catalog is valid: every
// comment request then answers COMMENT.UNSUPPORTED_RESOURCE_TYPE.
func NewResourceCatalog(types ...ResourceType) (*ResourceCatalog, error) {
	if len(types) > maxResourceTypes {
		return nil, fmt.Errorf("comment resource catalog exceeds %d types", maxResourceTypes)
	}
	catalog := &ResourceCatalog{types: make(map[string]ResourceType, len(types))}
	for _, resourceType := range types {
		if !validResourceType(resourceType.Name) || resourceType.Checker == nil {
			return nil, fmt.Errorf("comment resource type %q is invalid", resourceType.Name)
		}
		if _, exists := catalog.types[resourceType.Name]; exists {
			return nil, fmt.Errorf("duplicate comment resource type %q", resourceType.Name)
		}
		catalog.types[resourceType.Name] = resourceType
	}
	return catalog, nil
}

// Lookup returns a registered resource type.
func (c *ResourceCatalog) Lookup(name string) (ResourceType, bool) {
	if c == nil {
		return ResourceType{}, false
	}
	resourceType, ok := c.types[name]
	return resourceType, ok
}

// ProvideResourceCatalog lists the resource types members can comment on. The plugin ships none.
// A module makes its resource commentable by adding a type here and asking Wire for what its
// checker needs, for example:
//
//	func ProvideResourceCatalog(tasks domain.TaskReader) (*ResourceCatalog, error) {
//		return NewResourceCatalog(ResourceType{
//			Name: "task",
//			Checker: ResourceCheckerFunc(func(ctx context.Context, org domain.OrganizationContext, id string) (ResourceAccess, error) {
//				visible, err := tasks.VisibleTo(ctx, org.OrganizationID, org.UserID, id)
//				return ResourceAccess{Read: visible, Comment: visible}, err
//			}),
//			ActionURL: func(id string, commentID uint) string { return "/console/tasks/" + id },
//		})
//	}
//
// See contracts/COMMENT.md.
func ProvideResourceCatalog() (*ResourceCatalog, error) {
	return NewResourceCatalog()
}

func validResourceType(name string) bool {
	return len(name) <= maxResourceTypeLength && resourceTypePattern.MatchString(name)
}

func validResourceID(id string) bool {
	return len(id) <= maxResourceIDLen && resourceIDPattern.MatchString(id)
}
