# Frontend targets. Each mirrors an npm script so CI, the root Makefile and humans run the same commands.

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

NPM ?= npm

.PHONY: help
help: ## List targets
	@awk 'BEGIN{FS=":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/{printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

node_modules: package.json package-lock.json
	$(NPM) ci
	@touch node_modules

.PHONY: install
install: node_modules ## Install dependencies with npm ci

.PHONY: dev
dev: node_modules ## Start the Vite dev server
	$(NPM) run dev

.PHONY: build
build: node_modules ## Type-check and build to dist/
	$(NPM) run build

.PHONY: format
format: node_modules ## Rewrite files with Prettier
	$(NPM) run format

.PHONY: format-check
format-check: node_modules ## Fail if any file is not Prettier-formatted
	$(NPM) run format:check

.PHONY: lint
lint: node_modules ## ESLint with zero warnings allowed
	$(NPM) run lint

.PHONY: type-check
type-check: node_modules ## TypeScript project build without emit
	$(NPM) run type-check

.PHONY: test
test: node_modules ## Unit tests with coverage and threshold gate
	$(NPM) run test:coverage

.PHONY: check
check: format-check lint type-check test ## Every gate, in order: format, lint, type-check, tests + coverage
