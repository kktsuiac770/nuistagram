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
	sleep 10
	kubectl wait --namespace ingress-nginx \
	  --for=condition=ready pod \
	  --selector=app.kubernetes.io/component=controller \
	  --timeout=120s

kind-load: ## Build both Docker images and load them into the KinD cluster
	docker build -t $(BACKEND_IMAGE):$(IMAGE_TAG) -f Dockerfile .
	docker build -t $(FRONTEND_IMAGE):$(IMAGE_TAG) -f Dockerfile.frontend .
	kind load docker-image $(BACKEND_IMAGE):$(IMAGE_TAG) --name $(CLUSTER_NAME)
	kind load docker-image $(FRONTEND_IMAGE):$(IMAGE_TAG) --name $(CLUSTER_NAME)

OVERLAY          ?= local

kind-deploy: ## Apply Kubernetes manifests via Kustomize (OVERLAY=local|production)
	kubectl apply -k k8s/overlays/$(OVERLAY)

kind-up: kind-create kind-ingress kind-load kind-deploy ## Full cluster bring-up (create → ingress → images → deploy)

kind-delete: ## Tear down the KinD cluster entirely
	kind delete cluster --name $(CLUSTER_NAME)

# ─── Prometheus Operator / monitoring ─────────────────────────────────────────

PROMETHEUS_CHART_VERSION ?= 65.4.0

.PHONY: kind-prometheus kind-prometheus-delete kind-prometheus-port-forward

kind-prometheus: ## Install kube-prometheus-stack and apply the ServiceMonitor
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
	helm repo update
	kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
	kubectl label namespace monitoring kubernetes.io/metadata.name=monitoring --overwrite
	helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
	  --namespace monitoring \
	  --version $(PROMETHEUS_CHART_VERSION) \
	  --values k8s/monitoring/prometheus-values.yaml \
	  --wait --timeout 5m
	kubectl apply -k k8s/monitoring

kind-prometheus-port-forward: ## Forward Prometheus UI to localhost:9090 and Grafana to localhost:3000
	@echo "Prometheus → http://localhost:9090"
	@echo "Grafana    → http://localhost:3000  (admin / admin)"
	@kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090 &
	@kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3000:80

kind-prometheus-delete: ## Remove kube-prometheus-stack and its CRDs
	kubectl delete -k k8s/monitoring --ignore-not-found
	helm uninstall kube-prometheus-stack --namespace monitoring --ignore-not-found
	kubectl delete namespace monitoring --ignore-not-found
