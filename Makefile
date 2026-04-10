.PHONY: dev dev-docker

# Live reload with Air (install once: go install github.com/air-verse/air@latest).
# GOTARO_LIVE_TEMPLATES: edit .html, refresh browser — no wait for rebuild. Air still rebuilds on .go changes.
dev:
	@command -v air >/dev/null 2>&1 || { printf '%s\n' "Install Air: go install github.com/air-verse/air@latest"; exit 1; }
	GOTARO_LIVE_TEMPLATES=1 air

# Postgres + web with Air inside Docker (rebuilds on .go / template changes)
dev-docker:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
