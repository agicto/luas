# FastBuild Roadmap

This is the long-term plan: what we build, in which order, and how we know each stage is done.

| Document | Answers |
|---|---|
| [VISION.md](VISION.md) | Why FastBuild exists and the principles every stage keeps |
| This roadmap | What ships in which stage, and the exit criteria |
| [plans/](plans/) | How one capability is built, per [BUSINESS_IMPLEMENTATION_PLAN_STANDARD.md](BUSINESS_IMPLEMENTATION_PLAN_STANDARD.md) |
| [FRAMEWORK_QUALITY_ROADMAP.md](FRAMEWORK_QUALITY_ROADMAP.md) | The running quality, security, and performance backlog |
| [STARTER_BUSINESS_ROADMAP.md](STARTER_BUSINESS_ROADMAP.md) | Starter readiness and starter ownership |

Dates are targets, not promises. A stage ends when its exit criteria pass, not when its date arrives.
Review this roadmap at every minor release and move finished items into the changelog.

## Where We Are (October 2026)

- **Foundation (done, v0.1–v0.24):** contract-first API, Web, and Admin Console; revocable sessions,
  RBAC, audit, organizations, notifications, assets, settings, usage, webhooks, operators; security
  defaults, performance budgets, container and supply-chain checks, release automation.
- **Plugin registry phase 1 (in review):** `plugin:search|info|add|status`, checksummed index, lock
  file, module-path adaptation, atomic install, the `knowledge-base` plugin, and the
  `plugin-integration` skill. See [plans/plugin-registry.md](plans/plugin-registry.md).

## Ordering Rules

1. **The core stays light.** A capability enters the core only when nearly every project needs it,
   or when it is security-critical (sessions, cryptography, permission checks). Everything else is
   a starter or a plugin.
2. **Agents first, people always.** Every new asset ships with machine-readable metadata, an
   `INTEGRATION.md`, and checks, so a coding agent can choose, install, and verify it unaided.
3. **Proof before scale.** One plugin, theme, or template proves a format before we build ten.
4. **Every stage has a checkable exit.** Exit criteria are tests, checks, or measured numbers.
5. **No runtime plugin loading.** Plugins are source the project owns. We do not build a hook system
   that loads unreviewed code at runtime.

## Stage 1 — Plugin Registry Foundation (v0.25, Q4 2026)

| Deliverable | Status |
|---|---|
| Plugin manifest, registry index with SHA-256 checksums, `plugin:index --check` | Done |
| `plugin:search`, `plugin:info`, `plugin:add` (dry run, rollback), `plugin:status`, lock file | Done |
| Remote source `github:fastagi/fastbuild@<version>` for initialized projects | Done |
| `knowledge-base` plugin with PostgreSQL full-text and substring search | Done |
| Tier 1 installs, builds, and tests every official plugin; contract check lints every fragment | Done |
| `plugin-integration` skill and routing evals | Done |

**Exit:** PR merged, v0.25.0 released, and a project created with `make init-project` installs
`knowledge-base` from the release and passes `make check`.

## Stage 2 — Full-Stack Plugins and Plugin Lifecycle (v0.26–v0.27, Q4 2026–Q1 2027)

A plugin must deliver a working feature end to end, and must be upgradable and removable.

| Deliverable | Notes |
|---|---|
| Web and Admin registry items | Pages, hooks, and services as source, in a shadcn-compatible item format; strict services typed from the contract |
| `knowledge-base` UI | List, editor, and search pages for Web; read-only view for Admin |
| `plugin:update` | Three-way merge using the install checksums in the lock file; untouched files update, edited files get a reviewable diff |
| `plugin:remove` | Reverses starter wiring and the contract fragment, keeps migrations and data by default |
| `plugin:doctor` | Reports missing integration steps: starter not enabled, golden schema stale, types not generated |
| Plugin authoring guide | How to write, test, and submit a plugin, with a `make:plugin` generator |

**Exit:** installing `knowledge-base` yields working API, Web, and Admin screens with no manual code;
an update test edits one installed file, updates the plugin, and keeps the edit; removal leaves
`make check` green.

## Stage 3 — Themes and Blocks (v0.28, Q1 2027)

| Deliverable | Notes |
|---|---|
| Theme format | Design tokens shared by Web and Admin: color, typography, radius, density, dark mode |
| 3 official themes | A neutral default, an enterprise theme, and a content-first theme |
| 8 official blocks | Dashboard, data table, settings page, pricing table, empty and error states, auth pages, onboarding, landing hero |
| One install command | `fastbuild add <name>` resolves plugins, themes, and blocks from the same registry |
| Visual checks | Screenshot tests in light and dark mode at phone and desktop widths; accessibility lint |

**Exit:** an agent can switch the theme and add three blocks to a fresh project in one session, and
the screenshot and accessibility checks pass.

