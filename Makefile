# Copyright 2019-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#
#
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

.PHONY: $(NF) clean

.DEFAULT_GOAL: nfs

all: $(WEBCONSOLE)

$(WEBCONSOLE): $(GO_BIN_PATH)/$(WEBCONSOLE)

$(GO_BIN_PATH)/$(WEBCONSOLE): server.go  $(WEBCONSOLE_GO_FILES)
	@echo "Start building $(@F)...."
	cd frontend && \
	yarn install && \
	yarn build && \
	rm -rf ../public && \
	cp -R build ../public
#	cd $(WEBCONSOLE) && \
#	go build -ldflags "$(WEBCONSOLE_LDFLAGS)" -o $(ROOT_PATH)/$@ ./server.go
#


vpath %.go $(addprefix $(GO_SRC_PATH)/, $(GO_NF))

clean:
	rm -rf $(WEBCONSOLE)/$(GO_BIN_PATH)/$(WEBCONSOLE)

