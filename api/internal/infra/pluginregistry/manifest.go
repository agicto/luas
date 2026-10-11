// Package pluginregistry finds, verifies, and installs FastBuild plugins: versioned capabilities
// distributed as source from a registry and owned by the project once installed.
package pluginregistry

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

// ManifestFile is the manifest name inside each plugin directory.
const ManifestFile = "plugin.json"

// FilesDir holds a plugin's files laid out exactly as they land in the project.
const FilesDir = "files"

var (
	pluginNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	starterNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	goIdentPattern     = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	envNamePattern     = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

// Manifest describes one plugin. It is the contract between plugin authors, the CLI, and coding
// agents: everything the CLI does deterministically and everything an agent needs to decide
// whether the plugin fits is declared here.
type Manifest struct {
	Name string `json:"name"`
	// Kind is feature (the default), theme, or block.
	Kind        string   `json:"kind,omitempty"`
	Version     string   `json:"version"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords,omitempty"`
	// UseWhen lists situations, in the builder's words, where the plugin is the right choice.
	// Agents match requests against it.
	UseWhen []string `json:"use_when,omitempty"`
	// FastBuild is the minimum scaffold version the plugin supports, for example "0.25.0".
	FastBuild string       `json:"fastbuild"`
	Requires  Requirements `json:"requires"`
	Starter   *StarterSpec `json:"starter,omitempty"`
	// Fragment is the repository path of the OpenAPI fragment the plugin adds, if any.
	Fragment string `json:"fragment,omitempty"`
	// Wiring inserts lines at named slots in core files, for example a Web navigation item.
	Wiring []Insert `json:"wiring,omitempty"`
	Env    []EnvVar `json:"env,omitempty"`
}

// Plugin kinds.
const (
	// KindFeature is a capability: API module, contract, and optional browser screens.
	KindFeature = "feature"
	// KindTheme restyles the browser shells through design tokens; one theme is installed at a time.
	KindTheme = "theme"
	// KindBlock is a reusable page section the builder places where it fits.
	KindBlock = "block"
)

// PluginKind returns the manifest kind, defaulting to feature.
func (m Manifest) PluginKind() string {
	if m.Kind == "" {
		return KindFeature
	}
	return m.Kind
}

// AllInserts returns the starter wiring followed by the explicit wiring, with scaffoldModule as
// the module path in generated imports.
func (m Manifest) AllInserts(scaffoldModule string) []Insert {
	var inserts []Insert
	if m.Starter != nil {
		inserts = append(inserts, m.Starter.Inserts(scaffoldModule)...)
	}
	return append(inserts, m.Wiring...)
}

// Requirements names what must already exist in the project.
type Requirements struct {
	Starters []string `json:"starters,omitempty"`
	Plugins  []string `json:"plugins,omitempty"`
}

// StarterSpec describes the optional starter a plugin installs, so the CLI can wire it with the
// same edits as make:module.
type StarterSpec struct {
	// Name is the OPTIONAL_STARTERS value, for example "knowledge".
	Name string `json:"name"`
	// Const is the config constant, for example "StarterKnowledge".
	Const string `json:"const"`
	// Directory is the module directory under api/internal/modules.
	Directory string `json:"directory"`
	// Package is the Go package name; it defaults to Directory.
	Package string `json:"package,omitempty"`
	// Handler is the field name on starter.Handlers, for example "Knowledge".
	Handler string `json:"handler"`
}

// EnvVar documents a configuration variable the plugin reads.
type EnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Inserts returns the slot edits that register the starter, the same edits make:module makes.
func (s StarterSpec) Inserts(module string) []Insert {
	pkg := s.PackageName()
	importLine := `"` + module + "/internal/modules/" + s.Directory + `"`
	if pkg != s.Directory {
		// goimports requires an alias when the package name differs from its directory.
		importLine = pkg + " " + importLine
	}
	return []Insert{
		{Slot: "api.starter.constants", Lines: []string{s.Const + ` = "` + s.Name + `"`}},
		{Slot: "api.starter.names", Lines: []string{s.Const + ","}},
		{Slot: "api.starter.imports", Lines: []string{importLine}},
		{Slot: "api.starter.providers", Lines: []string{pkg + ".ProviderSet,"}},
		{Slot: "api.starter.handlers", Lines: []string{s.Handler + " *" + pkg + ".Handler"}},
		{Slot: "api.starter.manifests", Lines: []string{pkg + ".NewStarterManifest(h." + s.Handler + "),"}},
	}
}

// PackageName returns the Go package name of the installed starter.
func (s StarterSpec) PackageName() string {
	if s.Package != "" {
		return s.Package
	}
	return s.Directory
}

// Validate reports every problem with the manifest at once.
func (m Manifest) Validate() error {
	var problems []string
	if !pluginNamePattern.MatchString(m.Name) || len(m.Name) > 50 {
		problems = append(problems, "name must be lowercase words joined by hyphens, at most 50 characters")
	}
	if _, err := ParseVersion(m.Version); err != nil {
		problems = append(problems, "version must be semantic, for example 0.1.0")
	}
	if _, err := ParseVersion(m.FastBuild); err != nil {
		problems = append(problems, "fastbuild must be the minimum scaffold version, for example 0.25.0")
	}
	kind := m.PluginKind()
	if kind != KindFeature && kind != KindTheme && kind != KindBlock {
		problems = append(problems, "kind must be feature, theme, or block")
	}
	if kind != KindFeature && (m.Starter != nil || m.Fragment != "") {
		problems = append(problems, "only feature plugins install a starter or a contract fragment")
	}
	if kind == KindTheme && !slices.ContainsFunc(m.Wiring, func(insert Insert) bool { return strings.HasSuffix(insert.Slot, ".theme") }) {
		problems = append(problems, "a theme must wire at least one *.theme slot")
	}
	for _, insert := range m.Wiring {
		if err := insert.validate(); err != nil {
			problems = append(problems, err.Error())
		} else if _, ok := FindSlot(insert.Slot); !ok {
			problems = append(problems, fmt.Sprintf("wiring slot %q is not a project slot; see plugin:slots", insert.Slot))
		}
	}
	if strings.TrimSpace(m.Summary) == "" || len(m.Summary) > 120 {
		problems = append(problems, "summary is required and at most 120 characters")
	}
	if strings.TrimSpace(m.Description) == "" {
		problems = append(problems, "description is required")
	}
	if len(m.UseWhen) == 0 {
		problems = append(problems, "use_when needs at least one situation so agents can match requests")
	}
	for _, name := range m.Requires.Starters {
		if !starterNamePattern.MatchString(name) {
			problems = append(problems, fmt.Sprintf("requires.starters has invalid name %q", name))
		}
	}
	for _, name := range m.Requires.Plugins {
		if !pluginNamePattern.MatchString(name) || name == m.Name {
			problems = append(problems, fmt.Sprintf("requires.plugins has invalid name %q", name))
		}
	}
	if s := m.Starter; s != nil {
		if !starterNamePattern.MatchString(s.Name) {
			problems = append(problems, "starter.name must be lowercase letters and digits")
		}
		if !goIdentPattern.MatchString(s.Const) || !strings.HasPrefix(s.Const, "Starter") {
			problems = append(problems, "starter.const must be an exported identifier starting with Starter")
		}
		if !starterNamePattern.MatchString(strings.ReplaceAll(s.Directory, "_", "")) {
			problems = append(problems, "starter.directory must be a module directory name")
		}
		if !goIdentPattern.MatchString(s.Handler) {
			problems = append(problems, "starter.handler must be an exported identifier")
		}
		if s.Name != s.PackageName() {
			problems = append(problems, "starter.name must equal the Go package name, which names the starter")
		}
	}
	if m.Fragment != "" && (!strings.HasPrefix(m.Fragment, "contracts/fragments/") || path.Ext(m.Fragment) != ".yaml") {
		problems = append(problems, "fragment must be a contracts/fragments/*.yaml path")
	}
	for _, variable := range m.Env {
		if !envNamePattern.MatchString(variable.Name) || strings.TrimSpace(variable.Description) == "" {
			problems = append(problems, fmt.Sprintf("env %q needs an uppercase name and a description", variable.Name))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("plugin %q manifest: %s", m.Name, strings.Join(problems, "; "))
	}
	return nil
}

// ValidFilePath reports whether a plugin file path is a safe, repository-relative path.
func ValidFilePath(name string) bool {
	return projectfiles.SafeRelativePath(name)
}
