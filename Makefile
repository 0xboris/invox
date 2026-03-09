SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.DEFAULT_GOAL := help

GO ?= go
PACKAGE ?= ./cmd/invox
BIN_DIR ?= bin
BINARY_NAME ?= invox
BINARY_PATH ?= $(BIN_DIR)/$(BINARY_NAME)
CLI ?= ./$(BINARY_PATH)

INPUT ?= invoice.yaml
CUSTOMERS ?=
ISSUER ?=
TEMPLATE ?=
TEX_OUTPUT ?= invoice.tex
PDF_OUTPUT ?= $(basename $(INPUT)).pdf
ARGS ?=

COMMON_FLAGS = -i "$(INPUT)" $(if $(strip $(CUSTOMERS)),-c "$(CUSTOMERS)") $(if $(strip $(ISSUER)),-u "$(ISSUER)")
RENDER_FLAGS = $(COMMON_FLAGS) $(if $(strip $(TEMPLATE)),-t "$(TEMPLATE)")

.PHONY: help build test vet install validate render pdf clean
.PHONY: archive

help: ## Show available targets and overridable variables.
	@printf "Targets:\n"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' "$(lastword $(MAKEFILE_LIST))"
	@printf "\nVariables:\n"
	@printf "  INPUT       Invoice YAML input path (default: invoice.yaml)\n"
	@printf "  CUSTOMERS   Optional customers.yaml path override\n"
	@printf "  ISSUER      Optional issuer.yaml path override\n"
	@printf "  TEMPLATE    Optional template.tex path override for render/pdf\n"
	@printf "  TEX_OUTPUT  TeX output path for render (default: invoice.tex)\n"
	@printf "  PDF_OUTPUT  PDF output path for pdf (default: INPUT with .pdf extension)\n"
	@printf "  CLI         CLI executable for validate/render/pdf (default: ./bin/invox)\n"
	@printf "  ARGS        Extra CLI arguments appended to the command\n"
	@printf "\nExamples:\n"
	@printf "  make build\n"
	@printf "  make test\n"
	@printf "  make install\n"
	@printf "  make validate\n"
	@printf "  make render CUSTOMERS=customers.yaml ISSUER=issuer.yaml TEMPLATE=invoice_template.tex\n"
	@printf "  make pdf CLI=invox INPUT=invoice.yaml\n"
	@printf "  make archive CLI=invox INPUT=invoice.yaml\n"

$(BINARY_PATH):
	mkdir -p "$(BIN_DIR)"
	$(GO) build -o "$(BINARY_PATH)" "$(PACKAGE)"

build: $(BINARY_PATH) ## Build the local binary at ./bin/invox.

test: ## Run the Go test suite.
	$(GO) test ./...

vet: ## Run go vet across the module.
	$(GO) vet ./...

install: ## Install invox into GOBIN or GOPATH/bin for use from anywhere.
	$(GO) install "$(PACKAGE)"

validate: $(BINARY_PATH) ## Validate INPUT with the local CLI.
	$(CLI) validate $(COMMON_FLAGS) $(ARGS)

render: $(BINARY_PATH) ## Render INPUT to TEX_OUTPUT with the local CLI.
	$(CLI) render $(RENDER_FLAGS) -o "$(TEX_OUTPUT)" $(ARGS)

pdf: $(BINARY_PATH) ## Build INPUT to PDF_OUTPUT with the local CLI.
	$(CLI) build $(RENDER_FLAGS) -o "$(PDF_OUTPUT)" $(ARGS)

archive: $(BINARY_PATH) ## Move a built INPUT invoice into the configured archive directory.
	$(CLI) archive -i "$(INPUT)" $(ARGS)

clean: ## Remove the local build output directory.
	rm -rf "$(BIN_DIR)"
