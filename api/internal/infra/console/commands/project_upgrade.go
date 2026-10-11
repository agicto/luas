package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fastagi/fastbuild/api/internal/infra/console"
	"github.com/fastagi/fastbuild/api/internal/infra/pluginregistry"
	"github.com/fastagi/fastbuild/api/internal/infra/projectfiles"
)

// ProjectUpgradeCommand moves an initialized project to a newer FastBuild release, keeping the
// project's own changes.
type ProjectUpgradeCommand struct{ output *console.Output }

func NewProjectUpgradeCommand() *ProjectUpgradeCommand {
	return &ProjectUpgradeCommand{output: console.NewOutput()}
}

func (c *ProjectUpgradeCommand) Name() string { return "project:upgrade" }
func (c *ProjectUpgradeCommand) Description() string {
	return "Upgrade this project to a newer FastBuild release, merging the project's changes"
}
func (c *ProjectUpgradeCommand) Usage() string {
	return "project:upgrade --to=v0.26.0 [--dry-run] [--format=text|json]"
}

func (c *ProjectUpgradeCommand) Run(args []string) error {
	_, format, err := positionalAndFormat(args, "text")
	if err != nil {
		return err
	}
	to, ok := flagValue(args, "to")
	if !ok || to == "" {
		return errors.New("usage: " + c.Usage())
	}
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	marker, err := readProjectMarker(root)
	if err != nil {
		return fmt.Errorf("project:upgrade needs %s from project:init: %w", projectMarkerFile, err)
	}
	from, err := pluginregistry.ParseVersion(marker.ScaffoldVersion)
	if err != nil {
		return fmt.Errorf("%s records %q, which is not a release", projectMarkerFile, marker.ScaffoldVersion)
	}
	target, err := pluginregistry.ParseVersion(to)
	if err != nil {
		return fmt.Errorf("--to must be a release such as v0.26.0: %w", err)
	}
	if from.AtLeast(target) {
		return fmt.Errorf("this project is already on FastBuild %s", from)
	}

	baseDir, targetDir, cleanup, err := c.releases(args, "v"+from.String(), "v"+target.String(), format != "json")
	if err != nil {
		return err
	}
	defer cleanup()

	upgraded := *marker
	upgraded.ScaffoldVersion = "v" + target.String()
	encodedMarker, err := json.MarshalIndent(upgraded, "", "  ")
	if err != nil {
		return err
	}
	upgrade := &projectfiles.Upgrade{
		Root: root, Base: baseDir, Target: targetDir,
		Adapt: projectRewriter(ProjectOptions{
			Name: marker.Name, Slug: marker.Slug, Module: marker.Module, Repository: marker.Repository, Starters: marker.Starters,
		}),
		Skip: func(path string) bool {
			return strings.HasPrefix(path, registryDir+"/") || path == projectMarkerFile || path == pluginregistry.LockFile
		},
		Extra:  map[string][]byte{projectMarkerFile: append(encodedMarker, '\n')},
		DryRun: hasFlag(args, "dry-run"),
	}
	result, err := upgrade.Run()
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(map[string]any{"from": "v" + from.String(), "to": "v" + target.String(), "dry_run": upgrade.DryRun, "result": result})
	}
	c.report(result, from.String(), target.String(), upgrade.DryRun)
	return nil
}

// releases returns directories holding the two releases: --base-dir and --target-dir when given
// (an offline or pre-extracted copy), and otherwise the GitHub archives of the scaffold repository.
func (c *ProjectUpgradeCommand) releases(args []string, from, to string, verbose bool) (string, string, func(), error) {
	baseDir, hasBase := flagValue(args, "base-dir")
	targetDir, hasTarget := flagValue(args, "target-dir")
	if hasBase && hasTarget {
		return baseDir, targetDir, func() {}, nil
	}
	var temporary []string
	cleanup := func() {
		for _, dir := range temporary {
			_ = os.RemoveAll(dir)
		}
	}
	owner, repo, _ := strings.Cut(scaffoldGitHubRepository, "/")
	fetch := func(ref string) (string, error) {
		dir, err := os.MkdirTemp("", "fastbuild-release-*")
		if err != nil {
			return "", err
		}
		temporary = append(temporary, dir)
		if verbose {
			c.output.Info("Fetching FastBuild %s ...", ref)
		}
		if err := projectfiles.FetchGitHubArchive(context.Background(), nil, owner, repo, ref, "", dir); err != nil {
			return "", fmt.Errorf("fetch FastBuild %s: %w", ref, err)
		}
		return dir, nil
	}
	base, err := fetch(from)
	if err != nil {
		cleanup()
		return "", "", nil, err
	}
	target, err := fetch(to)
	if err != nil {
		cleanup()
		return "", "", nil, err
	}
	return base, target, cleanup, nil
}

func (c *ProjectUpgradeCommand) report(result *projectfiles.UpgradeResult, from, to string, dryRun bool) {
	verb := "Upgraded"
	if dryRun {
		verb = "Dry run: would upgrade"
	}
	c.output.Success("%s FastBuild %s → %s", verb, from, to)
	groups := []struct {
		mark  string
		files []string
	}{
		{"+ added", result.Added}, {"~ replaced", result.Replaced}, {"~ merged", result.Merged},
		{"! conflict", result.Conflicts}, {"? incoming", result.Incoming}, {"- deleted", result.Deleted},
		{"= kept", result.Kept},
	}
	for _, group := range groups {
		for _, file := range group.files {
			c.output.Line("  %s %s", group.mark, file)
		}
	}
	if dryRun {
		return
	}
	c.output.Section("Next")
	if len(result.Conflicts)+len(result.Incoming) > 0 {
		c.output.Line("  1. Resolve every %q conflict and merge every *%s file, keeping the project's intent.",
			projectfiles.ConflictMarker, projectfiles.IncomingSuffix)
	}
	c.output.Line("  - Read UPGRADING.md for %s → %s and apply its notes.", from, to)
	c.output.Line("  - cd api && go mod tidy && make wire; cd contracts && corepack pnpm install && corepack pnpm generate")
	c.output.Line("  - cd api && go run ./cmd/fastbuild plugin:doctor, then make check from the repository root.")
}