## Stage 4 — Official Plugin Set (v0.29–v0.3x, Q1–Q2 2027)

Each plugin is product-neutral, optional, and must pass the same install, build, test, and contract
checks as `knowledge-base`. Candidates, in priority order:

| Plugin | Purpose |
|---|---|
| `ai-gateway` | Model provider adapters, streaming chat, per-organization cost limits on top of `usage` |
| `knowledge-vector` | pgvector embeddings and hybrid search for `knowledge-base`, enabling retrieval for AI answers |
| `billing-stripe` | Plans, subscriptions, and webhooks mapped to `usage` quotas |
| `comments` | Threaded comments on any resource, with mentions through `notification` |
| `tags` | Organization-scoped labels for any resource |
| `forms` | Configurable forms and submissions with export |
| `pages` | Simple content pages and navigation for marketing sites |
| `import-export` | Bounded CSV import and export jobs on the workflow capability |

**Exit:** at least six official plugins, every one installable together in one project with
`make check` green, and a nightly job that installs each plugin on `main`.

## Stage 5 — v1.0 Enterprise Ready (Q2–Q3 2027)

| Deliverable | Notes |
|---|---|
| OIDC single sign-on | Core, not a plugin: login through enterprise identity providers |
| Multi-factor authentication | TOTP and recovery codes for users and operators |
| Shared rate limiting | A PostgreSQL or Redis adapter for multi-replica deployments |
| Stable API promise | Versioning and deprecation policy for HTTP contracts, CLI commands, the plugin manifest, and the lock file |
| `fastbuild upgrade` | Compares a project with the release it started from and applies scaffold updates as reviewable diffs |
| Dependency currency | Current GORM, actions, and Go dependency groups with no held-back majors without a written reason |
| External security review | Third-party review of sessions, permissions, and the plugin installer |
| Documentation site | Quickstart, guides, and a searchable plugin catalog |

**Exit:** three downstream projects running in production, an upgrade from v0.25 to v1.0 performed
with `fastbuild upgrade`, zero known high-severity vulnerabilities, and a published compatibility
promise.

## Stage 6 — Templates and Agent-Native Building (v1.x, H2 2027)

| Deliverable | Notes |
|---|---|
| Templates | Complete starting points: knowledge portal, SaaS application, internal tool; each is core plus plugins, theme, and sample data |
| `fastbuild new --template` | One command from template to running project |
| Describe-to-build flow | The agent turns a business description into a plan, selects plugins and blocks, installs, adapts, and verifies them |
| Agent evaluation suite | Scripted tasks measure how often agents choose the right plugin and finish integration with checks passing |
| Machine interface | Registry search, info, and status as JSON and as an MCP server, so any coding agent can use them |

**Exit:** for each template, an agent goes from a one-paragraph description to a deployed, passing
project in under one day; the evaluation suite reports at least 90% correct plugin selection.

## Stage 7 — Community Ecosystem (v1.x and later)

| Deliverable | Notes |
|---|---|
| Third-party registries | `--source=github:<owner>/<repo>@<version>` with the same checksum and archive rules |
| Signed releases | Signed registry indexes and provenance, verified before install |
| Trust levels | Official, verified, and community labels, with clear review rules for each |
| Compatibility matrix | Public results of each listed plugin against each FastBuild release |
| Admin plugin page | A read-only Admin Console view of installed plugins, versions, and local changes |
| Contribution program | Plugin submission review, maintainers, and recognition |

**Exit:** community plugins installed in real projects, and every listed plugin shows a current
compatibility result.

## Continuous Tracks

These run in every stage and are tracked in [FRAMEWORK_QUALITY_ROADMAP.md](FRAMEWORK_QUALITY_ROADMAP.md):

- **Security:** zero known high-severity vulnerabilities per release; dependency and container scans.
- **Performance:** nightly budgets for API latency, bundle size, and memory; regressions block release.
- **Developer experience:** clone to running locally in under 15 minutes; clear errors from every CLI command.
- **Agent guidance:** skills, `AGENTS.md`, and routing evals updated with every new asset type.

## Measures of Progress

| Measure | v0.25 | v1.0 | v1.x |
|---|---|---|---|
| Official plugins passing nightly install checks | 1 | 6+ | 15+ |
| Themes and blocks | 0 | 3 and 8 | 5 and 20+ |
| Downstream projects in production | — | 3+ | 20+ |
| Clone to production deployment | days | under 1 day | under 1 day, from a template |
| Agent plugin-selection accuracy (eval suite) | — | measured | 90%+ |

## Not Planned

- Runtime plugin loading, binary plugin packages, or a hook system that runs unreviewed code.
- A hosted FastBuild service before v1.0.
- Database targets other than PostgreSQL.
- Product-specific behavior in the core; it belongs in plugins or downstream projects.
