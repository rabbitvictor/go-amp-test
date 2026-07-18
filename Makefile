# go-amp-test — common dev commands.
# Run `make help` to list targets with descriptions.

SHELL := /usr/bin/env bash

# Project paths.
BINDIR   := bin
SERVER   := $(BINDIR)/server
CLI      := $(BINDIR)/go-amp-test

# Go build flags — keep CGO disabled (modernc.org/sqlite is pure Go).
CGO_ENABLED ?= 0
GO_BUILD_FLAGS := -trimpath -ldflags="-s -w"

# Docker image tag, overridable from the environment.
IMAGE_TAG ?= go-amp-test

# Smoke script env overrides (see scripts/cli_smoke.sh).
SMOKE_ENV ?=

.PHONY: help
help: ## show this list of targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## build all packages (go build ./...)
	CGO_ENABLED=$(CGO_ENABLED) go build ./...

.PHONY: build-server
build-server: ## build ./cmd/server into ./bin/server
	@mkdir -p $(BINDIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(SERVER) ./cmd/server

.PHONY: build-cli
build-cli: ## build ./cmd/cli into ./bin/go-amp-test
	@mkdir -p $(BINDIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(CLI) ./cmd/cli

.PHONY: run
run: ## run the server (go run ./cmd/server)
	go run ./cmd/server

.PHONY: cli-health
cli-health: ## run `go run ./cmd/cli health` against a local server
	go run ./cmd/cli health

.PHONY: vet
vet: ## go vet ./...
	go vet ./...

.PHONY: fmt
fmt: ## apply gofmt -s to the whole tree
	gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## fail if any file is not gofmt-clean (CI-friendly)
	@out=$$(gofmt -l -s .); if [ -n "$$out" ]; then \
		echo "gofmt would reformat:"; echo "$$out"; exit 1; \
	else echo "gofmt: clean"; fi

.PHONY: test
test: ## go test ./...
	go test ./...

.PHONY: check
check: build vet fmt-check test ## the full local gate (build + vet + fmt-check + test)

.PHONY: docker
docker: ## build the runtime image (docker build -t $(IMAGE_TAG) .)
	docker build -t $(IMAGE_TAG) .

.PHONY: smoke
smoke: ## run scripts/cli_smoke.sh (docker if available, else local)
	$(SMOKE_ENV) scripts/cli_smoke.sh

.PHONY: smoke-docker
smoke-docker: ## force the Docker server path in the smoke test
	USE_DOCKER=1 scripts/cli_smoke.sh

.PHONY: smoke-local
smoke-local: ## force the local go-build server path in the smoke test
	USE_DOCKER=0 scripts/cli_smoke.sh

.PHONY: clean
clean: ## remove the ./bin output directory
	rm -rf $(BINDIR)
