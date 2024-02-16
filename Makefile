ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOOLS_DIR := .tools
GOOS ?= linux
GOARCH ?= amd64

GOLANGCI_LINT_REPO = github.com/golangci/golangci-lint
GOLANGCI_LINT_VERSION = v1.56.1

# use the working dir as the app name, this should be the repo name
APP_NAME=$(shell basename $(CURDIR))

PHONY: all test lint build go-dependencies

all: go-dependencies test build

test: lint
	@echo Running unit tests...
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1

lint: $(TOOLS_DIR)/golangci-lint
	@echo Linting Go files...
	@$(TOOLS_DIR)/golangci-lint run --modules-download-mode=readonly

build:
	@CGO_ENABLED=0 go build -mod=readonly -v -o bin/${APP_NAME}

go-dependencies:
	@go mod download
	@go mod tidy

$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

$(TOOLS_DIR)/golangci-lint: | $(TOOLS_DIR)
	@echo "Installing $(GOLANGCI_LINT_REPO)/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"
	@GOBIN=$(ROOT_DIR)/$(TOOLS_DIR) go install $(GOLANGCI_LINT_REPO)/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
