.PHONY: run run-debug install-gotestsum test test-report testdox clean install-lint lint lint-fix install-swag swagger build build-debug dockerfile-lint sonar-local e2e-workflows

GOTESTSUM := $(shell go env GOPATH)/bin/gotestsum
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
SWAG := $(shell go env GOPATH)/bin/swag

# Keep in sync with github.com/swaggo/swag in go.mod (CLI + runtime types).
SWAG_VERSION ?= v1.16.6

# Parse API packages explicitly. With multiple -d entries, -g is relative to the first dir (cmd/fuse).
SWAG_DIRS := ./cmd/fuse,./internal/handlers,./internal/dtos,./internal/workflow,./pkg/workflow

run:
	go build -o bin/fuse cmd/fuse/main.go
	./bin/fuse server -o -p 9090 -l debug

# Install gotestsum into GOPATH/bin (same path as GOTESTSUM)
install-gotestsum:
	go install gotest.tools/gotestsum@latest

test: install-gotestsum
	$(GOTESTSUM) --junitfile test-report.xml --format testdox -- ./pkg/... ./internal/... ./tests/e2e

test-benchmark:
	go test -bench=. -benchmem ./...

# Run workflow E2E against a running API (requires -tags=e2e; default http://localhost:9090 via E2E_API_URL).
# -parallel 1 avoids t.Parallel() disk/helper tests racing with workflow suites on one server.
# Unit tests for helpers: go test ./tests/e2e
e2e-workflows:
	docker compose --profile fuse-e2e -f docker-compose.e2e.yml up --build -d
	go test -tags=e2e ./tests/e2e -v -count=1 -timeout 15m
	docker compose --profile fuse-e2e -f docker-compose.e2e.yml down -v

build:
	go build -o bin/fuse cmd/fuse/main.go

build-debug:
	go build -tags pprof -o bin/fuse cmd/fuse/main.go

run-debug: build-debug
	./bin/fuse server -o -p 9090 -l debug

clean:
	rm -rf bin/

# golangci-lint must be built with Go >= the "go" line in go.mod (golangci-lint refuses older toolchains).
# GOTOOLCHAIN makes `go install` use that SDK even when the host `go` is older (e.g. gvm go1.24 + go.mod 1.26).
GOMOD_GOVER := $(shell sed -n 's/^go //p' go.mod | head -1 | tr -d ' \t\r')

# Install golangci-lint into GOPATH/bin (same path as GOLANGCI_LINT)
install-lint:
	GOTOOLCHAIN=go$(GOMOD_GOVER).0 go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

# Run linter
lint: install-lint
	$(GOLANGCI_LINT) run ./... --timeout=5m

# Run linter with auto-fix
lint-fix: install-lint
	$(GOLANGCI_LINT) run ./... --fix --timeout=5m

format:
	go fmt ./...

# Install swag CLI into GOPATH/bin (same path as SWAG)
install-swag:
	GOTOOLCHAIN=go$(GOMOD_GOVER).0 go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

# Generate Swagger documentation (OpenAPI 2.0 under docs/)
swagger: install-swag
	$(SWAG) init -g main.go -o docs/ -d $(SWAG_DIRS)

# Lint Dockerfiles (consecutive RUN merge, etc.). Mirrors Sonar Docker rules; requires Docker.
HADOLINT_IMAGE := hadolint/hadolint:2.12.0-alpine
dockerfile-lint:
	docker run --rm -v "$$(pwd):/work" -w /work --entrypoint /bin/hadolint $(HADOLINT_IMAGE) \
		--config /work/.hadolint.yaml Dockerfile Dockerfile.dev

# SonarCloud local analysis (reads ./sonar-project.properties). Requires Docker.
# SONAR_TOKEN: https://sonarcloud.io/account/security — use a SonarCloud user or org
# scoped token with analysis permission (not a GitHub OAuth token).
# If you see "Error 404 on .../analysis/analyses", try SONAR_REGION=us for US-hosted orgs.
# Optional: SONAR_SCANNER_IMAGE=sonarsource/sonar-scanner-cli:latest
SONAR_SCANNER_IMAGE ?= sonarsource/sonar-scanner-cli:latest

sonar-local:
	@test -n "$$SONAR_TOKEN" || (echo "Set SONAR_TOKEN (SonarCloud > My Account > Security)." >&2; exit 1)
	@set -e; \
	opts=""; \
	b=$$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true); \
	if [ -n "$$b" ] && [ "$$b" != "HEAD" ]; then \
		opts="$$opts -Dsonar.branch.name=$$b"; \
	fi; \
	if [ -n "$$SONAR_REGION" ]; then \
		opts="$$opts -Dsonar.region=$$SONAR_REGION"; \
	fi; \
	rev=$$(git rev-parse HEAD 2>/dev/null || true); \
	if [ -n "$$rev" ]; then \
		opts="$$opts -Dsonar.scm.revision=$$rev"; \
	fi; \
	docker run --rm \
		-e SONAR_TOKEN \
		-v "$$(pwd):/usr/src" \
		-w /usr/src \
		$(SONAR_SCANNER_IMAGE) \
		$$opts

dkb:
	docker build -t fuse-app:dev .

dkx:
	docker stop fuse-local
	docker rm fuse-local
	docker run --name fuse-local --env-file .env -p 9090:9090 fuse-app:dev
