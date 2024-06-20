<!--
# SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
# Copyright 2019 free5GC.org

SPDX-License-Identifier: Apache-2.0
-->
[![Go Report Card](https://goreportcard.com/badge/github.com/omec-project/webconsole)](https://goreportcard.com/report/github.com/omec-project/webconsole)

# WebConsole

Webconsole is used as a configuration service in SD-Core. It has following
features Configuration Service provides APIs for subscriber management.

1. It provides APIs for Network Slice management.
2. It  communicates with 4G as well as 5G network functions on the southbound interface.
3. It does configuration translation wherever required and also stores the subscription in mongoDB.
4. 5G clients can connect & get complete configuration copy through grpc interface.
5. 4G clients communicate with Webconsole through REST interface

# UI

Webconsole can optionally serve static files, which constitute the frontend part of the application.

This functionality is not mandatory. The frontend files must be provided and placed in `webconsole/ui/frontend_files` before compilation, as they are not included in the webconsole codebase.

To compile the webconsole including the UI option:
```
make webconsole-ui
```
or
```
go build --tags ui -o <output-file> ./server.go
```

Once the webconsole is running, the UI will be seen on:
```
http://<webconsole-ip>:5000/
```

## Webconsole Architecture diagram

![Architecture](/docs/images/architecture1.png)

## Upcoming Features

1. Supporting dedicated flow QoS APIs
2. Removal of Subscription to trigger 3gpp call flows
