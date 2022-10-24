
# Image URL to use all building/pushing image targets
PREFIX?=projects.registry.vmware.com/tanzu_observability
DOCKER_IMAGE?=kubernetes-operator-snapshot

GO_IMPORTS_BIN:=$(if $(which goimports),$(which goimports),$(GOPATH)/bin/goimports)
SEMVER_CLI_BIN:=$(if $(which semver-cli),$(which semver-cli),$(GOPATH)/bin/semver-cli)

ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif

RELEASE_VERSION?=$(shell cat ./release/OPERATOR_VERSION)
VERSION?=$(shell semver-cli inc patch $(RELEASE_VERSION))$(VERSION_POSTFIX)
IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
NS=observability-system
LDFLAGS=-X main.version=$(VERSION)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
REPO_DIR=$(shell git rev-parse --show-toplevel)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

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
$(GO_IMPORTS_BIN):
	@(cd $(REPO_DIR)/..; CGO_ENABLED=0 go install golang.org/x/tools/cmd/goimports@latest)

semver-cli: $(SEMVER_CLI_BIN)

$(SEMVER_CLI_BIN):
	@(cd $(REPO_DIR)/..; CGO_ENABLED=0 go install github.com/davidrjonas/semver-cli@latest)

.PHONY: manifests
manifests: controller-gen config/crd/bases/wavefront.com_wavefronts.yaml

config/crd/bases/wavefront.com_wavefronts.yaml: api/v1alpha1/wavefront_types.go controllers/wavefront_controller.go
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen api/v1alpha1/zz_generated.deepcopy.go

api/v1alpha1/zz_generated.deepcopy.go: hack/boilerplate.go.txt api/v1alpha1/wavefront_types.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: $(GO_IMPORTS_BIN)
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs goimports -w

