# Security Policy

Security is a project-wide concern across the API, Web app, contracts, dependencies, containers, and development tooling.

## Supported Versions

Security fixes target the latest tagged release and the current `main` branch. Older releases may receive a fix when practical, but users should plan to upgrade to the latest release.

## Report A Vulnerability

Do not open a public issue, discussion, or pull request for a suspected vulnerability.

Use the private vulnerability-reporting facility provided by the project's hosting platform. If that facility is unavailable, contact a maintainer through a private channel published by the project before sharing technical details publicly.

Include enough information to reproduce and assess the report:

- affected version or commit;
- affected API route, Web route, command, image, or deployment mode;
- prerequisites and a minimal reproduction;
- expected and observed behavior;
- potential confidentiality, integrity, or availability impact;
- proof-of-concept material, logs, or traces with secrets and personal data removed;
- any known workaround.

Reports will be triaged for reproducibility, severity, affected versions, and coordinated remediation. Please allow maintainers time to investigate and publish a fix before public disclosure.

## Security Boundaries

Luas provides secure defaults and executable checks, but a downstream deployment still owns:

- secrets and key rotation;
- TLS, ingress, proxy trust, and network policy;
- database security, backups, retention, and disaster recovery;
- cloud and provider identities;
- dependency and image update cadence;
- monitoring, incident response, and disclosure obligations;
- product-specific authorization and data-classification policy.

Read [docs/DEPENDENCY_SECURITY.md](docs/DEPENDENCY_SECURITY.md), [docs/CONTAINER_SECURITY.md](docs/CONTAINER_SECURITY.md), [api/docs/OBSERVABILITY.md](api/docs/OBSERVABILITY.md), and [web/docs/SECURITY.md](web/docs/SECURITY.md) before production deployment.
