package pluginregistry

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryWith writes a registry holding one version of each given plugin.
func registryWith(t *testing.T, plugins ...fixturePlugin) *LocalSource {
	t.Helper()
	dir := t.TempDir()
	for _, plugin := range plugins {
		pluginDir := filepath.Join(dir, PluginsDir, plugin.manifest.Name)
		raw, err := json.Marshal(plugin.manifest)
		require.NoError(t, err)
		writeFile(t, pluginDir, ManifestFile, string(raw))
		for file, content := range plugin.files {
			writeFile(t, filepath.Join(pluginDir, FilesDir), file, content)
		}
	}
	index, err := BuildIndex(dir)
	require.NoError(t, err)
	raw, err := index.Marshal()
	require.NoError(t, err)
	writeFile(t, dir, IndexFile, string(raw))
	return NewLocalSource(dir, "")
}

type fixturePlugin struct {
	manifest Manifest
	files    map[string]string
}

const (
	notesFile  = "api/internal/modules/notes/notes.go"
	helperFile = "api/internal/modules/notes/helper.go"
	extraFile  = "api/internal/modules/notes/extra.go"
	notesFrag  = "contracts/fragments/notes.yaml"
	notesBody1 = "package notes\n\n// Limit caps notes.\nconst Limit = 10\n\n// Title names the page.\nconst Title = \"Notes\"\n"
	notesBody2 = "package notes\n\n// Limit caps notes.\nconst Limit = 20\n\n// Title names the page.\nconst Title = \"Notes\"\n"
)

func notesPlugin(version string, files map[string]string, wiring ...Insert) fixturePlugin {
	return fixturePlugin{
		manifest: Manifest{
			Name: "notes", Version: version, Summary: "Notes", Description: "Notes fixture.",
			UseWhen: []string{"notes"}, FastBuild: "0.24.0",
			Starter:  &StarterSpec{Name: "notes", Const: "StarterNotes", Directory: "notes", Handler: "Notes"},
			Fragment: notesFrag,
			Wiring:   wiring,
		},
		files: files,
	}
}

var featureInsert = Insert{Slot: "web.features", Lines: []string{"'notes',"}}

func installNotes(t *testing.T, root string, plugin fixturePlugin) (*Installer, *fakeContract) {
	t.Helper()
	contract := &fakeContract{root: root}
	installer := &Installer{
		Root: root, Source: registryWith(t, plugin), ScaffoldModule: testModule,
		KnownStarters: []string{"user", "organization"}, Contract: contract,
	}
	_, err := installer.Install("notes")
	require.NoError(t, err)
	return installer, contract
}

func v1Files() map[string]string {
	return map[string]string{notesFile: notesBody1, helperFile: "package notes\n", notesFrag: "paths: {}\n"}
}

func TestRemoveRestoresTheProjectExactly(t *testing.T) {
	root := projectWithSlots(t)
	installer, contract := installNotes(t, root, notesPlugin("0.1.0", v1Files(), featureInsert))

	result, err := installer.Remove("notes", RemoveOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{notesFile, helperFile}, result.Deleted)
	assert.Len(t, result.Unwired, 7)
	assert.Empty(t, result.Manual)
	for file, content := range slotFixtures {
		assert.Equal(t, content, readFile(t, root, file), file)
	}
	assert.NoDirExists(t, filepath.Join(root, "api/internal/modules/notes"))
	assert.NoFileExists(t, filepath.Join(root, LockFile))
	require.Len(t, contract.removed, 1)
	assert.Equal(t, []string{"/v1/demo-notes"}, contract.removed[0].Paths)
}

func TestRemoveRefusesLocalChangesWithoutForceAndReportsEditedWiring(t *testing.T) {
	root := projectWithSlots(t)
	installer, _ := installNotes(t, root, notesPlugin("0.1.0", v1Files(), featureInsert))
	writeFile(t, root, notesFile, "package notes\n// mine\n")

	_, err := installer.Remove("notes", RemoveOptions{})
	require.ErrorContains(t, err, "has local changes")
	assert.FileExists(t, filepath.Join(root, notesFile))

	writeFile(t, root, featuresFile, slotFixtures[featuresFile]) // the builder rewrote the slot
	result, err := installer.Remove("notes", RemoveOptions{Force: true})
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(root, notesFile))
	require.Len(t, result.Manual, 1)
	assert.Contains(t, result.Manual[0], "slot web.features")
}

