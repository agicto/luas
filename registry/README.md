# FastBuild Plugin Registry

Official FastBuild plugins: versioned capabilities distributed as source. A plugin is installed into
a project with one command, and from then on the project owns the code.

```bash
cd api
go run ./cmd/fastbuild plugin:search knowledge
go run ./cmd/fastbuild plugin:info knowledge-base
go run ./cmd/fastbuild plugin:add knowledge-base
```

A project created with `make init-project` installs plugins from the FastBuild release it started
from; this directory is not copied into new projects.

## Layout

```text
registry/
  index.json                  generated catalog with a SHA-256 for every file (do not edit)
  plugins/<name>/
    plugin.json               manifest: identity, compatibility, requirements, starter, files
    README.md                 what the plugin does, for people
    INTEGRATION.md            adaptive steps an agent performs after installing, each verifiable
    files/                    files laid out exactly as they land in the project
```

## Adding Or Changing A Plugin

1. Edit the plugin under `plugins/<name>/` and raise its `version`.
2. Regenerate the index: `cd api && go run ./cmd/fastbuild plugin:index`.
3. Run `cd api && go test ./internal/infra/console/commands -run Plugin` — every plugin is installed
   into a copy of the API, built, and tested.

The manifest format and the install workflow are described in
[docs/plans/plugin-registry.md](../docs/plans/plugin-registry.md).
