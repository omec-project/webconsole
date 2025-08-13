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

![Architecture](/docs/images/architecture.png)

The webconsole has two services: **WebUI** and **NF Config**.

### WebUI Service

The **WebUI** service runs by default on port `5000`. This port can be configured using
the `cfgport` parameter in the configuration file.
It provides a HTTP REST API that allows configuration of **network slices** and **subscribers** in the network.

### NF Config Service
The **NF Config** service runs on port `5001`. It is a HTTP REST service that provides configuration
to the Network Functions (NFs) upon request.

- The configuration is stored in-memory.
- It is loaded from the database at startup.
- It is updated whenever a successful write request is made through the WebUI service.

To run the service over HTTPS, add the following parameters to the configuration file:

```yaml
configuration:
...
  nfconfig-tls:
    pem: <path-to-cert.pem>
    key: <path-to-key.pem>
```

There are six endpoints exposed by this service.

| Endpoint Name        | NF                  | HTTP Method | Path                           | Body  | Response          |
|----------------------|---------------------|-------------|--------------------------------|-------|--------------------------|
| Access and Mobility  | AMF                 | GET         | `/nfconfig/access-mobility`    | None  | [List of AccessAndMobility](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_access_and_mobility.go) |
| PLMN ID              | AUSF NRF UDM UDR    | GET         | `/nfconfig/plmn`               | None  | [List of Plmns](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_plmn_id.go)             |
| PLMN-SNSSAI          | NSSF                | GET         | `/nfconfig/plmn-snssai`        | None  | [List of Plmn-Snssai ](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_plmn_snssai.go)         |
| Policy Control       | PCF                 | GET         | `/nfconfig/policy-control`     | None  | [List of PolicyControl](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_policy_control.go)      |
| Session Management   | SMF                 | GET         | `/nfconfig/session-management` | None  | [List of Session Management](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_session_management.go)  |
| IMSI QoS             | PCF                 | GET         | `/nfconfig/qos/{dnn}/{imsi}`   | None  | [List of ImsiQoS](https://github.com/omec-project/openapi/blob/main/nfConfigApi/model_imsi_qos.go)            |

To make modifications to the NF Config API, please refer to the
[NF config API documentation](https://github.com/omec-project/openapi/blob/main/nfConfigApi/README.md)
in the [openapi](https://github.com/omec-project/openapi) repository.

## Upcoming Features

1. Supporting dedicated flow QoS APIs
2. Removal of Subscription to trigger 3gpp call flows