func TestRemoveRefusesWhileAnotherPluginRequiresIt(t *testing.T) {
	root := projectWithSlots(t)
	installer, _ := installNotes(t, root, notesPlugin("0.1.0", v1Files()))
	lock, _ := ReadLock(root)
	lock.Plugins["notes-export"] = LockedEntry{Version: "0.1.0", Requires: []string{"notes"}, Files: map[string]string{}}
	require.NoError(t, lock.Write(root))

	_, err := installer.Remove("notes", RemoveOptions{})
	require.ErrorContains(t, err, "required by notes-export")
}

func TestUpdateReplacesUntouchedFilesMergesEditsAndRewires(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for three-way merges")
	}
	root := projectWithSlots(t)
	v1 := notesPlugin("0.1.0", v1Files())
	installer, contract := installNotes(t, root, v1)
	// The builder retitles the page; version 0.2.0 raises the limit, drops helper.go, adds extra.go,
	// and wires a Web feature.
	writeFile(t, root, notesFile, "package notes\n\n// Limit caps notes.\nconst Limit = 10\n\n// Title names the page.\nconst Title = \"Team notes\"\n")
	installer.Source = registryWith(t, notesPlugin("0.2.0", map[string]string{
		notesFile: notesBody2, extraFile: "package notes\n", notesFrag: "paths: {}\n",
	}, featureInsert))

	result, err := installer.Update("notes", registryWith(t, v1))
	require.NoError(t, err)
	assert.Equal(t, []string{notesFile}, result.Merged)
	assert.Equal(t, []string{extraFile}, result.Written)
	assert.Equal(t, []string{helperFile}, result.Deleted)
	assert.Empty(t, result.Conflicts)
	assert.False(t, result.ContractMerged, "an unchanged fragment keeps the merged contract")
	assert.Len(t, result.Wired, 1)
	merged := readFile(t, root, notesFile)
	assert.Contains(t, merged, "const Limit = 20")
	assert.Contains(t, merged, "\"Team notes\"")
	assert.Contains(t, readFile(t, root, featuresFile), "  'notes',\n  // fastbuild:slot web.features")
	assert.Empty(t, contract.removed)

	lock, _ := ReadLock(root)
	locked := lock.Plugins["notes"]
	assert.Equal(t, "0.2.0", locked.Version)
	assert.Len(t, locked.Wiring, 7)
	assert.NotContains(t, locked.Files, helperFile)
	assert.Equal(t, "modified", fileState(root, notesFile, locked.Files[notesFile]), "local edits stay visible")

	_, err = installer.Update("notes", nil)
	assert.ErrorContains(t, err, "already at 0.2.0")
}

func TestUpdateWritesConflictsAndIncomingVersions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required for three-way merges")
	}
	root := projectWithSlots(t)
	v1 := notesPlugin("0.1.0", v1Files())
	installer, _ := installNotes(t, root, v1)
	writeFile(t, root, notesFile, "package notes\n\n// Limit caps notes.\nconst Limit = 15\n\n// Title names the page.\nconst Title = \"Notes\"\n")
	writeFile(t, root, helperFile, "package notes\n// mine\n")
	installer.Source = registryWith(t, notesPlugin("0.2.0", map[string]string{
		notesFile: notesBody2, helperFile: "package notes\n// upstream\n", notesFrag: "paths: {}\n",
	}))

	result, err := installer.Update("notes", registryWith(t, v1))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{notesFile, helperFile}, result.Conflicts)
	assert.Contains(t, readFile(t, root, notesFile), ConflictMarker)
	problems := Doctor(root, mustLock(t, root))
	require.NotEmpty(t, problems)
	assert.Contains(t, problems[0].Problem, "unresolved update conflicts")
}

