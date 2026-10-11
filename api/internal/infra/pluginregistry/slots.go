package pluginregistry

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// SlotMarker starts every slot marker comment, for example `// fastbuild:slot api.starter.names`
// in Go and TypeScript or `/* fastbuild:slot web.theme */` in CSS.
const SlotMarker = "fastbuild:slot "

var slotNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)

// Slot is a named extension point: a marker line in a core file before which plugins insert
// lines. Slots are the only places plugins edit core files, so every edit is visible in the
// source, recorded in the lock file, and reversible.
type Slot struct {
	Name        string `json:"name"`
	File        string `json:"file"`
	Description string `json:"description"`
}

// ProjectSlots is the catalog of extension points in a FastBuild project. A guard test keeps it
// aligned with the markers in the source.
var ProjectSlots = []Slot{
	{"api.starter.constants", "api/internal/infra/config/starters.go", "Starter name constant, for example `StarterKnowledge = \"knowledge\"`"},
	{"api.starter.names", "api/internal/infra/config/starters.go", "Starter constant in StarterNames()"},
	{"api.starter.imports", "api/internal/starter/defaults.go", "Import of the starter module package"},
	{"api.starter.providers", "api/internal/starter/defaults.go", "The module's Wire ProviderSet"},
	{"api.starter.handlers", "api/internal/starter/defaults.go", "The module's field on starter.Handlers"},
	{"api.starter.manifests", "api/internal/starter/defaults.go", "The module's manifest in OptionalManifests()"},
	{"web.features", "web/src/config/optional-features.ts", "Optional Web feature name"},
	{"web.feature-dependencies", "web/src/config/optional-features.ts", "Features an optional Web feature requires"},
	{"web.http.error-codes", "web/src/http/codes.ts", "API error code constant from the plugin's contract fragment, so the Go API adapter forwards it"},
	{"web.routes.console", "web/src/constants/routes.ts", "Console route constant"},
	{"web.console.nav-icons", "web/src/app/(protected)/(console)/console/layout.tsx", "lucide-react icon import for a console navigation item"},
	{"web.console.nav", "web/src/app/(protected)/(console)/console/layout.tsx", "Console navigation item guarded by its feature"},
	{"web.i18n.modules", "web/src/i18n/module-names.ts", "Message module name"},
	{"web.i18n.loaders", "web/src/i18n/loader.ts", "Per-locale message module loaders"},
	{"web.i18n.message-imports", "web/src/i18n/modules/index.ts", "Import of the source-locale messages and their type"},
	{"web.i18n.messages", "web/src/i18n/modules/index.ts", "Source-locale messages entry"},
	{"web.i18n.message-schema", "web/src/i18n/modules/index.ts", "Message schema entry"},
	{"web.i18n.zh-hans-schema", "web/src/i18n/modules/index.ts", "zh-Hans message schema entry"},
	{"web.i18n.client-namespaces", "web/src/i18n/client-message-namespaces.ts", "Client message namespaces of a route scope"},
	{"web.i18n.nav.en-us", "web/src/i18n/modules/nav/en-US.ts", "English navigation label"},
	{"web.i18n.nav.zh-hans", "web/src/i18n/modules/nav/zh-Hans.ts", "Chinese navigation label"},
	{"web.theme", "web/src/themes/index.css", "Theme stylesheet import; at most one theme is installed"},
	{"admin.theme", "admin/src/styles/globals.css", "Theme stylesheet import; at most one theme is installed"},
}

// FindSlot returns the named slot from the project catalog.
func FindSlot(name string) (Slot, bool) {
	for _, slot := range ProjectSlots {
		if slot.Name == name {
			return slot, true
		}
	}
	return Slot{}, false
}

// Insert adds lines before a slot marker. Lines are written without the marker's indentation,
// which the CLI adds.
type Insert struct {
	Slot  string   `json:"slot"`
	Lines []string `json:"lines"`
}

// AppliedInsert records an insert the CLI made, so remove and update can reverse exactly it.
type AppliedInsert struct {
	Slot  string   `json:"slot"`
	File  string   `json:"file"`
	Lines []string `json:"lines"`
}

func (i Insert) validate() error {
	if !slotNamePattern.MatchString(i.Slot) {
		return fmt.Errorf("wiring slot %q is not a slot name", i.Slot)
	}
	if len(i.Lines) == 0 {
		return fmt.Errorf("wiring for slot %q has no lines", i.Slot)
	}
	for _, line := range i.Lines {
		if strings.ContainsAny(line, "\n\r") || strings.Contains(line, SlotMarker) {
			return fmt.Errorf("wiring for slot %q has a line with a newline or a slot marker", i.Slot)
		}
	}
	return nil
}

// errSlotMissing reports a slot the project does not have, for example after web/ was removed.
var errSlotMissing = errors.New("slot marker not found")

