# FastBuild Vision

## Vision

Every team can build a production-grade business application as fast as it can describe one.

## Mission

FastBuild gives teams, and the coding agents working with them, a secure, fast, contract-first
foundation with the hard parts already done: authentication, permissions, audit, deployment, and CI.
People spend their time on the business; agents assemble proven building blocks instead of
improvising them, and every change passes the same production-grade checks.

## Who It Is For

- **Builders working with coding agents** who describe a business and need it to run in production,
  not only in a demo.
- **Enterprise teams** that want a reviewed starting point instead of rebuilding authentication,
  permissions, and audit for every project.
- **Startups** that need to ship quickly without accumulating security and architecture debt.

## Principles

1. **Fast by default** — fast to start (`make init-project`, `make dev`) and fast at runtime, with
   performance budgets enforced nightly.
2. **Secure by default** — the safe behavior is the default; risky behavior is an explicit opt-in.
3. **Contracts first** — the OpenAPI contract is the single source of truth, and both browser shells
   are type-checked against it.
4. **A foundation, not a product** — the core holds what every project needs; everything else is an
   optional starter or a plugin, and product behavior belongs to the projects built on it.
5. **Built for people and agents together** — rules are written down for agents (`AGENTS.md`,
   skills) and enforced by machine-checkable guards, so an agent's mistake fails a check instead of
   reaching production.

## Building Blocks

FastBuild grows through reusable assets that a coding agent can find, install, and adapt:

| Level | What it is | Example |
|---|---|---|
| Theme | Design tokens for the Web application and Admin Console | Color, typography, radius |
| Block | A reusable page section | Pricing table, dashboard, settings page |
| Plugin | A self-contained capability: API module, migrations, contract, permissions, settings, UI | Knowledge base, payments, AI model adapter |
| Template | A complete application: core, plugins, theme, and sample data | Knowledge portal, SaaS, internal tool |

Plugins are integrated as source, not as opaque packages: the code lands in the project, the
project owns it, and the agent can adapt it to the business. Heavy engines and third-party SDKs stay
ordinary dependencies, and the security core (sessions, cryptography, permission checks) stays in
FastBuild and updates through releases. A lock file records where every installed plugin came from,
so updates can be compared and merged instead of lost.

Three layers work together:

```text
Skills    (knowledge)   the agent knows the best practice and which building block fits
Registry  (assets)      the agent installs proven, versioned source instead of improvising it
Checks    (guardrails)  every change is verified before it ships
```

## Milestones

The stages, deliverables, and exit criteria are in [ROADMAP.md](ROADMAP.md).

| Stage | Goal | Key work |
|---|---|---|
| v0.x | A solid foundation | Contracts, security, performance, deployment, release automation (done) |
| v0.25+ | Plugin registry | Plugin manifest, `plugin:add`, lock file, agent skill, first official plugins |
| v1.0 | Enterprise ready | OIDC single sign-on, MFA, shared rate limiting, a stable API promise, first production downstream projects |
| v1.x | Agent-native building | Themes, blocks, and templates; describe a capability and an agent assembles, adapts, and verifies it |

## How We Measure Progress

- Time from `git clone` to a running production deployment: under one day.
- Downstream projects running in production.
- Zero known high-severity vulnerabilities in each release.
- Official and community plugins that install and pass the checks on the current release.
