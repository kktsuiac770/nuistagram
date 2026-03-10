.PHONY: dev backend frontend build test test-backend test-frontend

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

test: test-backend test-frontend ## Run all tests

test-backend: ## Run backend tests with race detector and coverage
	cd backend && go test -v ./... -race -coverprofile=coverage.out

test-frontend: ## Run frontend tests
	cd frontend && npm run test:run

# ─── Kubernetes / KinD ────────────────────────────────────────────────────────

CLUSTER_NAME     ?= nuistagram
BACKEND_IMAGE    ?= nuistagram-backend
FRONTEND_IMAGE   ?= nuistagram-frontend
IMAGE_TAG        ?= latest

.PHONY: kind-create kind-ingress kind-load kind-deploy kind-up kind-delete

kind-create: ## Create the KinD cluster
	kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml

kind-ingress: ## Install nginx ingress controller and wait for it to be ready
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.0/deploy/static/provider/kind/deploy.yaml
	kubectl wait --namespace ingress-nginx \
	  --for=condition=ready pod \
	  --selector=app.kubernetes.io/component=controller \
	  --timeout=120s

kind-load: ## Build both Docker images and load them into the KinD cluster
	docker build -t $(BACKEND_IMAGE):$(IMAGE_TAG) -f Dockerfile .
	docker build -t $(FRONTEND_IMAGE):$(IMAGE_TAG) -f Dockerfile.frontend .
	kind load docker-image $(BACKEND_IMAGE):$(IMAGE_TAG) --name $(CLUSTER_NAME)
	kind load docker-image $(FRONTEND_IMAGE):$(IMAGE_TAG) --name $(CLUSTER_NAME)

kind-deploy: ## Apply all Kubernetes manifests
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/frontend-deployment.yaml
	kubectl apply -f k8s/frontend-service.yaml
	kubectl apply -f k8s/ingress.yaml

kind-up: kind-create kind-ingress kind-load kind-deploy ## Full cluster bring-up (create → ingress → images → deploy)

kind-delete: ## Tear down the KinD cluster entirely
	kind delete cluster --name $(CLUSTER_NAME)
