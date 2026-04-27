# Formae Plugin Makefile
#
# Targets:
#   build   - Build the plugin binary
#   test    - Run tests
#   lint    - Run linter
#   clean   - Remove build artifacts
#   install - Build and install plugin locally (binary + schema + manifest)

# Plugin metadata - extracted from formae-plugin.pkl
PLUGIN_NAME := $(shell pkl eval -x 'name' formae-plugin.pkl 2>/dev/null || echo "example")
PLUGIN_VERSION := $(shell pkl eval -x 'version' formae-plugin.pkl 2>/dev/null || echo "0.0.0")
PLUGIN_NAMESPACE := $(shell pkl eval -x 'namespace' formae-plugin.pkl 2>/dev/null || echo "EXAMPLE")

# Build settings
GO := go
GOFLAGS := -trimpath
BINARY := $(PLUGIN_NAME)

# Installation paths
# Plugin discovery expects lowercase directory names matching the plugin name
PLUGIN_BASE_DIR := $(HOME)/.pel/formae/plugins
INSTALL_DIR := $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v$(PLUGIN_VERSION)

.PHONY: all build test test-unit test-integration lint verify-schema clean install install-dev gen-pkl help setup-credentials clean-environment conformance-test conformance-test-crud conformance-test-discovery

all: build

## build: Build the plugin binary and update manifest
build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY) .
	@MIN_VERSION=$$($(GO) list -m -f '{{.Dir}}' github.com/platform-engineering-labs/formae/pkg/plugin 2>/dev/null | xargs -I{} grep 'MinFormaeVersion' {}/version.go 2>/dev/null | grep -oE '"[0-9]+\.[0-9]+\.[0-9]+"' | tr -d '"'); \
	if [ -n "$$MIN_VERSION" ]; then \
		echo "Updating minFormaeVersion to $$MIN_VERSION"; \
		if [ "$$(uname)" = "Darwin" ]; then \
			sed -i '' 's/^minFormaeVersion = .*/minFormaeVersion = "'"$$MIN_VERSION"'"/' formae-plugin.pkl; \
		else \
			sed -i 's/^minFormaeVersion = .*/minFormaeVersion = "'"$$MIN_VERSION"'"/' formae-plugin.pkl; \
		fi; \
	fi

## test: Run all tests
test:
	$(GO) test -v ./...

## test-unit: Run unit tests only (tests with //go:build unit tag)
test-unit:
	$(GO) test -v -tags=unit ./...

## test-integration: Run integration tests (requires cloud credentials)
## Add tests with //go:build integration tag
test-integration:
	$(GO) test -v -tags=integration ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## verify-schema: Validate PKL schema files
## Checks that schema files are well-formed and follow formae conventions.
verify-schema:
	$(GO) run github.com/platform-engineering-labs/formae/pkg/plugin/testutil/cmd/verify-schema --namespace $(PLUGIN_NAMESPACE) ./schema/pkl

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/

