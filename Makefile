SHELL := /bin/bash

.DEFAULT_GOAL := help

REGISTRY ?= base
TAG      ?= dev

# ---------- help ----------
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} \
	/^[a-zA-Z0-9_-]+:.*?##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---------- proto ----------
.PHONY: proto proto-go proto-ts proto-check
proto: proto-go proto-ts ## Regenerate all proto code (Go + TS)

proto-go: ## Generate Go from proto/
	buf generate

proto-ts: ## Generate TS from proto/ into each app's web/src/gen
	cd apps/webapp/web && pnpm run buf:generate

proto-check: ## Fail if generated code is out of date
	@buf format -d
	@buf lint
	@buf breaking --against '.git#branch=main' || true

# ---------- infra ----------
.PHONY: infra-up infra-down infra-logs
infra-up: ## docker-compose up postgres + nats
	docker compose up -d
	@echo "→ postgres :5432, nats :4222"

infra-down: ## stop infra
	docker compose down

infra-logs:
	docker compose logs -f

# ---------- dev ----------
.PHONY: dev dev-user dev-webapp-bff dev-webapp-web
dev: ## Run services + BFF + web concurrently (requires overmind)
	overmind start -f Procfile.dev

dev-user: ## Run only the user service
	cd services/user && BASE_USER_DB_AUTO_MIGRATE=true go run ./cmd/user

dev-webapp-bff: ## Run only the webapp BFF
	cd apps/webapp && go run ./cmd/webapp

dev-webapp-web: ## Run only the SPA dev server
	pnpm --filter @base/webapp dev

# ---------- migrate ----------
.PHONY: migrate-up migrate-down
MIGRATE_URL ?= postgres://base:base@localhost:5432/base_user?sslmode=disable
SVC ?= user
migrate-up: ## Apply migrations (SVC=user)
	migrate -database "$(MIGRATE_URL)" -path services/$(SVC)/migrations up

migrate-down: ## Roll back one migration (SVC=user)
	migrate -database "$(MIGRATE_URL)" -path services/$(SVC)/migrations down 1

# ---------- build / lint / test ----------
.PHONY: build test lint tidy fmt
build: ## Build all Go binaries to bin/
	@mkdir -p bin
	cd services/user && go build -o ../../bin/user ./cmd/user
	cd apps/webapp && go build -o ../../bin/webapp ./cmd/webapp

test: ## Run all Go tests
	cd services/user && go test ./...
	cd apps/webapp && go test ./...
	cd pkg/auth && go test ./...

lint: ## golangci-lint over all Go modules
	golangci-lint run ./...

tidy: ## Tidy every module
	for d in pkg/*/ services/*/ apps/*/ gen/; do \
		(cd $$d && go mod tidy) || exit 1; \
	done
	go work sync

fmt: ## gofmt + prettier
	gofmt -w .
	pnpm -r exec prettier -w "src/**/*.{ts,tsx,css}" 2>/dev/null || true

# ---------- images ----------
.PHONY: images images-services images-apis images-webs
images: ## Build all container images
	REGISTRY=$(REGISTRY) TAG=$(TAG) docker buildx bake

images-services:
	REGISTRY=$(REGISTRY) TAG=$(TAG) docker buildx bake services

images-apis:
	REGISTRY=$(REGISTRY) TAG=$(TAG) docker buildx bake apis

images-webs:
	REGISTRY=$(REGISTRY) TAG=$(TAG) docker buildx bake webs

# ---------- k8s ----------
.PHONY: k8s-dev k8s-prod
k8s-dev: ## kubectl apply dev overlay
	kubectl apply -k deploy/k8s/overlays/dev

k8s-prod: ## kubectl apply prod overlay
	kubectl apply -k deploy/k8s/overlays/prod

# ---------- k3d / argocd ----------
.PHONY: k3d-up k3d-down argocd-install argocd-ui argocd-pass
k3d-up: ## Create local k3d cluster with ingress-nginx
	bash scripts/k3d-up.sh

k3d-down: ## Delete local k3d cluster
	bash scripts/k3d-down.sh

argocd-install: ## Install Argo CD + dev Application (needs GITHUB_USERNAME, GITHUB_TOKEN)
	bash scripts/argocd-install.sh

argocd-ui: ## Port-forward Argo CD UI to https://localhost:8080
	kubectl -n argocd port-forward svc/argocd-server 8080:443

argocd-pass: ## Print Argo CD admin password
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d; echo
