package starter

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/zgiai/luas/api/internal/starter/assembly"
)

var starterNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// Catalog separates immutable default starters from explicitly selected optional starters.
type Catalog struct {
	defaults      []assembly.StarterManifest
	defaultNames  map[string]struct{}
	optional      map[string]assembly.StarterManifest
	optionalNames []string
}

// NewCatalog creates and validates one deterministic starter catalog.
func NewCatalog(defaults, optional []assembly.StarterManifest) (*Catalog, error) {
	catalog := &Catalog{
		defaults:     make([]assembly.StarterManifest, 0, len(defaults)),
		defaultNames: make(map[string]struct{}, len(defaults)),
		optional:     make(map[string]assembly.StarterManifest, len(optional)),
	}

	seen := make(map[string]struct{}, len(defaults)+len(optional))
	for _, manifest := range defaults {
		name, err := validateCatalogManifest(manifest)
		if err != nil {
			return nil, fmt.Errorf("validate default starter: %w", err)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate starter manifest %q", name)
		}
		seen[name] = struct{}{}
		catalog.defaultNames[name] = struct{}{}
		catalog.defaults = append(catalog.defaults, manifest)
	}

	for _, manifest := range optional {
		name, err := validateCatalogManifest(manifest)
		if err != nil {
			return nil, fmt.Errorf("validate optional starter: %w", err)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate starter manifest %q", name)
		}
		seen[name] = struct{}{}
		catalog.optional[name] = manifest
		catalog.optionalNames = append(catalog.optionalNames, name)
	}
	slices.Sort(catalog.optionalNames)

	return catalog, nil
}

// Select returns all defaults followed by optional starters in configured order.
func (c *Catalog) Select(names []string) ([]assembly.StarterManifest, error) {
	if c == nil {
		return nil, fmt.Errorf("starter catalog is required")
	}

	selected := append([]assembly.StarterManifest(nil), c.defaults...)
	seen := make(map[string]struct{}, len(names))
	for _, rawName := range names {
		name := rawName
		if name == "" || name != strings.TrimSpace(name) || !starterNamePattern.MatchString(name) {
			return nil, fmt.Errorf("optional starter %q must use a canonical lowercase name", rawName)
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("duplicate optional starter %q", name)
		}
		seen[name] = struct{}{}

		if _, isDefault := c.defaultNames[name]; isDefault {
			return nil, fmt.Errorf("%q is a default starter and must not be listed in OPTIONAL_STARTERS", name)
		}

		manifest, exists := c.optional[name]
		if !exists {
			available := strings.Join(c.optionalNames, ", ")
			if available == "" {
				available = "none"
			}
			return nil, fmt.Errorf("unknown optional starter %q (available: %s)", name, available)
		}
		selected = append(selected, manifest)
	}

	return selected, nil
}

// OptionalNames returns the available optional starter names in stable order.
func (c *Catalog) OptionalNames() []string {
	if c == nil {
		return nil
	}
	return slices.Clone(c.optionalNames)
}

func validateCatalogManifest(manifest assembly.StarterManifest) (string, error) {
	if isNilValue(manifest) {
		return "", fmt.Errorf("starter manifest is required")
	}
	name := manifest.Name()
	if !starterNamePattern.MatchString(name) {
		return "", fmt.Errorf("starter manifest name %q must be canonical lowercase", manifest.Name())
	}
	return name, nil
}

func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
