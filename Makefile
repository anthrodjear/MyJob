.PHONY: help setup migrate seed start stop clean build

# Default target
help:
	@echo "AI Job Search Agent - Commands:"
	@echo ""
	@echo "  Setup & Start:"
	@echo "  make setup       - First-time setup (create .env, start infra, run migrations)"
	@echo "  make start       - Start all services"
	@echo "  make stop        - Stop all services"
	@echo "  make build       - Build all Docker images"
	@echo "  make clean       - Remove containers and volumes"
	@echo ""
	@echo "  Database:"
	@echo "  make migrate     - Run database migrations"
	@echo "  make seed        - Seed initial data"
	@echo ""
	@echo "  Development:"
	@echo "  make dev-api     - Run Go API locally"
	@echo "  make dev-worker  - Run Go Worker locally"
	@echo "  make dev-frontend - Run Next.js frontend locally"
	@echo "  make dev-browser - Run Browser Agent locally"
	@echo ""
	@echo "  Build (local):"
	@echo "  make build-api   - Build Go API binary"
	@echo "  make build-worker - Build Go Worker binary"
	@echo "  make build-frontend - Build Next.js frontend"
	@echo "  make build-browser - Build Browser Agent"
	@echo ""
	@echo "  Logs:"
	@echo "  make logs        - View all service logs"
	@echo "  make logs-api    - View API logs"
	@echo "  make logs-worker - View Worker logs"
	@echo "  make logs-browser - View Browser Agent logs"
	@echo "  make logs-frontend - View Frontend logs"
	@echo ""
	@echo "  Shell Access:"
	@echo "  make shell-api       - Shell into API container"
	@echo "  make shell-worker    - Shell into Worker container"
	@echo "  make shell-browser   - Shell into Browser Agent container"
	@echo "  make shell-frontend  - Shell into Frontend container"
	@echo "  make shell-postgres  - PostgreSQL CLI"
	@echo "  make shell-redis     - Redis CLI"
	@echo ""
	@echo "  Testing:"
	@echo "  make test        - Run all tests"
	@echo "  make test-api    - Run Go API tests"
	@echo "  make test-frontend - Run Frontend tests"
	@echo ""
	@echo "  Utils:"
	@echo "  make hash-password PASSWORD=yourpass - Generate bcrypt hash for AUTH_PASSWORD_HASH"

# First-time setup
setup:
	@if not exist .env copy .env.example .env
	@echo "Created .env file. Please edit it with your API keys."
	docker compose up -d postgres redis ollama
	docker compose exec ollama ollama pull mxbai-embed-large
	docker compose exec ollama ollama pull qwen2.5:latest
	@echo "Setup complete! Run 'make migrate' to initialize the database."

# Run migrations
migrate:
	docker compose exec api ./migrate.sh

# Seed initial data
seed:
	docker compose exec api ./seed.sh

# Start all services
start:
	docker compose up -d

# Stop all services
stop:
	docker compose down

# Build Docker images
build:
	docker compose build

# Clean everything
clean:
	docker compose down -v
	@echo "Removed containers and volumes"

# ============================================
# Local Development
# ============================================

# Run Go API locally
dev-api:
	cd backend && go run ./cmd/api

# Run Go Worker locally
dev-worker:
	cd backend && go run ./cmd/worker

# Run Next.js frontend locally
dev-frontend:
	cd frontend && npm run dev

# Run Browser Agent locally
dev-browser:
	cd browser-agent && npm run dev

# ============================================
# Local Builds
# ============================================

# Build Go API binary
build-api:
	cd backend && go build -o ../bin/api.exe ./cmd/api

# Build Go Worker binary
build-worker:
	cd backend && go build -o ../bin/worker.exe ./cmd/worker

# Build Next.js frontend
build-frontend:
	cd frontend && npm run build

# Build Browser Agent
build-browser:
	cd browser-agent && npm run build

# Generate bcrypt password hash for AUTH_PASSWORD_HASH
# Usage: make hash-password PASSWORD="yourpassword"
hash-password:
	cd backend && go run ../scripts/hash_password.go "$(PASSWORD)"

# ============================================
# Logs
# ============================================

logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-worker:
	docker compose logs -f worker

logs-browser:
	docker compose logs -f browser-agent

logs-frontend:
	docker compose logs -f frontend

# ============================================
# Shell Access
# ============================================

shell-api:
	docker compose exec api sh

shell-worker:
	docker compose exec worker sh

shell-browser:
	docker compose exec browser-agent sh

shell-frontend:
	docker compose exec frontend sh

shell-postgres:
	docker compose exec postgres psql -U myjob -d myjob

shell-redis:
	docker compose exec redis redis-cli

# ============================================
# Testing
# ============================================

test:
	cd backend && go test ./...
	cd browser-agent && npm test
	cd frontend && npm test

test-api:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test

# ============================================
# CI / Lint
# ============================================

lint: lint-go lint-frontend lint-browser-agent

lint-go:
	cd backend && go vet ./...
	golangci-lint run --config ../.golangci.yml ./...

lint-frontend:
	cd frontend && npm run lint
	cd frontend && npx tsc --noEmit

lint-browser-agent:
	cd browser-agent && npx eslint src/ --max-warnings 0 || true
	cd browser-agent && npx tsc --noEmit

# Run full CI locally (lint + test + docker build)
ci-local: lint test build

# Docker compose CI mode (clean state, no host dependencies)
docker-ci:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml up -d --build

docker-ci-down:
	docker compose -f docker-compose.yml -f docker-compose.ci.yml down -v

# Health check
health-check:
	bash scripts/ci-health-check.sh

# ============================================
# Kubernetes / Helm
# ============================================

# Lint Helm chart
helm-lint:
	helm lint k8s/helm/myjob

# Dry-run Helm template (local rendering)
helm-template:
	helm template myjob k8s/helm/myjob

# Install/upgrade staging
helm-staging:
	helm upgrade --install myjob-staging k8s/helm/myjob \
		--namespace myjob-staging \
		--create-namespace \
		--values k8s/helm/myjob/values.yaml \
		--set api.image.tag=$(TAG) \
		--set worker.image.tag=$(TAG) \
		--set browserAgent.image.tag=$(TAG) \
		--set frontend.image.tag=$(TAG)

# Install/upgrade production
helm-production:
	helm upgrade --install myjob-production k8s/helm/myjob \
		--namespace myjob-production \
		--create-namespace \
		--values k8s/helm/myjob/values.yaml \
		--set api.image.tag=$(TAG) \
		--set worker.image.tag=$(TAG) \
		--set browserAgent.image.tag=$(TAG) \
		--set frontend.image.tag=$(TAG) \
		--set api.replicaCount=3 \
		--set worker.replicaCount=3 \
		--set frontend.replicaCount=3

# Rollback Helm release
helm-rollback:
	helm rollback $(RELEASE) -n $(NAMESPACE)

# Check HPA status
kubectl-hpa:
	kubectl get hpa -n $(NAMESPACE)

# Check all pods
kubectl-pods:
	kubectl get pods -n $(NAMESPACE) -o wide

# Tail logs
kubectl-logs:
	kubectl logs -n $(NAMESPACE) -l app.kubernetes.io/name=myjob --tail=100 -f

# Kustomize build
kustomize-build:
	kubectl kustomize k8s/kustomize/overlays/$(ENV)
