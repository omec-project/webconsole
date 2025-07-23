<!--
SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
Copyright 2019 free5GC.org
SPDX-FileCopyrightText: 2024 Canonical Ltd.
SPDX-License-Identifier: Apache-2.0
-->
[![Go Report Card](https://goreportcard.com/badge/github.com/omec-project/webconsole)](https://goreportcard.com/report/github.com/omec-project/webconsole)

# WebConsole

Webconsole is used as a configuration service in SD-Core. It has following
features Configuration Service provides APIs for subscriber management.

1. It provides APIs for Network Slice management.
2. It  communicates with 5G network functions on the southbound interface.
3. It does configuration translation wherever required and also stores the subscription in mongoDB.
4. 5G clients reach the webconsole on port 5001 to get a configuration copy.

## UI

Webconsole can optionally serve static files, which constitute the frontend part of the application.

To build webui with a frontend, place the static files under `webconsole/ui/frontend_files` before compilation.

To build the webconsole including the UI option:
```
make webconsole-ui
```

Access the UI at:
```
http://<webconsole-ip>:5000/
```

An example static file has been placed in the `webconsole/ui/frontend_files` directory.

## Authentication and Authorization

The authentication and authorization feature ensures that only verified and authorized users can access the webui resources and interact with the system.

This is an optional feature, disabled by default. For more details, refer to this [file](backend/auth/README.md).

##  MongoDB Transaction Support

This application requires a MongoDB deployment configured to support transactions,
such as a replica set or a sharded cluster. Standalone MongoDB instances do not
support transactions and will prevent the application from starting. Please ensure
your MongoDB instance is properly set up for transactions. For detailed configuration
instructions, see the [MongoDB Replica Set Documentation](https://www.mongodb.com/docs/kubernetes-operator/current/tutorial/deploy-replica-set/).

## Webconsole Architecture diagram

![Architecture](/docs/images/architecture1.png)

## Upcoming Features

1. Supporting dedicated flow QoS APIs
2. Removal of Subscription to trigger 3gpp call flows
