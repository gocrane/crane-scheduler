GOOS ?= $(shell go env GOOS)

# Git information
GIT_VERSION ?= $(shell git describe --tags --always)
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS = "-X github.com/gocrane/crane-scheduler/pkg/version.gitVersion=$(GIT_VERSION) \
                      -X github.com/gocrane/crane-scheduler/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
                      -X github.com/gocrane/crane-scheduler/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/gocrane/crane-scheduler/pkg/version.buildDate=$(BUILDDATE)"

# Images management
REGISTRY ?= docker.io
REGISTRY_NAMESPACE ?= gocrane
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""

# Image URL to use all building/pushing image targets
SCHEDULER_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/crane-scheduler:${GIT_VERSION}"
CONTROLLER_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/crane-scheduler-controller:${GIT_VERSION}"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: golangci-lint  ## Run golang lint against code
	@$(GOLANG_LINT) run \
      --timeout 30m \
      --disable-all \
      -E deadcode \
      -E unused \
      -E varcheck \
      -E ineffassign \
      -E goimports \
      -E gofmt \
      -E misspell \
      -E unparam \
      -E unconvert \
      -E govet \
      -E errcheck \
      -E structcheck

.PHONY: test
test: fmt vet lint ## Run tests.
	go test -coverprofile coverage.out -covermode=atomic ./...

.PHONY: build
build: scheduler controller

.PHONY: all
all: test scheduler controller

.PHONY: scheduler
scheduler: ## Build binary with the crane scheduler.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/scheduler cmd/scheduler/main.go

.PHONY: controller
controller: ## Build binary with the controller.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/controller cmd/controller/main.go

.PHONY: images
images: image-scheduler image-controller

.PHONY: image-scheduler
image-scheduler: ## Build docker image with the crane scheduler.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=scheduler -t ${SCHEDULER_IMG} .

.PHONY: image-controller
image-controller: ## Build docker image with the controller.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=controller -t ${CONTROLLER_IMG} .

.PHONY: push-images
push-images: push-image-scheduler push-image-controller

.PHONY: push-image-scheduler
push-image-scheduler: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${SCHEDULER_IMG}

.PHONY: push-image-controller
push-image-controller: ## Push images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${CONTROLLER_IMG}

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GOLANG_LINT_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLANG_LINT_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0 ;\
	rm -rf $$GOLANG_LINT_TMP_DIR ;\
	}
GOLANG_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANG_LINT=$(shell which golangci-lint)
endif

goimports:
ifeq (, $(shell which goimports))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GO_IMPORTS_TMP_DIR=$$(mktemp -d) ;\
	cd $$GO_IMPORTS_TMP_DIR ;\
	go mod init tmp ;\
	go get golang.org/x/tools/cmd/goimports@v0.1.7 ;\
	rm -rf $$GO_IMPORTS_TMP_DIR ;\
	}
GO_IMPORTS=$(shell go env GOPATH)/bin/goimports
else
GO_IMPORTS=$(shell which goimports)
endif
