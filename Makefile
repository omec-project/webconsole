# Copyright 2019-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#
#

PROJECT_NAME             := sdcore
VERSION                  ?= $(shell cat ./VERSION)

## Docker related
DOCKER_REGISTRY          ?=
DOCKER_REPOSITORY        ?=
DOCKER_TAG               ?= ${VERSION}
DOCKER_IMAGENAME         := ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}${PROJECT_NAME}:${DOCKER_TAG}
DOCKER_BUILDKIT          ?= 1
DOCKER_BUILD_ARGS        ?=

## Docker labels. Only set ref and commit date if committed
DOCKER_LABEL_VCS_URL     ?= $(shell git remote get-url $(shell git remote))
DOCKER_LABEL_VCS_REF     ?= $(shell git diff-index --quiet HEAD -- && git rev-parse HEAD || echo "unknown")
DOCKER_LABEL_COMMIT_DATE ?= $(shell git diff-index --quiet HEAD -- && git show -s --format=%cd --date=iso-strict HEAD || echo "unknown" )
DOCKER_LABEL_BUILD_DATE  ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DOCKER_TARGETS           ?= builder webui

# https://docs.docker.com/engine/reference/commandline/build/#specifying-target-build-stage---target
GO_BIN_PATH = bin
GO_SRC_PATH = ./
C_BUILD_PATH = build
ROOT_PATH = $(shell pwd)

WEBCONSOLE = webconsole

WEBCONSOLE_GO_FILES = $(shell find ./ -name "*.go" ! -name "*_test.go")

VERSION = $(shell git describe --tags)
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
WEBCONSOLE_COMMIT_HASH = $(shell git submodule status | grep $(WEBCONSOLE) | awk '{print $$(1)}' | cut -c1-8)
WEBCONSOLE_COMMIT_TIME = $(shell git log --pretty="%ai" -1 | awk '{time=$$(1)"T"$$(2)"Z"; print time}')
WEBCONSOLE_LDFLAGS = -X github.com/free5gc/version.VERSION=$(VERSION) \
                     -X github.com/free5gc/version.BUILD_TIME=$(BUILD_TIME) \
                     -X github.com/free5gc/version.COMMIT_HASH=$(WEBCONSOLE_COMMIT_HASH) \
                     -X github.com/free5gc/version.COMMIT_TIME=$(WEBCONSOLE_COMMIT_TIME)

.PHONY: $(NF) clean docker-build docker-push

.DEFAULT_GOAL: nfs

all: $(WEBCONSOLE)

$(WEBCONSOLE): $(GO_BIN_PATH)/$(WEBCONSOLE)

$(GO_BIN_PATH)/$(WEBCONSOLE): server.go  $(WEBCONSOLE_GO_FILES)
	@echo "Start building $(@F)...."
	go build -ldflags "$(WEBCONSOLE_LDFLAGS)" -o $(ROOT_PATH)/$@ ./server.go

vpath %.go $(addprefix $(GO_SRC_PATH)/, $(GO_NF))

clean:
	rm -rf $(WEBCONSOLE)/$(GO_BIN_PATH)/$(WEBCONSOLE)

docker-build:
	@go mod vendor
	for target in $(DOCKER_TARGETS); do \
                DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build  $(DOCKER_BUILD_ARGS) \
                        --target $$target \
                        --tag ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}5gc-$$target:${DOCKER_TAG} \
                        --build-arg org_label_schema_version="${VERSION}" \
                        --build-arg org_label_schema_vcs_url="${DOCKER_LABEL_VCS_URL}" \
                        --build-arg org_label_schema_vcs_ref="${DOCKER_LABEL_VCS_REF}" \
                        --build-arg org_label_schema_build_date="${DOCKER_LABEL_BUILD_DATE}" \
                        --build-arg org_opencord_vcs_commit_date="${DOCKER_LABEL_COMMIT_DATE}" \
                        . \
                        || exit 1; \
        done
	rm -rf vendor

docker-push:
	for target in $(DOCKER_TARGETS); do \
                docker push ${DOCKER_REGISTRY}${DOCKER_REPOSITORY}5gc-$$target:${DOCKER_TAG}; \
        done


