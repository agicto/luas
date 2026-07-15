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
	dependencies  map[string][]string
}

// NewCatalog creates and validates one deterministic starter catalog.
func NewCatalog(defaults, optional []assembly.StarterManifest) (*Catalog, error) {
	catalog := &Catalog{
		defaults:     make([]assembly.StarterManifest, 0, len(defaults)),
		defaultNames: make(map[string]struct{}, len(defaults)),
		optional:     make(map[string]assembly.StarterManifest, len(optional)),
		dependencies: make(map[string][]string, len(defaults)+len(optional)),
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
	if err := catalog.validateDependencies(); err != nil {
		return nil, err
	}

	return catalog, nil
}

// Select returns all defaults followed by optional starters in configured order.
func (c *Catalog) Select(names []string) ([]assembly.StarterManifest, error) {
	if c == nil {
		return nil, fmt.Errorf("starter catalog is required")
	}

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

		if _, exists := c.optional[name]; !exists {
			available := strings.Join(c.optionalNames, ", ")
			if available == "" {
				available = "none"
			}
			return nil, fmt.Errorf("unknown optional starter %q (available: %s)", name, available)
		}
	}
	for name := range seen {
		for _, dependency := range c.dependencies[name] {
			if _, isDefault := c.defaultNames[dependency]; isDefault {
				continue
			}
			if _, selected := seen[dependency]; !selected {
				return nil, fmt.Errorf("optional starter %q requires %q in OPTIONAL_STARTERS", name, dependency)
			}
		}
	}

	selected := append([]assembly.StarterManifest(nil), c.defaults...)
	assembled := make(map[string]struct{}, len(seen))
	var appendWithDependencies func(string)
	appendWithDependencies = func(name string) {
		if _, exists := assembled[name]; exists {
			return
		}
		for _, dependency := range c.dependencies[name] {
			if _, isOptional := c.optional[dependency]; isOptional {
				appendWithDependencies(dependency)
			}
		}
		assembled[name] = struct{}{}
		selected = append(selected, c.optional[name])
	}
	for _, name := range names {
		appendWithDependencies(name)
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

func (c *Catalog) validateDependencies() error {
	all := make(map[string]assembly.StarterManifest, len(c.defaults)+len(c.optional))
	for _, manifest := range c.defaults {
		all[manifest.Name()] = manifest
	}
	for name, manifest := range c.optional {
		all[name] = manifest
	}

	for name, manifest := range all {
		seen := make(map[string]struct{}, len(manifest.Dependencies()))
		for _, dependency := range manifest.Dependencies() {
			if dependency == "" || dependency != strings.TrimSpace(dependency) || !starterNamePattern.MatchString(dependency) {
				return fmt.Errorf("starter %q dependency %q must use a canonical lowercase name", name, dependency)
			}
			if dependency == name {
				return fmt.Errorf("starter %q cannot depend on itself", name)
			}
			if _, duplicate := seen[dependency]; duplicate {
				return fmt.Errorf("starter %q has duplicate dependency %q", name, dependency)
			}
			if _, exists := all[dependency]; !exists {
				return fmt.Errorf("starter %q depends on unknown starter %q", name, dependency)
			}
			if _, isDefault := c.defaultNames[name]; isDefault {
				if _, dependencyIsDefault := c.defaultNames[dependency]; !dependencyIsDefault {
					return fmt.Errorf("default starter %q cannot depend on optional starter %q", name, dependency)
				}
			}
			seen[dependency] = struct{}{}
			c.dependencies[name] = append(c.dependencies[name], dependency)
		}
	}

	state := make(map[string]uint8, len(all))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("starter dependency cycle includes %q", name)
		case 2:
			return nil
		}
		state[name] = 1
		for _, dependency := range c.dependencies[name] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[name] = 2
		return nil
	}
	for name := range all {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
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
