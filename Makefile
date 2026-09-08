.PHONY: dev dev-backend dev-frontend db-up db-down build test clean

db-up:
	docker compose up -d postgresdb redis

db-down:
	docker compose down

dev-backend: db-up
	cd backend && go run cmd/server/main.go

dev-frontend:
	cd frontend && pnpm dev

dev:
	mise run dev

build:
	cd backend && go build -o bin/server cmd/server/main.go
	cd frontend && pnpm build

test:
	cd backend && go test ./...
	cd frontend && pnpm check

clean:
	rm -rf backend/bin frontend/build frontend/.svelte-kit
