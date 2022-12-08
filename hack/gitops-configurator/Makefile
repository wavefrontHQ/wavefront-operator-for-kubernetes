ifeq ($(origin VERSION_POSTFIX), undefined)
VERSION_POSTFIX:=-alpha-$(shell whoami)-$(shell date +"%y%m%d%H%M%S")
endif

PREFIX?=projects.registry.vmware.com/tanzu_observability_keights_saas
DOCKER_IMAGE?=gitops-configurator
VERSION?=$(shell cat ./release/CONFIGURATOR_VERSION)$(VERSION_POSTFIX)

IMG?=$(PREFIX)/$(DOCKER_IMAGE):$(VERSION)
LOCAL_IMG=$(PREFIX)/$(DOCKER_IMAGE):latest

NS?=observability-system

.PHONY: all
all: docker-build

.PHONY: ssh-hosts-and-key
ssh-hosts-and-key:
	kubectl create -n $(NS) secret generic ssh-hosts-and-key --from-file known_hosts=$(HOME)/.ssh/known_hosts --from-file id_ed25519=$(HOME)/.ssh/id_ed25519 || true

.PHONY: docker-build
docker-build:
	docker build -t $(IMG) -t $(LOCAL_IMG) .

.PHONY: docker-push
docker-push: docker-build
	kind load docker-image $(LOCAL_IMG) --name kind
	#docker push ${IMG}

.PHONY: deploy
deploy: ssh-hosts-and-key docker-push
	kubectl apply -f deploy/

.PHONY: redeploy
redeploy: undeploy deploy

.PHONY: undeploy
undeploy:
	kubectl delete -f deploy/

img:
	@echo $(IMG)

limg:
	@echo $(LOCAL_IMG)