BINARY         := oven-provider
GOOS_NAME      := $(shell go env GOOS)
GOARCH_NAME    := $(shell go env GOARCH)
VERSION        ?= 0.1.0
TF_ACC_TERRAFORM_PATH ?= $(shell command -v tofu)

.PHONY: build install test test-acc lint fmt clean dev-overrides

build:
	go build -o bin/$(BINARY) .

install: build
	install -d "$(HOME)/bin"
	install bin/$(BINARY) "$(HOME)/bin/$(BINARY)"

# OpenTofu dev_overrides entry for ~/.tofurc
dev-overrides:
	@echo 'provider "registry.terraform.io/tom-tw/oven" {'
	@echo '  path = "$${HOME}/bin/$(BINARY)"'
	@echo '}'

test:
	go test ./...

test-acc: 
	TF_ACC=1 TF_ACC_TERRAFORM_PATH="$(TF_ACC_TERRAFORM_PATH)" \
	TF_ACC_TERRAFORM_VERSION=$$(tofu version -json | jq -r .terraform_version) \
	go test ./internal/provider/... -v

fmt:
	go fmt ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/