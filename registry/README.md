# FastBuild Plugin Registry

Official FastBuild plugins: versioned features, themes, and blocks distributed as source. A plugin is
installed into a project with one command; from then on the project owns the code, and the lock file
remembers exactly what the plugin added so it can be updated or removed later.

```bash
cd api
go run ./cmd/fastbuild plugin:search knowledge      # find (also: --format=json for agents)
go run ./cmd/fastbuild plugin:info knowledge-base   # decide: requirements, files, integration steps
go run ./cmd/fastbuild plugin:add knowledge-base    # install (--dry-run previews every change)
go run ./cmd/fastbuild plugin:doctor                # unfinished integration steps
go run ./cmd/fastbuild plugin:status                # files changed since installation
go run ./cmd/fastbuild plugin:update knowledge-base --source=github:fastagi/fastbuild@v0.26.0
go run ./cmd/fastbuild plugin:remove knowledge-base
```

A project created with `make init-project` installs plugins from the FastBuild release it started
from; this directory is not copied into new projects.

## For Coding Agents

Agents read the `plugin-integration` skill and call the commands with `--format=json`. Agents that
speak the Model Context Protocol can use the read-only tools of `fastbuild mcp` instead
(`project_info`, `plugin_search`, `plugin_info`, `plugin_preview`, `plugin_status`, `plugin_doctor`,
`plugin_slots`); installing stays an explicit CLI step the builder sees.

```bash
claude mcp add fastbuild -- sh -c 'cd api && go run ./cmd/fastbuild mcp'   # Claude Code
codex mcp add fastbuild -- sh -c 'cd api && go run ./cmd/fastbuild mcp'    # Codex
```

## Kinds

| Kind | Installs | Example |
|---|---|---|
| `feature` | API module and starter, migration, contract fragment, optional Web screens | `knowledge-base` |
| `theme` | Design token overrides for the Web application and the Admin Console; one theme at a time | `theme-enterprise` |
| `block` | A reusable page section under `web/src/components/blocks` | `block-pricing-table` |

## How A Plugin Touches The Project

- **Files** under `files/` land at the same repository paths. A file that already exists with other
  content stops the install before anything is written.
- **Slots** are the only way a plugin edits core files. A slot is a marker comment such as
  `// fastbuild:slot web.features`; the plugin's `wiring` lists lines to insert before it, and a
  `starter` spec expands into the `api.starter.*` slots. `plugin:slots` lists every slot.
- **Contract fragments** are merged into `contracts/openapi.yaml`, including `x-error-codes`.

Everything is applied in one step and rolled back on failure. The lock file records each file's
checksum, each inserted line, and each contract addition, which is what makes the lifecycle exact:

- `plugin:update` replaces files the project never changed, merges edited files three ways against
  the installed version (conflicts are left as `<<<<<<< local` markers; without the installed version
  the incoming file is saved as `<file>.fastbuild-new`), and re-applies changed wiring and contracts.
- `plugin:remove` deletes the files, removes the inserted lines, and removes the contract additions,
  restoring the core files byte for byte. Tables in existing databases are never dropped.
- `plugin:doctor` fails while a fragment is unmerged, a slot line is missing, a conflict is
  unresolved, or an incoming file waits to be merged.

## Layout

```text
registry/
  index.json                  generated catalog with a SHA-256 for every file (do not edit)
  plugins/<name>/
    plugin.json               manifest: identity, kind, compatibility, requirements, starter, wiring
    README.md                 what the plugin does, for people
    INTEGRATION.md            adaptive steps an agent performs after installing, each verifiable
    files/                    files laid out exactly as they land in the project
```

## Adding Or Changing A Plugin

1. Write or edit the plugin under `plugins/<name>/` and raise its `version` when users would notice.
2. Develop it installed: `plugin:add <name>`, finish its integration steps, edit the installed files,
   then `plugin:pack <name>` copies edited files back here (add new files by hand).
3. Regenerate the index: `cd api && go run ./cmd/fastbuild plugin:index`.
4. Prove it: `cd api && go test ./internal/infra/console/commands -run 'Plugin|Slots'` installs every
   plugin into a copy of the project, builds and tests it, and removes it again;
   `make contract-check` merges, lints, and removes every contract fragment.
5. `plugin:remove <name> --force` so the scaffold itself does not keep the plugin installed.

The design is described in [docs/plans/plugin-registry.md](../docs/plans/plugin-registry.md).
