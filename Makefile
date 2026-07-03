ARTIFACT_ID=service-account-producer-sidecar
MAKEFILES_VERSION=10.10.0
VERSION=0.1.0

IMAGE_NAME=registry.cloudogu.com/k8s/${ARTIFACT_ID}

GOTAG=1.26.4
LINT_VERSION=v2.9.0
GO_MAIN_PACKAGE=./cmd/${ARTIFACT_ID}

.DEFAULT_GOAL:=build

include build/make/variables.mk
include build/make/dependencies-gomod.mk
# main package moved to cmd/; override build.mk's default (which builds ".") to point at it.
GO_BUILD_FLAGS=-mod=vendor -a -tags netgo $(LDFLAGS) -installsuffix cgo -o $(BINARY) $(GO_MAIN_PACKAGE)
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/self-update.mk
include build/make/release.mk

.PHONY: info
info:
	@echo "version informations ..."
	@echo "Version       : $(VERSION)"
	@echo "Image Name    : $(IMAGE_NAME)"
	@echo "Image         : $(IMAGE_NAME):$(VERSION)"

.PHONY: name
name:
	@echo "${IMAGE_NAME}"

.PHONY: version
version:
	@echo "${VERSION}"

.PHONY: build
build:
	docker build -t "$(IMAGE_NAME):$(VERSION)" .

.PHONY: deploy
deploy: build
	docker push "$(IMAGE_NAME):$(VERSION)"
