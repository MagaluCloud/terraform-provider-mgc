SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

SKIP_TF_STEP ?= true

# Directories
SCRIPT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
DOCS_DIR := $(SCRIPT_DIR)/docs
RESOURCES_DIR := $(DOCS_DIR)/resources
DATA_SOURCES_DIR := $(DOCS_DIR)/data-sources
DOCS_EXTRA_DIR := $(SCRIPT_DIR)/docs-extra

# Files
SUBCATEGORY_JSON := $(SCRIPT_DIR)/subcategory.json

# External tools and paths
TF_PLUGIN_DOCS := github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.19.4

.PHONY: help update-subcategory check-example-usage check-empty-subcategory generate-docs setup-config

update-subcategory: ## Update subcategories in documentation files
	@echo "Updating subcategories..."
	@if [ ! -d "$(RESOURCES_DIR)" ] || [ ! -d "$(DATA_SOURCES_DIR)" ] || [ ! -f "$(SUBCATEGORY_JSON)" ]; then \
		echo "Error: Required directories or files are missing."; \
		exit 1; \
	fi
	@if ! command -v jq &> /dev/null; then \
		echo "Error: jq is not installed. Please install it to parse JSON files."; \
		exit 1; \
	fi
	@jq -r 'to_entries | .[] | [.key, (.value | join(" "))] | @tsv' "$(SUBCATEGORY_JSON)" | while IFS=$$'\t' read -r category patterns; do \
		patterns=$$(echo "$$patterns" | tr -d '"'); \
		echo "Processing category: $$category, patterns: $$patterns"; \
		for pattern in $$patterns; do \
			find "$(RESOURCES_DIR)" "$(DATA_SOURCES_DIR)" -type f -name "$$pattern" | xargs -I {} sed -i "s/subcategory: .*/subcategory: \"$$category\"/" {}; \
		done; \
	done
	@echo "Subcategories updated successfully."

check-example-usage: ## Check for missing example usage in documentation files
	@echo "Checking for missing example usage..."
	@error_count=0; \
	for dir in "$(DATA_SOURCES_DIR)" "$(RESOURCES_DIR)"; do \
		echo "Checking files in $$dir"; \
		while IFS= read -r -d '' file; do \
			if ! grep -q '^## Example Usage$$' "$$file"; then \
				echo "Error: Missing '## Example Usage' in $$file"; \
				((error_count++)); \
			fi; \
		done < <(find "$$dir" -type f \( -name "*.md" -o -name "*.markdown" \) -print0); \
	done; \
	echo "Found $$error_count file(s) missing '## Example Usage'."; \
	[ $$error_count -eq 0 ]

check-empty-subcategory: ## Check for empty subcategories in documentation files
	@echo "Checking for empty subcategories..."
	@error_count=0; \
	for dir in "$(DATA_SOURCES_DIR)" "$(RESOURCES_DIR)"; do \
		echo "Checking files in $$dir"; \
		while IFS= read -r -d '' file; do \
			if grep -q '^subcategory: ""$$' "$$file"; then \
				echo "Error: Empty subcategory found in $$file"; \
				((error_count++)); \
			fi; \
		done < <(find "$$dir" -type f \( -name "*.md" -o -name "*.markdown" \) -print0); \
	done; \
	echo "Found $$error_count file(s) with empty subcategory."; \
	[ $$error_count -eq 0 ]

tf-docs-setup: ## Setup terraform-docs
	@go install github.com/terraform-docs/terraform-docs@v0.15.0

tf-gen-docs: ## Generate terraform docs
	@mkdir -p $(DOCS_DIR)
	@go run $(TF_PLUGIN_DOCS) generate --provider-dir="$(SCRIPT_DIR)"

generate-docs: ## Generate documentation
	@echo "Generating documentation..."
	@mkdir -p $(DOCS_DIR)
	@go run $(TF_PLUGIN_DOCS) generate --provider-dir="$(SCRIPT_DIR)"
	@echo "Adding subcategories..."
	$(MAKE) update-subcategory
	@echo "Moving extra docs..."
	@cp -r $(DOCS_EXTRA_DIR)/. $(DOCS_DIR)
	@echo "Documentation generated successfully."

go-fmt:
	gofmt -s -l -w .

go-vet:
	go vet ./...

go-test:
	go test -v ./...

before-commit: go-test go-fmt generate-docs check-example-usage check-empty-subcategory

build:
	goreleaser release --snapshot --clean --config "release.yaml" --skip "sign"
