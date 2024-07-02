<!--
SPDX-License-Identifier: Apache-2.0
Copyright 2024 Canonical Ltd.
-->

# Dynamic Parameter Service

The Dynamic Parameter Service is a feature that serves configuration parameters from files specified by environment variables. These parameters can be accessed via REST endpoints defined within the application.

This is an optional functionality.

```
make webconsole-ui
```

## Dynamic Parameters

Currently, the service supports two types of configuration parameters:

### List of gNBs

Endpoint: /config/parameter/gnb
Environment Variable: GNB_CONFIG_PATH

This endpoint serves the list of gNBs configured in the file pointed to by the GNB_CONFIG_PATH environment variable.

### List of UPFs

Endpoint: /config/parameter/upf
Environment Variable: UPF_CONFIG_PATH

This endpoint serves the list of UPFs (User Plane Function) configured in the file pointed to by the UPF_CONFIG_PATH environment variable.

## Example

Assuming the service is running locally and the environment variables are set:

GNB_CONFIG_PATH=/path/to/gnb_list.json
UPF_CONFIG_PATH=/path/to/upf_list.json

You can access the following endpoints:

GET http://localhost:5000/config/parameter/gnb
GET http://localhost:5000/config/parameter/upf

These endpoints will return JSON responses containing the respective lists of gNBs and UPFs configured in the files.
