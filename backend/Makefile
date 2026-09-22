# Backend targets. Tools are pinned in tools/go.mod and built into ./bin by `make tools`.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

BIN          := $(CURDIR)/bin
GOLANGCI     := $(BIN)/golangci-lint
GOFUMPT      := $(BIN)/gofumpt
GOVULNCHECK  := $(BIN)/govulncheck
TOOLS        := $(GOLANGCI) $(GOFUMPT) $(GOVULNCHECK)

COVERPROFILE := coverage.out
COVER_MIN    := 85

# Build identity stamped into internal/observability (see buildinfo.go). When
# these are not set, the package falls back to the embedded VCS stamps.
BUILDINFO := github.com/isasumer/go-react-calculator/backend/internal/observability
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X $(BUILDINFO).Version=$(VERSION) -X $(BUILDINFO).Commit=$(COMMIT) -X $(BUILDINFO).BuildDate=$(DATE)

.PHONY: help
help: ## List targets
	@awk 'BEGIN{FS=":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/{printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tools
tools: $(TOOLS) ## Build pinned dev tools into ./bin

$(TOOLS): tools/go.mod tools/go.sum
	cd ./tools && GOBIN=$(BIN) go install tool
	@touch $(TOOLS)

.PHONY: fmt
fmt: $(GOFUMPT) ## Format code with gofumpt
	$(GOFUMPT) -w .

.PHONY: fmt-check
fmt-check: $(GOFUMPT) ## Fail if any file is not gofumpt-formatted
	@out=$$($(GOFUMPT) -l .); if [[ -n $$out ]]; then echo "not gofumpt-formatted:"; echo "$$out"; exit 1; fi

.PHONY: lint
lint: $(GOLANGCI) ## Run golangci-lint
	$(GOLANGCI) run ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Unit tests with race detector, shuffled order and coverage profile
	go test -race -shuffle=on -coverprofile=$(COVERPROFILE) ./...

.PHONY: cover
cover: $(COVERPROFILE) ## Print per-function coverage summary
	go tool cover -func=$(COVERPROFILE)

$(COVERPROFILE):
	$(MAKE) test

.PHONY: coverage-check
coverage-check: $(COVERPROFILE) ## Fail if total coverage is below $(COVER_MIN)%
	scripts/coverage-check.sh $(COVERPROFILE) $(COVER_MIN)

.PHONY: vuln
vuln: $(GOVULNCHECK) ## Scan dependencies and stdlib for known vulnerabilities
	$(GOVULNCHECK) ./...

.PHONY: build
build: ## Build the server binary into ./bin/server with version info
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN)/server ./cmd/server

.PHONY: run
run: ## Run the server (PORT=9000 make run to change the port)
	go run ./cmd/server

.PHONY: check
check: fmt-check lint vet test coverage-check ## Every gate: format, lint, vet, tests, coverage threshold

.PHONY: clean
clean: ## Remove build output and coverage files
	rm -rf $(BIN) $(COVERPROFILE)
