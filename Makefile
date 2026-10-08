SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help

GO ?= go
GORELEASER ?= goreleaser
PACKAGE ?= ./cmd/invox
BIN_DIR ?= bin
BINARY_NAME ?= invox
BINARY_PATH ?= $(BIN_DIR)/$(BINARY_NAME)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo DEV)
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
BUILD_PKG := github.com/0xboris/invox/internal/build
LDFLAGS := -X $(BUILD_PKG).Version=$(VERSION) -X $(BUILD_PKG).Date=$(BUILD_DATE)

.PHONY: help build test fuzz vet lint vulncheck fmt tidy docs install release-snapshot clean

help: ## Show available targets.
	@printf "Targets:\n"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-17s %s\n", $$1, $$2}' "$(lastword $(MAKEFILE_LIST))"
	@printf "\nVariables:\n"
	@printf "  GO          Go toolchain to use (default: go)\n"
	@printf "  BIN_DIR     Build output directory (default: bin)\n"
	@printf "  VERSION     Version stamped into the binary (default: git describe)\n"
	@printf "  GORELEASER  GoReleaser binary for release-snapshot (default: goreleaser)\n"
	@printf "  FUZZTIME    How long make fuzz runs each fuzz target (default: 10s)\n"
	@printf "  GOLANGCI_LINT  golangci-lint command (default: go run at the version CI uses)\n"

# build is phony so it always runs go build; Go's build cache keeps it cheap
# and the binary can never be stale.
build: ## Build the local binary at ./bin/invox.
	mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(BINARY_PATH)" "$(PACKAGE)"

test: ## Run the Go test suite with the race detector.
	$(GO) test -race ./...

# Every package with a fuzz target; go test -fuzz takes one package at a time.
FUZZ_PACKAGES ?= $(shell grep -rl --include='*_test.go' '^func Fuzz' . | xargs -n1 dirname | sort -u)
FUZZTIME ?= 10s

fuzz: ## Run each fuzz target for FUZZTIME (default: 10s).
	set -e; \
	if [ -z "$(FUZZ_PACKAGES)" ]; then echo "no fuzz targets found" >&2; exit 1; fi; \
	for package in $(FUZZ_PACKAGES); do \
		targets="$$($(GO) test -list '^Fuzz' $$package)" || exit 1; \
		for target in $$(printf '%s\n' "$$targets" | grep '^Fuzz'); do \
			echo "$$package $$target"; \
			$(GO) test -run '^$$' -fuzz "^$$target\$$" -fuzztime "$(FUZZTIME)" $$package || exit 1; \
		done; \
	done

vet: ## Run go vet across the module.
	$(GO) vet ./...

# Keep in step with the lint and vulncheck jobs in .github/workflows/ci.yml.
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOVULNCHECK ?= $(GO) run golang.org/x/vuln/cmd/govulncheck@v1.8.0

lint: ## Run golangci-lint with the rules in .golangci.yml.
	$(GOLANGCI_LINT) run ./...

vulncheck: ## Check the module and the Go toolchain for known vulnerabilities.
	$(GOVULNCHECK) ./...

fmt: ## Format all Go files in place with gofmt.
	gofmt -w .

tidy: ## Check that go.mod and go.sum are tidy (prints the diff otherwise).
	$(GO) mod tidy -diff

docs: ## Regenerate docs/cli and share/man/man1 from the command tree (Linux or macOS).
	$(GO) run ./internal/docs/gen

install: ## Install invox into GOBIN or GOPATH/bin for use from anywhere.
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags "$(LDFLAGS)" "$(PACKAGE)"

release-snapshot: ## Build every release archive into ./dist without publishing (needs goreleaser v2).
	$(GORELEASER) release --snapshot --clean --skip=publish

clean: ## Remove the build output: bin, dist and completions.
	rm -rf "$(BIN_DIR)" dist completions
