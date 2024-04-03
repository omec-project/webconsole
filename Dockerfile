# SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
# SPDX-FileCopyrightText: 2024 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.22.2-bookworm AS builder

LABEL maintainer="Aether SD-Core <dev@lists.aetherproject.org>"

RUN apt-get update && \
    apt-get -y install --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    gcc \
    cmake \
    autoconf \
    libtool \
    pkg-config \
    libmnl-dev \
    libyaml-dev \
    unzip && \
    apt-get clean

WORKDIR $GOPATH/src/webconsole
COPY . .
RUN make all && \
    CGO_ENABLED=0 go build -a -installsuffix nocgo -o webconsole -x server.go

FROM alpine:3.19 as webui

LABEL description="ONF open source 5G Core Network" \
    version="Stage 3"

ARG DEBUG_TOOLS


# Install debug tools ~85MB (if DEBUG_TOOLS is set to true)
RUN if [ "$DEBUG_TOOLS" = "true" ]; then \
        apk update && apk add --no-cache -U vim strace net-tools curl netcat-openbsd bind-tools; \
        fi

# Set working dir
WORKDIR /free5gc/webconsole

# Copy executable and default certs
COPY --from=builder /go/src/webconsole/webconsole .
