.PHONY: up down migrate seed api web test tidy

up:
	docker compose up -d
	@echo "waiting for postgres..."
	@until docker compose exec -T postgres pg_isready -U localdiscovery -d localdiscovery >/dev/null 2>&1; do sleep 1; done
	@echo "postgres is ready"

down:
	docker compose down

migrate:
	cd api && go run ./cmd/seed -migrate-only

seed:
	cd api && go run ./cmd/seed

api:
	cd api && go run ./cmd/api

web:
	cd web && npm run dev

test:
	cd api && go test ./...
	cd web && npx tsc --noEmit

tidy:
	cd api && go mod tidy
