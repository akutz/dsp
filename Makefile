#!/usr/bin/env bash

#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

## --------------------------------------
## General
## --------------------------------------

SHELL:=/usr/bin/env bash
.DEFAULT_GOAL:=help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE=on

# Get the information about the platform on which the tools are built/run.
GOHOSTOS := $(shell go env GOHOSTOS)
GOHOSTARCH := $(shell go env GOHOSTARCH)
GOHOSTOSARCH := $(GOHOSTOS)_$(GOHOSTARCH)

# Default the GOOS and GOARCH values to be the same as the platform on which
# this Makefile is being executed.
GOOS ?= $(GOHOSTOS)
GOARCH ?= $(GOHOSTARCH)

# Directories
BIN_DIR       := bin
TOOLS_DIR     := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin/$(GOHOSTOSARCH)

# Binaries
BIN_NAME      := $(BIN_DIR)/dsp.$(GOOS)_$(GOARCH)

# Tooling binaries
GO_RELEASER   := $(TOOLS_BIN_DIR)/go-releaser
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint


## --------------------------------------
## Help
## --------------------------------------

# The help will print out all targets with their descriptions organized below
# their categories. The categories are represented by `##@` and the target
# descriptions by `##`.
#
# The awk commands is responsible to read the entire set of makefiles included
# in this invocation, looking for lines of the file as xyz: ## something, and
# then pretty-format the target and help. Then, if there's a line with ##@
# something, that gets pretty-printed as a category.
# 
# More info over the usage of ANSI control characters for terminal
# formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
#
# More info over awk command: http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


## --------------------------------------
## Tooling Binaries
## --------------------------------------

TOOLING_BINARIES := $(GO_RELEASER) $(GOLANGCI_LINT)
tools: $(TOOLING_BINARIES) ## Build tooling binaries
$(TOOLING_BINARIES):
	make -C $(TOOLS_DIR) $(@F)


## --------------------------------------
## Linting and fixing linter errors
## --------------------------------------

.PHONY: lint
lint: ## Run all the lint targets
	$(MAKE) lint-go-full

GOLANGCI_LINT_FLAGS ?= --fast=true
.PHONY: lint-go
lint-go: $(GOLANGCI_LINT)
lint-go: ## Lint codebase
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_FLAGS)

.PHONY: lint-go-full
lint-go-full: GOLANGCI_LINT_FLAGS = --fast=false
lint-go-full: lint-go ## Run slower linters to detect possible issues


## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Validates the modules
	go mod tidy
	cd hack/tools && go mod tidy


## --------------------------------------
## Build
## --------------------------------------

build: $(BIN_NAME)
build: ## Build dsp
$(BIN_NAME): go.mod main.go
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags netgo,osusergo -o $@ .


## --------------------------------------
## Release
## --------------------------------------

.PHONY: release
release: ## Release dsp
release: $(GO_RELEASER)
	$(GO_RELEASER) release --snapshot --clean


## --------------------------------------
## Clean and clobber
## --------------------------------------

.PHONY: clean
clean: ## Clean up
	rm -fr $(BIN_NAME) dist


