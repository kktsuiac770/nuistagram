.PHONY: dev backend frontend build

dev: ## Start backend and frontend concurrently
	@mkdir -p backend/data backend/static/uploads
	@trap 'kill 0' SIGINT; \
	(cd backend && go run ./cmd/server) & \
	(cd frontend && npm run dev) & \
	wait

backend: ## Start the backend server
	@mkdir -p backend/data backend/static/uploads
	cd backend && go run ./cmd/server

frontend: ## Start the frontend dev server
	cd frontend && npm run dev

build: ## Build backend binary and frontend bundle
	cd backend && go build -o nuistagram ./cmd/server
	cd frontend && npm run build
