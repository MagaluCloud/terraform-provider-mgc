SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

SKIP_TF_STEP ?= true
CSPELL_VERSION = "latest"

# Test selector for acceptance-test targets (override, e.g. RUN=TestAccKubernetes)
RUN ?= TestAcc

# Optional env profile for acceptance tests: PROFILE=dev loads env/dev.env
# (plain KEY=value lines) into the environment of every recipe.
ifdef PROFILE
include env/$(PROFILE).env
export $(shell grep -E '^[A-Za-z_][A-Za-z0-9_]*=' env/$(PROFILE).env | cut -d= -f1)
endif

# Go commands
GO              := go
GOFMT           := gofmt
GOVET           := go vet
GOTEST          := go test

# Directories
DOCS_DIR_PATH   := docs
MGC_DIR_PATH    := mgc
SCRIPT_DIR      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
DOCS_DIR        := $(SCRIPT_DIR)/$(DOCS_DIR_PATH)
RESOURCES_DIR   := $(DOCS_DIR)/resources
DATA_SOURCES_DIR := $(DOCS_DIR)/data-sources
DOCS_EXTRA_DIR  := $(SCRIPT_DIR)/docs-extra

# Files
SUBCATEGORY_JSON := $(SCRIPT_DIR)/subcategory.json

# External tools and paths
TF_PLUGIN_DOCS  := github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
TERRAFORM_DOCS  := github.com/terraform-docs/terraform-docs@latest

