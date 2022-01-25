# Copyright 2019-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.14.4-stretch AS builder

LABEL maintainer="ONF <omec-dev@opennetworking.org>"

#RUN apt remove cmdtest yarn
RUN apt-get update
RUN apt-get -y install apt-transport-https ca-certificates
RUN apt-get -y upgrade
RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg > pubkey.gpg
RUN apt-key add pubkey.gpg
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" |  tee /etc/apt/sources.list.d/yarn.list
RUN apt-get update
RUN apt-get -y install gcc cmake autoconf libtool pkg-config libmnl-dev libyaml-dev  nodejs yarn unzip
RUN apt-get clean
ENV PROTOC_ZIP=protoc-3.14.0-linux-x86_64.zip
RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/${PROTOC_ZIP}
#RUN unzip -o ${PROTOC_ZIP} -d ./proto 
#RUN chmod 755 -R ./proto/bin
#ENV BASE=/usr
# Copy into path
#RUN cp ./proto/bin/protoc ${BASE}/bin/
#RUN cp -R ./proto/include/* ${BASE}/include/

RUN cd $GOPATH/src && mkdir -p webconsole
COPY . $GOPATH/src/webconsole
#RUN cd $GOPATH/src/webconsole/proto \
#    && go get -u google.golang.org/protobuf/cmd/protoc-gen-go \
#    && go install google.golang.org/protobuf/cmd/protoc-gen-go \
#    && go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc \
#    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc \
#    && protoc -I ./ --go_out=. config.proto \
#    && protoc -I ./ --go-grpc_out=. config.proto

RUN cd $GOPATH/src/webconsole \
    && make all \
    && CGO_ENABLED=0 go build -a -installsuffix nocgo -o webconsole -x server.go

FROM alpine:3.8 as webui

LABEL description="ONF open source 5G Core Network" \
    version="Stage 3"

ARG DEBUG_TOOLS

# Install debug tools ~ 100MB (if DEBUG_TOOLS is set to true)
RUN apk update
RUN apk add -U vim strace net-tools curl netcat-openbsd bind-tools

# Set working dir
WORKDIR /free5gc
RUN mkdir -p webconsole/

# Copy executable and default certs
COPY --from=builder /go/src/webconsole/webconsole ./webconsole
COPY --from=builder /go/src/webconsole/public ./webconsole/public
