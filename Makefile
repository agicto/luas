.PHONY: check agent-check governance api-check web-check web-spa-check dependency-scan sbom container-scan container-sbom

check: governance api-check web-check web-spa-check

agent-check:
	@bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh
	@PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py
	@PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-english-source.py
	@bash .agents/skills/scripts/validate-skill.sh --all
	@git diff --check

governance: agent-check
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-route-contract-discovery.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-auth-contract-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-rate-limit-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-cache-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-database-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-web-performance-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-web-security-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-web-ui-primitive-boundary.py
	node web-spa/scripts/check-architecture.mjs
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-api-key-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-audit-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-permission-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-notification-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-asset-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-setting-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-usage-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-webhook-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-ai-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-sensitive-telemetry.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-config-authority.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-email-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-ci-actions.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-dependency-supply-chain.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-container-supply-chain.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-surface-catalog.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-starter-catalog.py
	bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh
	bash .agents/skills/luas-framework-review/scripts/check-branch-governance.sh

api-check:
	cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1
	cd api && make route-catalog-check

web-check:
	cd web && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 2

web-spa-check:
	cd web-spa && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 2

dependency-scan:
	bash scripts/dependency-security.sh scan

sbom:
	bash scripts/dependency-security.sh sbom "$${SBOM_OUTPUT:-$${TMPDIR:-/tmp}/luas.cdx.json}"

container-scan:
	@test -n "$${IMAGE:-}" || { echo "IMAGE is required (for example, IMAGE=luas-api:container-check)" >&2; exit 2; }
	bash scripts/container-security.sh scan "$${IMAGE}"

container-sbom:
	@test -n "$${IMAGE:-}" || { echo "IMAGE is required (for example, IMAGE=luas-api:container-check)" >&2; exit 2; }
	bash scripts/container-security.sh sbom "$${IMAGE}" "$${SBOM_OUTPUT:-$${TMPDIR:-/tmp}/luas-container.cdx.json}"
