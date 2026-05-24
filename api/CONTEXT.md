# ZGO Context

ZGO is a Go application scaffold, not a framework-only kernel and not a product application.

## Layers

- `core`: reusable runtime and infrastructure used by every ZGO app. Examples: `internal/bootstrap`, `internal/infra`, `routes`, `pkg`.
- `starter`: default business-ready building blocks that ship with a new ZGO app. Current starters are `user`, `apikey`, and `audit`.
- `capability`: technical modules that expose reusable integrations or helpers without owning an application route model. Example: `internal/capabilities/ai`.
- `optional starter`: starter-quality modules that are not wired into a new app by default. Example: `permission`.
- `example`: code or docs whose main job is demonstration, not default assembly.

## Assembly Terms

- `starter registry`: the single module that decides which starters, starter migrations, and starter seeders are active in the scaffold.
- `command manifest`: the seam that groups related CLI commands so registration does not drift across `cmd/zgo` and command packages.
- `default scaffold`: the out-of-the-box ZGO app assembled from `core` plus the default `starter` set.

## Current Defaults

- `user`: auth starter for register, login, JWT auth, profile, password management.
- `apikey`: starter for machine access using hashed API keys and request middleware.
- `audit`: starter for global write-side audit logging and per-user audit history.
- `permission`: optional starter for RBAC; available in the repo but not part of the default scaffold.
