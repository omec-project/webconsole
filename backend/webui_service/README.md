<!--
SPDX-License-Identifier: Apache-2.0
Copyright 2024 Canonical Ltd.
-->

# Swagger UI Service

The webconsole can optionally serve a [swagger UI](https://github.com/swaggo/swag).
The API documentation is automatically generated from code annotations.

To generate the swagger UI files run:
```
swag init -g backend/webui_service/swagger_ui_service.go --outputTypes go
```
The `docs.go` file will automatically be created in `webconsole/docs`

The swagger UI operations are executed by default on `localhost`. If the webconsole server runs remotely, set the following environment variable.
```
export WEBUI_ENDPOINT=<webconsole-ip>:5000
```
Build the webconsole including the UI option:
```
make webconsole-ui
```
Access the swagger UI at:
```
http://<webconsole-ip>:5000/swagger/index.html
```
The `doc.json` file, which can be integrated in a frontend implementation, is available at:
```
http://<webconsole-ip>:5000/swagger/doc.json
```