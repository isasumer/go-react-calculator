# Playwright suite against the composed stack (H3-04). Assumes the stack is already up —
# `docker compose -f compose.yaml -f compose.e2e.yaml up --build --wait` from the repo root — and
# reachable at $E2E_BASE_URL (default http://localhost:8080).

.PHONY: install
install: ## Install npm deps and the browsers the config needs (chromium + webkit)
	npm ci
	npx playwright install chromium webkit

.PHONY: test
test: ## Run the full Playwright suite (used by `make e2e` at the repo root)
	npm ci
	npx playwright test