func TestUpdateWithoutTheInstalledVersionSavesIncomingFiles(t *testing.T) {
	root := projectWithSlots(t)
	installer, _ := installNotes(t, root, notesPlugin("0.1.0", v1Files()))
	writeFile(t, root, notesFile, "package notes\n// mine\n")
	installer.Source = registryWith(t, notesPlugin("0.2.0", map[string]string{
		notesFile: notesBody2, helperFile: "package notes\n", notesFrag: "paths: {}\n",
	}))

	result, err := installer.Update("notes", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{notesFile}, result.Incoming)
	assert.Equal(t, "package notes\n// mine\n", readFile(t, root, notesFile))
	assert.Equal(t, notesBody2, readFile(t, root, notesFile+IncomingSuffix))
	problems := Doctor(root, mustLock(t, root))
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Problem, "waits to be merged")
}

func TestUpdateReplacesAChangedContract(t *testing.T) {
	root := projectWithSlots(t)
	installer, contract := installNotes(t, root, notesPlugin("0.1.0", v1Files()))
	installer.Source = registryWith(t, notesPlugin("0.2.0", map[string]string{
		notesFile: notesBody1, helperFile: "package notes\n", notesFrag: "paths: {changed: true}\n",
	}))

	result, err := installer.Update("notes", nil)
	require.NoError(t, err)
	assert.True(t, result.ContractMerged)
	require.Len(t, contract.removed, 1)
	assert.Equal(t, []string{notesFrag, notesFrag}, contract.merged)
	assert.NoFileExists(t, filepath.Join(root, notesFrag))
}

func TestOnlyOneThemeIsInstalled(t *testing.T) {
	root := projectWithSlots(t)
	writeFile(t, root, "web/src/themes/index.css", "@import \"./light.css\";\n/* fastbuild:slot web.theme */\n")
	theme := func(name string) fixturePlugin {
		return fixturePlugin{
			manifest: Manifest{
				Name: name, Kind: KindTheme, Version: "0.1.0", Summary: "Theme", Description: "Theme fixture.",
				UseWhen: []string{"style"}, FastBuild: "0.24.0",
				Wiring: []Insert{{Slot: "web.theme", Lines: []string{`@import "./presets/` + name + `.css";`}}},
			},
			files: map[string]string{"web/src/themes/presets/" + name + ".css": ":root {}\n"},
		}
	}
	installer := &Installer{Root: root, Source: registryWith(t, theme("ocean"), theme("forest")), ScaffoldModule: testModule}
	_, err := installer.Install("ocean")
	require.NoError(t, err)
	assert.Equal(t, "@import \"./light.css\";\n@import \"./presets/ocean.css\";\n/* fastbuild:slot web.theme */\n",
		readFile(t, root, "web/src/themes/index.css"))

	_, err = installer.Install("forest")
	require.ErrorContains(t, err, `theme "ocean" is installed`)
}

func TestDoctorReportsUnfinishedIntegration(t *testing.T) {
	root := projectWithSlots(t)
	installer, _ := installNotes(t, root, notesPlugin("0.1.0", v1Files(), featureInsert))
	assert.Empty(t, Doctor(root, mustLock(t, root)))

	writeFile(t, root, featuresFile, slotFixtures[featuresFile])
	writeFile(t, root, ContractFile, slotFixtures[ContractFile])
	problems := Doctor(root, mustLock(t, root))
	require.Len(t, problems, 2)
	assert.Contains(t, problems[0].Problem, "slot web.features")
	assert.Contains(t, problems[1].Problem, "lacks /v1/demo-notes")

	installer.Contract = nil
	_, err := installer.Remove("notes", RemoveOptions{})
	require.NoError(t, err)
}

func mustLock(t *testing.T, root string) *Lock {
	t.Helper()
	lock, err := ReadLock(root)
	require.NoError(t, err)
	return lock
}

