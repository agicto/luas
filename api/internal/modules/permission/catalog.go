package permission

import (
	"fmt"
	"slices"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	PermissionRolesRead         domain.PermissionKey = "permission.roles.read"
	PermissionRolesManage       domain.PermissionKey = "permission.roles.manage"
	PermissionAssignmentsRead   domain.PermissionKey = "permission.assignments.read"
	PermissionAssignmentsManage domain.PermissionKey = "permission.assignments.manage"
)

// Catalog is the immutable set of permission keys owned by reviewed policy checks.
type Catalog struct {
	keys   []domain.PermissionKey
	keySet map[domain.PermissionKey]struct{}
}

// NewCatalog validates and freezes one application permission catalog.
func NewCatalog(keys ...domain.PermissionKey) (*Catalog, error) {
	catalog := &Catalog{
		keys:   make([]domain.PermissionKey, 0, len(keys)),
		keySet: make(map[domain.PermissionKey]struct{}, len(keys)),
	}
	for _, key := range keys {
		if !key.IsValid() {
			return nil, fmt.Errorf("permission key %q must use canonical lowercase dotted segments", key)
		}
		if _, duplicate := catalog.keySet[key]; duplicate {
			return nil, fmt.Errorf("duplicate permission key %q", key)
		}
		catalog.keySet[key] = struct{}{}
		catalog.keys = append(catalog.keys, key)
	}
	slices.Sort(catalog.keys)
	return catalog, nil
}

// NewDefaultCatalog builds the permission starter's built-in management catalog.
func NewDefaultCatalog() (*Catalog, error) {
	return NewCatalog(DefaultPermissionKeys()...)
}

// DefaultPermissionKeys returns a defensive copy of built-in management permissions.
func DefaultPermissionKeys() []domain.PermissionKey {
	return []domain.PermissionKey{
		PermissionRolesRead,
		PermissionRolesManage,
		PermissionAssignmentsRead,
		PermissionAssignmentsManage,
	}
}

// Keys returns registered keys in stable lexical order.
func (c *Catalog) Keys() []domain.PermissionKey {
	if c == nil {
		return nil
	}
	return slices.Clone(c.keys)
}

// Contains reports exact key registration.
func (c *Catalog) Contains(key domain.PermissionKey) bool {
	if c == nil {
		return false
	}
	_, exists := c.keySet[key]
	return exists
}
