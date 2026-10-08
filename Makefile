SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help

GO ?= go
PACKAGE ?= ./cmd/invox
BIN_DIR ?= bin
BINARY_NAME ?= invox
BINARY_PATH ?= $(BIN_DIR)/$(BINARY_NAME)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo DEV)
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
BUILD_PKG := github.com/0xboris/invox/internal/build
LDFLAGS := -X $(BUILD_PKG).Version=$(VERSION) -X $(BUILD_PKG).Date=$(BUILD_DATE)

.PHONY: help build test fuzz vet lint fmt tidy docs install clean

help: ## Show available targets.
	@printf "Targets:\n"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' "$(lastword $(MAKEFILE_LIST))"
	@printf "\nVariables:\n"
	@printf "  GO          Go toolchain to use (default: go)\n"
	@printf "  BIN_DIR     Build output directory (default: bin)\n"
	@printf "  VERSION     Version stamped into the binary (default: git describe)\n"
	@printf "  FUZZTIME    How long make fuzz runs each fuzz target (default: 10s)\n"

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
		for target in $$($(GO) test -list '^Fuzz' $$package | grep '^Fuzz'); do \
			echo "$$package $$target"; \
			$(GO) test -run '^$$' -fuzz "^$$target\$$" -fuzztime "$(FUZZTIME)" $$package || exit 1; \
		done; \
	done

vet: ## Run go vet across the module.
	$(GO) vet ./...

lint: ## Run the linters (go vet for now).
	$(GO) vet ./...

fmt: ## Format all Go files in place with gofmt.
	gofmt -w .

tidy: ## Check that go.mod and go.sum are tidy (prints the diff otherwise).
	$(GO) mod tidy -diff

docs: ## Regenerate docs/cli and share/man/man1 from the command tree.
	$(GO) run ./internal/docs/gen

install: ## Install invox into GOBIN or GOPATH/bin for use from anywhere.
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags "$(LDFLAGS)" "$(PACKAGE)"

clean: ## Remove the local build output directory.
	rm -rf "$(BIN_DIR)"