## install: Build and install plugin locally (binary + schema + manifest)
## Installs to ~/.pel/formae/plugins/<name>/v<version>/
## Removes any existing versions of the plugin first to ensure clean state.
install: build
	@echo "Installing $(PLUGIN_NAME) v$(PLUGIN_VERSION) (namespace: $(PLUGIN_NAMESPACE))..."
	@rm -rf $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)
	@mkdir -p $(INSTALL_DIR)/schema/pkl
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@cp -r schema/pkl/* $(INSTALL_DIR)/schema/pkl/
	@if [ -f schema/Config.pkl ]; then cp schema/Config.pkl $(INSTALL_DIR)/schema/; fi
	@cp formae-plugin.pkl $(INSTALL_DIR)/
	@echo "Installed to $(INSTALL_DIR)"
	@echo "  - Binary: $(INSTALL_DIR)/$(BINARY)"
	@echo "  - Schema: $(INSTALL_DIR)/schema/"
	@echo "  - Manifest: $(INSTALL_DIR)/formae-plugin.pkl"

## install-dev: Install as v0.0.0 for development/debugging
install-dev: build
	@echo "Installing $(PLUGIN_NAME) v0.0.0 (dev) (namespace: $(PLUGIN_NAMESPACE))..."
	@rm -rf $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)
	@mkdir -p $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0/schema/pkl
	@cp bin/$(BINARY) $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0/$(BINARY)
	@cp -r schema/pkl/* $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0/schema/pkl/
	@cp formae-plugin.pkl $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0/
	@echo "Installed to $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v0.0.0"

## gen-pkl: Resolve PKL dependencies
gen-pkl:
	pkl project resolve schema/pkl
	pkl project resolve examples
	pkl project resolve testdata

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## setup-credentials: Verify OCI credentials are configured
setup-credentials:
	@./scripts/ci/setup-credentials.sh

## clean-environment: Clean up test resources in cloud environment
## Called before and after conformance tests. Edit scripts/ci/clean-environment.sh
## to configure for your provider.
clean-environment:
	@./scripts/ci/clean-environment.sh

## conformance-test: Run all conformance tests (CRUD + discovery)
## Usage: make conformance-test [VERSION=0.80.0] [TEST=bucket]
## Downloads the specified formae version (or latest) and runs conformance tests.
## Calls clean-environment before and after tests.
##
## Parameters:
##   VERSION - Formae version to test against (default: latest)
##   TEST    - Filter tests by name pattern (e.g., TEST=bucket)
conformance-test: conformance-test-crud conformance-test-discovery

## conformance-test-crud: Run only CRUD lifecycle tests
## Usage: make conformance-test-crud [VERSION=0.80.0] [TEST=bucket]
## Note: Environment cleanup is skipped when FORMAE_TEST_FILTER is set (e.g. matrix CI)
## to avoid parallel jobs deleting each other's resources. Use clean-environment target
## or a separate CI cleanup job instead.
conformance-test-crud: install
	@if [ -z "$(FORMAE_TEST_FILTER)" ] && [ -z "$(TEST)" ]; then \
		echo "Pre-test cleanup..."; \
		./scripts/ci/clean-environment.sh || true; \
		echo ""; \
	fi
	@echo "Running CRUD conformance tests..."
	@FORMAE_TEST_FILTER="$(if $(TEST),$(TEST),$(FORMAE_TEST_FILTER))" FORMAE_TEST_TYPE=crud ./scripts/run-conformance-tests.sh $(VERSION); \
	TEST_EXIT=$$?; \
	if [ -z "$(FORMAE_TEST_FILTER)" ] && [ -z "$(TEST)" ]; then \
		echo ""; \
		echo "Post-test cleanup..."; \
		./scripts/ci/clean-environment.sh || true; \
	fi; \
	exit $$TEST_EXIT

## conformance-test-discovery: Run only discovery tests
## Usage: make conformance-test-discovery [VERSION=0.80.0] [TEST=bucket]
## NOTE: natgateway, servicegateway, instance, cluster, nodepool, and virtualnodepool
## are excluded by default due to service limits in us-chicago-1.
DISCOVERY_DEFAULT_FILTER := policy,vcn,volume,bucket,networksecuritygroup,internetgateway,routetable,securitylist,subnet,dhcpoptions,nsg_securityrule
conformance-test-discovery: install
	@if [ -z "$(FORMAE_TEST_FILTER)" ] && [ -z "$(TEST)" ]; then \
		echo "Pre-test cleanup..."; \
		./scripts/ci/clean-environment.sh || true; \
		echo ""; \
	fi
	@echo "Running discovery conformance tests..."
	@FORMAE_TEST_FILTER="$(if $(TEST),$(TEST),$(if $(FORMAE_TEST_FILTER),$(FORMAE_TEST_FILTER),$(DISCOVERY_DEFAULT_FILTER)))" FORMAE_TEST_TYPE=discovery ./scripts/run-conformance-tests.sh $(VERSION); \
	TEST_EXIT=$$?; \
	if [ -z "$(FORMAE_TEST_FILTER)" ] && [ -z "$(TEST)" ]; then \
		echo ""; \
		echo "Post-test cleanup..."; \
		./scripts/ci/clean-environment.sh || true; \
	fi; \
	exit $$TEST_EXIT
