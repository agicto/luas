package tag

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"sync"
)

var (
	resourceTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)
	resourceIDPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
)

// ResourceChecker reports whether a resource exists in an organization. It must answer false, not
// an error, for a resource of another organization, so tagging never discloses foreign IDs.
type ResourceChecker func(ctx context.Context, organizationID uint, resourceID string) (bool, error)

// ResourceTypes is the code-owned catalog of resource types that can carry tags. The tag starter
// ships with no types: each module that wants its resources tagged takes *ResourceTypes in its own
// Wire constructor and registers a type there, at assembly time, before the server accepts
// requests. The tag module never imports those modules.
//
//	func NewHandler(service Service, tagTypes *tag.ResourceTypes) (*Handler, error) {
//		return &Handler{service: service}, tagTypes.Register("knowledge_article", service.ArticleExists)
//	}
type ResourceTypes struct {
	mu       sync.RWMutex
	checkers map[string]ResourceChecker
}

// NewResourceTypes returns an empty catalog. Wire provides one per application.
func NewResourceTypes() *ResourceTypes {
	return &ResourceTypes{checkers: map[string]ResourceChecker{}}
}

// Register adds a resource type. Names are lowercase snake_case, at most 50 characters, and
// registered once; they are stored with every assignment, so renaming one needs a data migration.
func (r *ResourceTypes) Register(name string, checker ResourceChecker) error {
	if !resourceTypePattern.MatchString(name) {
		return fmt.Errorf("tag resource type %q must be lowercase snake_case, at most %d characters", name, maxResourceTypeLength)
	}
	if checker == nil {
		return fmt.Errorf("tag resource type %q needs a checker", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.checkers[name]; exists {
		return fmt.Errorf("tag resource type %q is already registered", name)
	}
	r.checkers[name] = checker
	return nil
}

// Names returns the registered types in order.
func (r *ResourceTypes) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.checkers))
	for name := range r.checkers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Registered reports whether name is a registered type.
func (r *ResourceTypes) Registered(name string) bool {
	_, ok := r.checker(name)
	return ok
}

// Exists asks the owning module whether the resource is in the organization. Unknown types and
// malformed IDs answer ErrResourceNotFound.
func (r *ResourceTypes) Exists(ctx context.Context, organizationID uint, ref ResourceRef) error {
	checker, ok := r.checker(ref.Type)
	if !ok || !resourceIDPattern.MatchString(ref.ID) {
		return ErrResourceNotFound
	}
	exists, err := checker(ctx, organizationID, ref.ID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrResourceNotFound
	}
	return nil
}

func (r *ResourceTypes) checker(name string) (ResourceChecker, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	checker, ok := r.checkers[name]
	return checker, ok
}

// ValidResourceRef reports whether ref is well formed; it does not check registration.
func ValidResourceRef(ref ResourceRef) bool {
	return resourceTypePattern.MatchString(ref.Type) && resourceIDPattern.MatchString(ref.ID)
}
