ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif

PREFIX?=projects.registry.vmware.com/tanzu_observability_keights_saas
DOCKER_IMAGE?=gitops-configurator
VERSION?=$(shell cat ./release/CONFIGURATOR_VERSION)$(VERSION_POSTFIX)

IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
LOCAL_IMG=$(PREFIX)/$(DOCKER_IMAGE):latest

.PHONY: all
all: docker-build

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) -t $(LOCAL_IMG) .

.PHONY: docker-push
docker-push: docker-build
	kind load docker-image $(LOCAL_IMG) --name kind
	#docker push ${IMG}

.PHONY: deploy
deploy: docker-push
	kubectl apply -f deploy/

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/

img:
	echo $(IMG)