// applyInsert inserts lines before the slot marker in content. It reports false when equivalent
// lines are already there, so the CLI never claims lines it did not add.
func applyInsert(file string, content []byte, slot string, lines []string) ([]byte, bool, error) {
	source := strings.Split(string(content), "\n")
	marker, err := markerLine(source, slot)
	if err != nil {
		return nil, false, err
	}
	if findLines(source, lines) >= 0 {
		return content, false, nil
	}
	indent := source[marker][:len(source[marker])-len(strings.TrimLeft(source[marker], " \t"))]
	inserted := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			inserted = append(inserted, "")
			continue
		}
		inserted = append(inserted, indent+line)
	}
	updated := slices.Concat(source[:marker], inserted, source[marker:])
	formatted, err := formatSource(file, []byte(strings.Join(updated, "\n")))
	return formatted, err == nil, err
}

// removeInsert deletes the recorded lines wherever formatting moved them. It reports false when
// they are no longer there, for example after a builder edited them.
func removeInsert(file string, content []byte, lines []string) ([]byte, bool, error) {
	source := strings.Split(string(content), "\n")
	start := findLines(source, lines)
	if start < 0 {
		return content, false, nil
	}
	updated := slices.Concat(source[:start], source[start+len(lines):])
	formatted, err := formatSource(file, []byte(strings.Join(updated, "\n")))
	return formatted, err == nil, err
}

func markerLine(source []string, slot string) (int, error) {
	found := -1
	for index, line := range source {
		if slotOnLine(line) == slot {
			if found >= 0 {
				return -1, fmt.Errorf("slot %s is marked twice", slot)
			}
			found = index
		}
	}
	if found < 0 {
		return -1, fmt.Errorf("%w: %s", errSlotMissing, slot)
	}
	return found, nil
}

// slotOnLine returns the slot named by a line that is exactly a marker comment, or "".
func slotOnLine(line string) string {
	trimmed := strings.TrimSpace(line)
	name, ok := strings.CutPrefix(trimmed, "// "+SlotMarker)
	if !ok {
		if name, ok = strings.CutPrefix(trimmed, "/* "+SlotMarker); !ok {
			return ""
		}
		if name, ok = strings.CutSuffix(name, " */"); !ok {
			return ""
		}
	}
	if !slotNamePattern.MatchString(name) {
		return ""
	}
	return name
}

// findLines returns where lines appear contiguously in source, comparing whitespace-normalized
// text so gofmt alignment and re-indentation still match. Blank lines must match blank lines.
func findLines(source, lines []string) int {
	if len(lines) == 0 {
		return -1
	}
	for start := 0; start+len(lines) <= len(source); start++ {
		matched := true
		for offset, line := range lines {
			if normalizeLine(source[start+offset]) != normalizeLine(line) {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

func normalizeLine(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

// formatSource keeps Go files gofmt-clean after an edit; other files keep their text.
func formatSource(file string, content []byte) ([]byte, error) {
	if filepath.Ext(file) != ".go" {
		return content, nil
	}
	formatted, err := format.Source(content)
	if err != nil {
		return nil, fmt.Errorf("%s does not compile after the edit: %w", file, err)
	}
	return formatted, nil
}

// ScanSlotMarkers lists every slot marker under the given repository directories, keyed by slot
// name with the files that mark it. Guards use it to keep ProjectSlots and the source aligned.
func ScanSlotMarkers(root string, dirs ...string) (map[string][]string, error) {
	found := map[string][]string{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, fs.ErrNotExist) {
					return nil
				}
				return walkErr
			}
			if entry.IsDir() {
				if name := entry.Name(); name == "node_modules" || name == "testdata" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			switch name := entry.Name(); {
			case strings.HasSuffix(name, "_test.go"), strings.Contains(name, ".test."):
				return nil // fixtures
			case !slices.Contains([]string{".go", ".ts", ".tsx", ".css"}, filepath.Ext(name)):
				return nil
			}
			content, err := os.ReadFile(current)
			if err != nil || !bytes.Contains(content, []byte(SlotMarker)) {
				return err
			}
			relative, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			for _, line := range strings.Split(string(content), "\n") {
				if slot := slotOnLine(line); slot != "" {
					found[slot] = append(found[slot], filepath.ToSlash(relative))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return found, nil
}

// ApplyInserts adds inserts at the project's slots under root, writing every file or none, and
// returns the inserts it applied. make:module uses it to register a new starter exactly as
// plugin:add does.
func ApplyInserts(root string, inserts []Insert) ([]AppliedInsert, error) {
	changes := newChangeSet(root)
	var applied []AppliedInsert
	for _, insert := range inserts {
		slot, ok := FindSlot(insert.Slot)
		if !ok {
			return nil, fmt.Errorf("unknown slot %q", insert.Slot)
		}
		added, err := changes.applyInsert(slot, insert.Lines)
		if err != nil {
			return nil, err
		}
		if added {
			applied = append(applied, AppliedInsert{Slot: slot.Name, File: slot.File, Lines: insert.Lines})
		}
	}
	return applied, changes.commit()
}
