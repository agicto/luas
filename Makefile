.PHONY: check governance api-check web-check

check: governance api-check web-check

governance:
	bash .agents/skills/luas-framework-review/scripts/check-vocabulary.sh
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-doc-links.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-error-contracts.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-auth-contract-boundary.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-config-authority.py
	PYTHONDONTWRITEBYTECODE=1 python3 .agents/skills/luas-framework-review/scripts/check-surface-catalog.py
	bash .agents/skills/luas-framework-review/scripts/check-api-boundaries.sh
	bash .agents/skills/luas-framework-review/scripts/check-branch-governance.sh
	bash .agents/skills/scripts/validate-skill.sh --all

api-check:
	cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1

web-check:
	cd web && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 2
