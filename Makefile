# Git
GIT_VERSION ?= $(shell git describe --tags --always)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT_ID ?= $(shell git rev-parse --short HEAD)

# Image URL to use all building/pushing image targets
REG ?= ghcr.io
REG_NS ?= koordinator-sh
REG_USER ?= ""
REG_PWD ?= ""

YARN_COPILOT_IMG ?= "${REG}/${REG_NS}/yarn-copilot:${GIT_BRANCH}-${GIT_COMMIT_ID}"
YARN_OPERATOR_IMG ?= "${REG}/${REG_NS}/yarn-operator:${GIT_BRANCH}-${GIT_COMMIT_ID}"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

AGENT_MODE ?= hostMode
# Set license header files.
LICENSE_HEADER_GO ?= hack/boilerplate/boilerplate.go.txt

PACKAGES ?= $(shell go list ./... | grep -vE 'vendor|test/e2e')

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

.PHONY: all
all: build

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

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="$(LICENSE_HEADER_GO)" paths="./apis/..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: lint-go lint-license ## Lint all code.

.PHONY: lint-go
lint-go: golangci-lint ## Lint Go code.
	$(GOLANGCI_LINT) run -v --timeout=10m

.PHONY: lint-license
lint-license:
	@hack/update-license-header.sh

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" agent_mode=$(AGENT_MODE) go test $(PACKAGES) -race -covermode atomic -coverprofile cover.out

.PHONY: fast-test
fast-test: envtest ## Run tests fast.
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" agent_mode=$(AGENT_MODE) go test $(PACKAGES) -race -covermode atomic -coverprofile cover.out

##@ Build

.PHONY: build
build: build-yarn-operator

.PHONY: build-yarn-copilot
build-yarn-copilot: ## Build yarn-copilot binary.
	go build -o bin/yarn-copilot cmd/yarn-copilot/main.go

.PHONY: build-yarn-operator
build-yarn-operator: ## Build yarn-operator binary.
	go build -o bin/yarn-operator cmd/yarn-operator/main.go

.PHONY: docker-build
docker-build: test docker-build-yarn-copilot docker-build-yarn-operator

.PHONY: docker-build-yarn-copilot
docker-build-yarn-copilot: ## Build docker image with the yarn-copilot.
	docker build --pull -t ${YARN_COPILOT_IMG} -f docker/yarn-copilot.dockerfile .

.PHONY: docker-build-yarn-operator
docker-build-yarn-operator: ## Build docker image with the yarn-operator.
	docker build --pull -t ${YARN_OPERATOR_IMG} -f docker/yarn-operator.dockerfile .

.PHONY: docker-push
docker-push: docker-push-yarn-copilot docker-push-yarn-operator

.PHONY: docker-push-yarn-copilot
docker-push-yarn-copilot: ## Push docker image with the yarn-copilot.
ifneq ($(REG_USER), "")
	docker login -u $(REG_USER) -p $(REG_PWD) ${REG}
endif
	docker push ${YARN_COPILOT_IMG}

.PHONY: docker-push-yarn-operator
docker-push-yarn-operator: ## Push docker image with the yarn-operator.
ifneq ($(REG_USER), "")
	docker login -u $(REG_USER) -p $(REG_PWD) ${REG}
endif
	docker push ${YARN_OPERATOR_IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image manager=$(YARN_OPERATOR_IMG) yarn-copilot=$(YARN_COPILOT_IMG)
	@hack/kustomize.sh $(KUSTOMIZE) | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@hack/kustomize.sh $(KUSTOMIZE) | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GINKGO ?= $(LOCALBIN)/ginkgo
HACK_DIR ?= $(PWD)/hack

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.0
GOLANGCILINT_VERSION ?= v1.47.3
GINKGO_VERSION ?= v1.16.4

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCILINT_VERSION)

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/ginkgo@$(GINKGO_VERSION)