.PHONY: checkfmt
checkfmt: $(GO_IMPORTS_BIN)
	@if [ $$(goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*") | wc -l) -gt 0 ]; then \
		echo $$'\e[31mgoimports FAILED!!!\e[0m'; \
		goimports -d $$(find . -type f -name '*.go' -not -path "./vendor/*"); \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

GOOS?=$(go env GOOS)
GOARCH?=$(go env GOARCH)

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -ldflags "$(LDFLAGS)" -o build/$(GOOS)/$(GOARCH)/manager main.go
	rm -rf build/$(GOOS)/$(GOARCH)/deploy
	cp -r deploy build/$(GOOS)/$(GOARCH)
	cp open_source_licenses.txt build/

.PHONY: clean
clean:
	rm -rf bin
	rm -rf build

.PHONY: docker-build
docker-build: $(SEMVER_CLI_BIN) ## Build docker image with the manager.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build -o fmt -o vet
	docker build -t ${IMG} -f Dockerfile build

BUILDER_SUFFIX=$(shell echo $(PREFIX) | cut -d '/' -f1)

.PHONY: docker-xplatform-build
docker-xplatform-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build -o fmt -o vet
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 make build -o fmt -o vet
	docker buildx create --use --node wavefront_operator_builder_$(BUILDER_SUFFIX)
	docker buildx build --platform linux/amd64,linux/arm64 --push --pull -t ${IMG} -f Dockerfile build

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = true
endif

copy-base-patches:
	cp config/manager/patches-base.yaml config/manager/patches.yaml

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen

.PHONY: controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.7)

KUBE_LINTER = $(shell pwd)/bin/kube-linter
.PHONY: install-kube-linter
install-kube-linter: ## Download kube-linter locally if necessary.
	$(call go-get-tool,$(KUBE_LINTER),golang.stackrox.io/kube-linter/cmd/kube-linter@v0.4.0)

KUBE_SCORE = $(shell pwd)/bin/kube-score
.PHONY: install-kube-score
install-kube-score: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUBE_SCORE),github.com/zegl/kube-score/cmd/kube-score@v1.14.0)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOOS= GOARCH= GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef

OPERATOR_BUILD_DIR:=$(REPO_DIR)/build/operator
OPERATOR_BUILD_YAML:=$(OPERATOR_BUILD_DIR)/wavefront-operator.yaml
DEPLOY_SOURCE?=kind

.PHONY: kubernetes-yaml
kubernetes-yaml: manifests kustomize
	mkdir -p $(OPERATOR_BUILD_DIR)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default > $(OPERATOR_BUILD_YAML)
	cp $(REPO_DIR)/hack/build/kustomization.yaml $(OPERATOR_BUILD_DIR)

.PHONY: rc-kubernetes-yaml
rc-kubernetes-yaml:
	mkdir -p $(OPERATOR_BUILD_DIR)
	curl https://raw.githubusercontent.com/wavefrontHQ/wavefront-operator-for-kubernetes/$(OPERATOR_YAML_RC_SHA)/wavefront-operator-$(GIT_BRANCH).yaml \
		-o $(OPERATOR_BUILD_YAML)
	cp $(REPO_DIR)/hack/build/kustomization.yaml $(OPERATOR_BUILD_DIR)

.PHONY: xplatform-kubernetes-yaml
xplatform-kubernetes-yaml: docker-xplatform-build copy-base-patches kubernetes-yaml

.PHONY: released-kubernetes-yaml
released-kubernetes-yaml: copy-base-patches kubernetes-yaml
	cp $(OPERATOR_BUILD_YAML) $(REPO_DIR)/deploy/kubernetes/wavefront-operator.yaml

.PHONY: kind-kubernetes-yaml
kind-kubernetes-yaml: docker-build copy-kind-patches kubernetes-yaml
	kind load docker-image $(IMG)

.PHONY: copy-kind-patches
copy-kind-patches:
	cp config/manager/patches-kind.yaml config/manager/patches.yaml

.PHONY: deploy
deploy: $(DEPLOY_SOURCE)-kubernetes-yaml
	kubectl apply -k $(OPERATOR_BUILD_DIR)
	kubectl create -n $(NS) secret generic wavefront-secret --from-literal token=$(WAVEFRONT_TOKEN) || true

.PHONY: undeploy
undeploy: $(DEPLOY_SOURCE)-kubernetes-yaml
	kubectl delete --ignore-not-found=$(ignore-not-found) -n $(NS) secret wavefront-secret || true
	kubectl delete --ignore-not-found=$(ignore-not-found) -k $(OPERATOR_BUILD_DIR) || true

.PHONY: integration-test
integration-test: install-kube-score install-kube-linter undeploy deploy
	(cd $(REPO_DIR)/hack/test && ./run-e2e-tests.sh -t $(WAVEFRONT_TOKEN) -n $(CONFIG_CLUSTER_NAME))

.PHONY: clean-cluster
clean-cluster:
	(cd $(REPO_DIR)/hack/test && ./clean-cluster.sh)

#----- KIND ----#
.PHONY: nuke-kind
nuke-kind:
	kind delete cluster
	kind create cluster

#----- GKE -----#
GCP_PROJECT?=wavefront-gcp-dev

gke-connect-to-cluster: gke-cluster-name-check
	gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone us-central1-c --project $(GCP_PROJECT)

gke-cluster-name-check:
	@if [ -z ${GKE_CLUSTER_NAME} ]; then echo "Need to set GKE_CLUSTER_NAME" && exit 1; fi

#----- AKS -----#
aks-subscription-id-check:
	@if [ -z ${AKS_SUBSCRIPTION_ID} ]; then echo "Need to set AKS_SUBSCRIPTION_ID" && exit 1; fi

aks-connect-to-cluster: aks-subscription-id-check
	az account set --subscription

#----- EKS -----#
ECR_REPO_PREFIX=tobs/k8s/saas
WAVEFRONT_DEV_AWS_ACC_ID=095415062695
AWS_PROFILE=wavefront-dev
AWS_REGION=us-west-2
ECR_ENDPOINT=$(WAVEFRONT_DEV_AWS_ACC_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
COLLECTOR_ECR_REPO=$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector
TEST_PROXY_ECR_REPO=$(ECR_REPO_PREFIX)/test-proxy

ecr-host:
	echo $(ECR_ENDPOINT)/$(ECR_REPO_PREFIX)/wavefront-kubernetes-collector

docker-login-eks:
	@aws ecr get-login-password --region $(AWS_REGION) --profile $(AWS_PROFILE) |  docker login --username AWS --password-stdin $(ECR_ENDPOINT)

target-eks: docker-login-eks
	@aws eks --region $(AWS_REGION) update-kubeconfig --name k8s-saas-team-dev --profile $(AWS_PROFILE)

# create a new branch from main
# usage: make branch JIRA=XXXX OR make branch NAME=YYYY
branch:
	$(eval NAME := $(if $(JIRA),K8SAAS-$(JIRA),$(NAME)))
	@if [ -z "$(NAME)" ]; then \
		echo "usage: make branch JIRA=XXXX OR make branch NAME=YYYY"; \
		exit 1; \
  	fi
	git stash
	git checkout main
	git pull
	git checkout -b $(NAME)

git-rebase:
	git fetch origin
	git rebase origin/main
	git log --oneline -n 10