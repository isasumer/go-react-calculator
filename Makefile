# Umbrella targets. Each delegates to backend/ and frontend/ once those tickets land (P0-02, P0-03).
# Until then, targets fail fast with a message naming the ticket that provides them.

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

BACKEND_DIR  := backend
FRONTEND_DIR := frontend
E2E_DIR      := e2e

# $(call need,<dir>,<ticket>) — abort with a clear message when a sub-project is not there yet.
define need
	@test -d $(1) || { echo "error: $(1)/ does not exist yet — it is delivered by ticket $(2)"; exit 1; }
endef

.PHONY: help
help: ## List targets
	@awk 'BEGIN{FS=":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/{printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: check
check: ## Run every gate: lint, vet, type-check, tests, coverage thresholds (both apps)
	$(call need,$(BACKEND_DIR),P0-02)
	$(call need,$(FRONTEND_DIR),P0-03)
	$(MAKE) -C $(BACKEND_DIR) check
	$(MAKE) -C $(FRONTEND_DIR) check

.PHONY: lint
lint: ## Lint both apps
	$(call need,$(BACKEND_DIR),P0-02)
	$(call need,$(FRONTEND_DIR),P0-03)
	$(MAKE) -C $(BACKEND_DIR) lint
	$(MAKE) -C $(FRONTEND_DIR) lint

.PHONY: test
test: ## Unit tests with coverage for both apps
	$(call need,$(BACKEND_DIR),P0-02)
	$(call need,$(FRONTEND_DIR),P0-03)
	$(MAKE) -C $(BACKEND_DIR) test
	$(MAKE) -C $(FRONTEND_DIR) test

.PHONY: dev
dev: ## Run backend and frontend dev servers (Ctrl-C stops both)
	$(call need,$(BACKEND_DIR),P0-02)
	$(call need,$(FRONTEND_DIR),P0-03)
	@trap 'kill 0' INT TERM; \
	  $(MAKE) -C $(BACKEND_DIR) run & \
	  $(MAKE) -C $(FRONTEND_DIR) dev & \
	  wait

.PHONY: up
up: ## Build and start the full stack with Docker Compose (H3-03)
	@test -f compose.yaml || { echo "error: compose.yaml does not exist yet — it is delivered by ticket H3-03"; exit 1; }
	docker compose up --build --wait

.PHONY: down
down: ## Stop the compose stack
	@test -f compose.yaml || { echo "error: compose.yaml does not exist yet — it is delivered by ticket H3-03"; exit 1; }
	docker compose down --remove-orphans

.PHONY: e2e
e2e: ## Run Playwright end-to-end tests against the compose stack (H3-04)
	$(call need,$(E2E_DIR),H3-04)
	$(MAKE) -C $(E2E_DIR) test
