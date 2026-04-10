.PHONY: dev dev-docker

# Live reload with Air (install once: go install github.com/air-verse/air@latest)
dev:
	@command -v air >/dev/null 2>&1 || { printf '%s\n' "Install Air: go install github.com/air-verse/air@latest"; exit 1; }
	air

# Postgres + web with Air inside Docker (rebuilds on .go / template changes)
dev-docker:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
