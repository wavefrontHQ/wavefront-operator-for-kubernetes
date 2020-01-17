PREFIX?=wavefronthq
DOCKER_IMAGE=wavefront-operator-for-kubernetes
ARCH?=amd64
OUT_DIR?=build/_output/bin
GOLANG_VERSION?=1.13

BINARY_NAME=wavefront-operator

ifndef TEMP_DIR
TEMP_DIR:=$(shell mktemp -d /tmp/wavefront.XXXXXX)
endif

VERSION?=0.9.0
GIT_COMMIT:=$(shell git rev-parse --short HEAD)

REPO_DIR:=$(shell pwd)

# for testing, the built image will also be tagged with this name provided via an environment variable
OVERRIDE_IMAGE_NAME?=${OPERATOR_TEST_IMAGE}

LDFLAGS=-w -X main.ver=$(VERSION) -X main.commit=$(GIT_COMMIT)

all: build

fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs gofmt -s -w

tests:
	go clean -testcache
	go test -v -race ./...

build: clean fmt
	go vet -composites=false ./...
	GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/$(BINARY_NAME) ./cmd/manager/

container: clean
	# Run build in a container in order to have reproducible builds
	docker run --rm -v $(TEMP_DIR):/build -v $(REPO_DIR):/go/wavefront-operator-for-kubernetes -w /go/wavefront-operator-for-kubernetes golang:$(GOLANG_VERSION) /bin/bash -c "\
		GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags \"$(LDFLAGS)\" -o /build/$(OUT_DIR)/$(BINARY_NAME) ./cmd/manager/"

	cp -R build/* $(TEMP_DIR)
	ls $(TEMP_DIR)
	docker build --pull -t $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(TEMP_DIR)
ifneq ($(OVERRIDE_IMAGE_NAME),)
	docker tag $(PREFIX)/$(DOCKER_IMAGE):$(VERSION) $(OVERRIDE_IMAGE_NAME)
endif

clean:
	rm -f $(OUT_DIR)/$(BINARY_NAME)

.PHONY: all fmt container clean
