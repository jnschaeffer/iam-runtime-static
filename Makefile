all: lint test
PHONY: test coverage lint golint clean vendor docker-up docker-down unit-test
# use the working dir as the app name, this should be the repo name
APP_NAME=$(shell basename $(CURDIR))

test: | unit-test

unit-test: | lint
	@echo Running unit tests...
	@go test -cover -short -tags testtools ./...

coverage:
	@echo Generating coverage report...
	@go test ./... -race -coverprofile=coverage.out -covermode=atomic -tags testtools -p 1
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint

golint: | vendor
	@echo Linting Go files...
	@golangci-lint run --build-tags "-tags testtools"

build:
	@go mod download
	@CGO_ENABLED=0 go build -mod=readonly -v -o bin/${APP_NAME}
