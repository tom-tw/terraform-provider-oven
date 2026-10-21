BINARY         := terraform-provider-oven
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

# OpenTofu dev_overrides entry for ~/.tofurc (note: the override
# points at the DIRECTORY containing terraform-provider-oven).
dev-overrides:
	@echo 'provider "registry.opentofu.org/tom-tw/oven" {'
	@echo '  path = "$${HOME}/bin"'
	@echo '}'

# OpenTofu ignores in-process provider reattach during `init`, so
# acceptance tests run through a dev_overrides CLI config pointing at
# the freshly built binary directory.
test-acc: build
	@printf 'provider_installation {\n  dev_overrides {\n    "registry.opentofu.org/tom-tw/oven" = "%s"\n  }\n}\n' "$(PWD)/bin" > .tofurc.acc
	@TF_ACC=1 TF_CLI_CONFIG_FILE="$(PWD)/.tofurc.acc" \
	  TF_ACC_PROVIDER_HOST="registry.opentofu.org" \
	  TF_ACC_PROVIDER_NAMESPACE="tom-tw" \
	  TF_ACC_TERRAFORM_PATH="$(TF_ACC_TERRAFORM_PATH)" \
	  TF_ACC_TERRAFORM_VERSION=$$(tofu version | head -1 | awk '{print $$2}' | tr -d v) \
	  go test ./internal/provider/... -v; \
	rm -f .tofurc.acc

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/