# Styling
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
NC     := \033[0m # No Color

# Declare all targets as phony
.PHONY: help update-subcategory check-example-usage check-empty-subcategory generate-docs \
        tf-docs-setup tf-gen-docs go-fmt go-vet go-test testacc-record testacc-replay testacc-live \
        download-cassetes pre-commit build before-commit debug clean all

install:
	@export GOBIN=${PWD}/bin
	@export PATH=${GOBIN}:${PATH}
	@go install $(TF_PLUGIN_DOCS)

help: ## Display this help screen
	@echo -e "$(GREEN)Available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

update-subcategory: ## Update subcategories in documentation files
	@echo -e "$(GREEN)Updating subcategories...$(NC)"
	@if [ ! -d "$(RESOURCES_DIR)" ] || [ ! -d "$(DATA_SOURCES_DIR)" ] || [ ! -f "$(SUBCATEGORY_JSON)" ]; then \
		echo -e "$(RED)Error: Required directories or files are missing.$(NC)"; \
		exit 1; \
	fi
	@if ! command -v jq &> /dev/null; then \
		echo -e "$(RED)Error: jq is not installed. Please install it to parse JSON files.$(NC)"; \
		exit 1; \
	fi
	@jq -r 'to_entries | .[] | [.key, (.value | join(" "))] | @tsv' "$(SUBCATEGORY_JSON)" | while IFS=$$'\t' read -r category patterns; do \
		patterns=$$(echo "$$patterns" | tr -d '"'); \
		echo "Processing category: $$category, patterns: $$patterns"; \
		for pattern in $$patterns; do \
			find "$(RESOURCES_DIR)" "$(DATA_SOURCES_DIR)" -type f -name "$$pattern" -print0 | \
			while IFS= read -r -d '' file; do \
				tmp_file="$$file.tmp"; \
				sed "s/subcategory: .*/subcategory: \"$$category\"/" "$$file" > "$$tmp_file" && mv "$$tmp_file" "$$file"; \
			done; \
		done; \
	done
	@echo -e "$(GREEN)Subcategories updated successfully.$(NC)"

check-example-usage: ## Check for missing example usage in documentation files
	@echo -e "$(GREEN)Checking for missing example usage...$(NC)"
	@error_count=0; \
	for dir in "$(DATA_SOURCES_DIR)" "$(RESOURCES_DIR)"; do \
		echo "Checking files in $$dir"; \
		while IFS= read -r -d '' file; do \
			if ! grep -q '^## Example Usage$$' "$$file"; then \
				echo -e "$(RED)Error: Missing '## Example Usage' in $$file$(NC)"; \
				((error_count++)); \
			fi; \
		done < <(find "$$dir" -type f \( -name "*.md" -o -name "*.markdown" \) -print0); \
	done; \
	echo "Found $$error_count file(s) missing '## Example Usage'."; \
	[ $$error_count -eq 0 ]

check-empty-subcategory: ## Check for empty subcategories in documentation files
	@echo -e "$(GREEN)Checking for empty subcategories...$(NC)"
	@error_count=0; \
	for dir in "$(DATA_SOURCES_DIR)" "$(RESOURCES_DIR)"; do \
		echo "Checking files in $$dir"; \
		while IFS= read -r -d '' file; do \
			if grep -q '^subcategory: ""$$' "$$file"; then \
				echo -e "$(RED)Error: Empty subcategory found in $$file$(NC)"; \
				((error_count++)); \
			fi; \
		done < <(find "$$dir" -type f \( -name "*.md" -o -name "*.markdown" \) -print0); \
	done; \
	echo "Found $$error_count file(s) with empty subcategory."; \
	[ $$error_count -eq 0 ]

tf-docs-setup: ## Setup terraform-docs
	@echo -e "$(GREEN)Installing terraform-docs...$(NC)"
	@$(GO) install $(TERRAFORM_DOCS)

tf-gen-docs: ## Generate terraform docs
	@echo -e "$(GREEN)Generating terraform docs with tfplugindocs...$(NC)"
	@mkdir -p $(DOCS_DIR)
	@$(GO) run $(TF_PLUGIN_DOCS) generate --provider-dir="$(SCRIPT_DIR)" --provider-name="terraform-provider-mgc"

generate-docs: tf-gen-docs ## Generate full documentation
	@echo -e "$(GREEN)Generating documentation...$(NC)"
	@mkdir -p $(DOCS_DIR)
	@$(GO) run $(TF_PLUGIN_DOCS) generate --provider-dir="$(SCRIPT_DIR)" --provider-name="terraform-provider-mgc"
	@echo -e "$(GREEN)Adding subcategories...$(NC)"
	$(MAKE) update-subcategory
	@echo -e "$(GREEN)Moving extra docs...$(NC)"
	@cp -r $(DOCS_EXTRA_DIR)/. $(DOCS_DIR)
	@echo -e "$(GREEN)Documentation generated successfully.$(NC)"

go-fmt: ## Format Go code
	@echo -e "$(GREEN)Formatting Go code...$(NC)"
	@$(GOFMT) -s -l -w .

go-vet: ## Run Go vet
	@echo -e "$(GREEN)Running go vet...$(NC)"
	@$(GOVET) ./...

go-test: ## Run Go tests
	@echo -e "$(GREEN)Running tests...$(NC)"
	@$(GOTEST) -v ./...

testacc-record: ## Record VCR cassettes against a live API (RUN= is mandatory; requires MGC_API_KEY and MGC_ENDPOINT)
	@test "$(origin RUN)" = "command line" || { echo -e "$(RED)RUN must be given explicitly (e.g. make testacc-record RUN=TestAccKubernetesCluster_basic); recording everything at once is never implicit$(NC)"; exit 1; }
	@test -n "$${MGC_API_KEY:-}" || { echo -e "$(RED)MGC_API_KEY is required to record$(NC)"; exit 1; }
	@test -n "$${MGC_ENDPOINT:-}" || { echo -e "$(RED)MGC_ENDPOINT is required to record (root URL of the target API; no default fallback)$(NC)"; exit 1; }
	@echo -e "$(GREEN)Recording acceptance-test cassettes against $$MGC_ENDPOINT...$(NC)"
	@TF_ACC=1 MGC_VCR_MODE=record $(GOTEST) -p 1 -parallel 1 -v ./mgc/... -run '$(RUN)' -timeout 180m

testacc-replay: ## Replay acceptance tests from cassettes (hermetic: no env, no secrets, no network)
	@echo -e "$(GREEN)Replaying acceptance tests from cassettes...$(NC)"
	@TF_ACC=1 MGC_VCR_MODE=replay $(GOTEST) -v ./mgc/... -run '$(RUN)' -timeout 30m

testacc-live: ## Run acceptance tests live, without cassettes (requires MGC_API_KEY and MGC_ENDPOINT; use PROFILE=dev for the fake server)
	@test -n "$${MGC_API_KEY:-}" || { echo -e "$(RED)MGC_API_KEY is required to run live$(NC)"; exit 1; }
	@test -n "$${MGC_ENDPOINT:-}" || { echo -e "$(RED)MGC_ENDPOINT is required to run live (root URL of the target API; no default fallback)$(NC)"; exit 1; }
	@echo -e "$(GREEN)Running acceptance tests live against $$MGC_ENDPOINT...$(NC)"
	@TF_ACC=1 MGC_VCR_MODE=off $(GOTEST) -v ./mgc/... -run '$(RUN)' -timeout 180m

download-cassetes: ## Download a published cassette set into staging (default: main; override with VERSION=<set>)
	@go run ./mgc/internal/acctest/cassettes download $(VERSION)

pre-commit: ## Publish cassettes: gate the staging set with a full replay, then upload it under the branch name
	@$(MAKE) testacc-replay
	@go run ./mgc/internal/acctest/cassettes publish

.PHONY: sweep
sweep: ## DESTRUCTIVE: delete leaked acceptance-test resources (tf-acctest-*) on MGC_ENDPOINT
	@test -n "$${MGC_ENDPOINT:-}" || { echo -e "$(RED)MGC_ENDPOINT is required (root URL of the API to sweep)$(NC)"; exit 1; }
	@test -n "$${MGC_API_KEY:-}"  || { echo -e "$(RED)MGC_API_KEY is required$(NC)"; exit 1; }
	@echo -e "$(GREEN)Sweeping leaked tf-acctest resources on $$MGC_ENDPOINT...$(NC)"
	@$(GOTEST) -v $$(grep -rl AddTestSweepers mgc --include='*_test.go' \
	  | xargs -n1 dirname | sort -u | sed 's#^#./#') \
	  -sweep="$${MGC_REGION:-br-se1}" $(if $(SWEEP_RUN),-sweep-run="$(SWEEP_RUN)") -timeout 60m

build: ## Build the provider
	@echo -e "$(GREEN)Building the provider...$(NC)"
	@goreleaser release --snapshot --clean --config "release.yaml" --skip "sign"

before-commit: go-test go-fmt spell-check-docs generate-docs check-example-usage check-empty-subcategory ## Run all checks before committing code
	@echo -e "$(GREEN)All pre-commit checks passed!$(NC)"

debug: ## Run the provider in debug mode
	@echo -e "$(GREEN)Running in debug mode...$(NC)"
	@$(GO) run main.go --debug

clean: ## Clean build artifacts
	@echo -e "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf dist/
	@rm -f terraform-provider-mgc
	@echo -e "$(GREEN)Clean complete$(NC)"

all: clean go-fmt go-vet go-test generate-docs build ## Run all main tasks
	@echo -e "$(GREEN)All tasks completed successfully!$(NC)"

spell-check-docs: ## Spell check documentation
	@echo -e "$(GREEN)*** Checking docs for miss spellings... ***$(NC)"
	@grep . cspell.txt | sort > .cspell.txt && mv .cspell.txt cspell.txt
	@docker run --quiet -v ${PWD}:/workdir ghcr.io/streetsidesoftware/cspell:$(CSPELL_VERSION) lint -c cspell.json --no-progress --unique $(DOCS_DIR_PATH)
	@echo -e "$(GREEN)*** MGC Terraform Provider is correctly written! ***$(NC)"

spell-check-code: ## Spell check codebase
	@echo -e "$(GREEN)*** Checking code base for miss spellings... ***$(NC)"
	@docker run -v ${PWD}:/workdir ghcr.io/streetsidesoftware/cspell:$(CSPELL_VERSION) lint -c cspell.json --no-progress --unique $(MGC_DIR_PATH)
	@echo -e "$(GREEN)*** MGC Terraform Provider is correctly written! ***$(NC)"
