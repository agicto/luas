.PHONY: check api-check web-check

check: api-check web-check

api-check:
	cd api && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 1

web-check:
	cd web && bash ../.agents/skills/verification-before-completion/scripts/run-tiers.sh 2