func TestSlotsInsertWithIndentationAndSurviveFormatting(t *testing.T) {
	source := []byte(slotFixtures[startersFile])
	updated, added, err := applyInsert(startersFile, source, "api.starter.constants", []string{`StarterLongerName = "longer"`})
	require.NoError(t, err)
	require.True(t, added)
	// gofmt realigns the const block, so StarterUser gains padding; removal still matches.
	assert.Contains(t, string(updated), "StarterUser       = \"user\"")

	_, added, err = applyInsert(startersFile, updated, "api.starter.constants", []string{`StarterLongerName   =   "longer"`})
	require.NoError(t, err)
	assert.False(t, added, "equivalent lines are not inserted twice")

	restored, removed, err := removeInsert(startersFile, updated, []string{`StarterLongerName = "longer"`})
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, slotFixtures[startersFile], string(restored))

	_, _, err = applyInsert(startersFile, source, "api.starter.unknown", []string{"x"})
	assert.ErrorIs(t, err, errSlotMissing)
}

func TestManifestValidatesKindsAndWiring(t *testing.T) {
	base := Manifest{Name: "x", Version: "0.1.0", Summary: "s", Description: "d", UseWhen: []string{"u"}, FastBuild: "0.25.0"}
	theme := base
	theme.Kind = KindTheme
	assert.ErrorContains(t, theme.Validate(), "a theme must wire at least one *.theme slot")
	theme.Fragment = "contracts/fragments/x.yaml"
	assert.ErrorContains(t, theme.Validate(), "only feature plugins install")

	wired := base
	wired.Wiring = []Insert{{Slot: "web.nowhere", Lines: []string{"x"}}}
	assert.ErrorContains(t, wired.Validate(), `"web.nowhere" is not a project slot`)
	wired.Wiring = []Insert{{Slot: "web.features", Lines: []string{"// fastbuild:slot web.features"}}}
	assert.ErrorContains(t, wired.Validate(), "slot marker")
	unknown := base
	unknown.Kind = "widget"
	assert.ErrorContains(t, unknown.Validate(), "kind must be")
}

func TestTemplatesInstallTheirPluginsTogetherOrNotAtAll(t *testing.T) {
	root := projectWithSlots(t)
	writeFile(t, root, "web/src/themes/index.css", "/* fastbuild:slot web.theme */\n")
	block := func(name string, requires ...string) fixturePlugin {
		return fixturePlugin{
			manifest: Manifest{Name: name, Kind: KindBlock, Version: "0.1.0", Summary: "Block", Description: "Block.",
				UseWhen: []string{"page"}, FastBuild: "0.24.0", Requires: Requirements{Plugins: requires}},
			files: map[string]string{"web/src/components/blocks/" + name + "/index.tsx": "export {};\n"},
		}
	}
	template := fixturePlugin{manifest: Manifest{
		Name: "portal", Kind: KindTemplate, Version: "0.1.0", Summary: "Portal", Description: "Portal template.",
		UseWhen: []string{"portal"}, FastBuild: "0.24.0", Requires: Requirements{Starters: []string{"organization"}},
		Includes: []string{"block-a", "block-b"},
	}}
	installer := &Installer{
		Root: root, Source: registryWith(t, block("block-a"), block("block-b", "block-a"), template),
		ScaffoldModule: testModule, KnownStarters: []string{"organization"},
	}

	installer.DryRun = true
	planned, err := installer.InstallTemplate("portal")
	require.NoError(t, err, "a dry run treats earlier plugins as installed")
	assert.Len(t, planned.Installed, 2)
	assert.NoFileExists(t, filepath.Join(root, LockFile))

	installer.DryRun = false
	writeFile(t, root, "web/src/components/blocks/block-b/index.tsx", "mine\n")
	_, err = installer.InstallTemplate("portal")
	require.ErrorContains(t, err, "would overwrite existing files")
	assert.NoFileExists(t, filepath.Join(root, "web/src/components/blocks/block-a/index.tsx"), "block-a is removed again")
	assert.NoFileExists(t, filepath.Join(root, LockFile))

	require.NoError(t, os.Remove(filepath.Join(root, "web/src/components/blocks/block-b/index.tsx")))
	result, err := installer.InstallTemplate("portal")
	require.NoError(t, err)
	assert.Len(t, result.Installed, 2)
	assert.Len(t, mustLock(t, root).Plugins, 2)

	_, err = installer.InstallTemplate("block-a")
	assert.ErrorContains(t, err, "not a template")
}
