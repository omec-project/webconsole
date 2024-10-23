// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd

// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "OMEC Project - Webconsole",
            "url": "https://github.com/omec-project/webconsole"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/subscriber/": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the list of subscribers",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Subscribers"
                ],
                "responses": {
                    "200": {
                        "description": "List of subscribers. Null if there are no subscribers",
                        "schema": {
                            "$ref": "#/definitions/configmodels.SubsListIE"
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error retrieving subscribers"
                    }
                }
            }
        },
        "/api/subscriber/{imsi}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Get subscriber by IMSI (UE ID)",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Subscribers"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "example": "imsi-208930100007487",
                        "description": "IMSI (UE ID)",
                        "name": "imsi",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Subscriber"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Subscriber not found"
                    },
                    "500": {
                        "description": "Error retrieving subscriber"
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create subscriber by IMSI (UE ID)",
                "tags": [
                    "Subscribers"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "IMSI (UE ID)",
                        "name": "imsi",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": " ",
                        "name": "content",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/configmodels.SubsOverrideData"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Subscriber created"
                    },
                    "400": {
                        "description": "Invalid subscriber content"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error creating subscriber"
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Delete an existing subscriber",
                "tags": [
                    "Subscribers"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "IMSI (UE ID)",
                        "name": "imsi",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Subscriber deleted successfully"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error deleting subscriber"
                    }
                }
            }
        },
        "/config/v1/account/": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the list of user accounts",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "User Accounts"
                ],
                "responses": {
                    "200": {
                        "description": "List of user accounts",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/configmodels.GetUserAccountResponse"
                            }
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Error retrieving user accounts"
                    }
                }
            }
        },
        "/config/v1/account/{username}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the user account",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "User Accounts"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "Username of the user account",
                        "name": "username",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User account",
                        "schema": {
                            "$ref": "#/definitions/configmodels.GetUserAccountResponse"
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "User account not found. Or Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Error retrieving user account"
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create a new user account",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "User Accounts"
                ],
                "parameters": [
                    {
                        "description": "Username and password",
                        "name": "params",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/configmodels.CreateUserAccountParams"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User account created"
                    },
                    "400": {
                        "description": "Bad request"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Failed to create the user account"
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Delete an existing user account",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "User Accounts"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "Username of the user account",
                        "name": "username",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User account deleted"
                    },
                    "400": {
                        "description": "Failed to delete the user account"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "User account not found. Or Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Failed to delete the user account"
                    }
                }
            }
        },
        "/config/v1/account/{username}/change_password": {
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create a new user account",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "User Accounts"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "Username",
                        "name": "username",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Username and password",
                        "name": "params",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/configmodels.ChangePasswordParams"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Password changed"
                    },
                    "400": {
                        "description": "Bad request"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Failed to update the user account"
                    }
                }
            }
        },
        "/config/v1/device-group/": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the list of device groups",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Device Groups"
                ],
                "responses": {
                    "200": {
                        "description": "List of device group names",
                        "schema": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            }
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error retrieving device groups"
                    }
                }
            }
        },
        "/config/v1/device-group/{deviceGroupName}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the device group",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Device Groups"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "deviceGroupName",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Device group",
                        "schema": {
                            "$ref": "#/definitions/configmodels.DeviceGroups"
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Device group not found"
                    },
                    "500": {
                        "description": "Error retrieving device group"
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create a new device group",
                "tags": [
                    "Device Groups"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "deviceGroupName",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": " ",
                        "name": "content",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/configmodels.DeviceGroups"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Device group created"
                    },
                    "400": {
                        "description": "Invalid device group content"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error creating device group"
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Delete an existing device group",
                "tags": [
                    "Device Groups"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "deviceGroupName",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Device group deleted successfully"
                    },
                    "400": {
                        "description": "Invalid device group name provided"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error deleting device group"
                    }
                }
            }
        },
        "/config/v1/network-slice/": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the list of network slices",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Network Slices"
                ],
                "responses": {
                    "200": {
                        "description": "List of network slice names",
                        "schema": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            }
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error retrieving network slices"
                    }
                }
            }
        },
        "/config/v1/network-slice/{sliceName}": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Return the network slice",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Network Slices"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "sliceName",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Network slice",
                        "schema": {
                            "$ref": "#/definitions/configmodels.Slice"
                        }
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Network slices not found"
                    },
                    "500": {
                        "description": "Error retrieving network slice"
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Create a new network slice",
                "tags": [
                    "Network Slices"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "sliceName",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": " ",
                        "name": "content",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/configmodels.Slice"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Network slice created"
                    },
                    "400": {
                        "description": "Invalid network slice content"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error creating network slice"
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "description": "Delete an existing network slice",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Network Slices"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": " ",
                        "name": "sliceName",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "202": {
                        "description": "Network slice deleted successfully"
                    },
                    "400": {
                        "description": "Invalid network slice name provided"
                    },
                    "401": {
                        "description": "Authorization failed"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Error deleting network slice"
                    }
                }
            }
        },
        "/login": {
            "post": {
                "description": "Log in. Only available if enableAuthentication is enabled.",
                "tags": [
                    "Auth"
                ],
                "parameters": [
                    {
                        "description": " ",
                        "name": "loginParams",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/auth.LoginParams"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Authorization token",
                        "schema": {
                            "$ref": "#/definitions/auth.LoginResponse"
                        }
                    },
                    "400": {
                        "description": "Bad request"
                    },
                    "401": {
                        "description": "Authentication failed"
                    },
                    "404": {
                        "description": "Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/status": {
            "get": {
                "description": "Get Status. Only available if enableAuthentication is enabled.",
                "tags": [
                    "Auth"
                ],
                "responses": {
                    "200": {
                        "description": "Webui status",
                        "schema": {
                            "$ref": "#/definitions/auth.StatusResponse"
                        }
                    },
                    "404": {
                        "description": "Page not found if enableAuthentication is disabled"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        }
    },
    "definitions": {
        "auth.LoginParams": {
            "type": "object",
            "properties": {
                "password": {
                    "type": "string"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "auth.LoginResponse": {
            "type": "object",
            "properties": {
                "token": {
                    "type": "string"
                }
            }
        },
        "auth.StatusResponse": {
            "type": "object",
            "properties": {
                "initialized": {
                    "type": "boolean"
                }
            }
        },
        "configmodels.ChangePasswordParams": {
            "type": "object",
            "properties": {
                "password": {
                    "type": "string"
                }
            }
        },
        "configmodels.CreateUserAccountParams": {
            "type": "object",
            "properties": {
                "password": {
                    "type": "string"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "configmodels.DeviceGroups": {
            "type": "object",
            "properties": {
                "group-name": {
                    "type": "string"
                },
                "imsis": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "ip-domain-expanded": {
                    "$ref": "#/definitions/configmodels.DeviceGroupsIpDomainExpanded"
                },
                "ip-domain-name": {
                    "type": "string"
                },
                "site-info": {
                    "type": "string"
                }
            }
        },
        "configmodels.DeviceGroupsIpDomainExpanded": {
            "type": "object",
            "properties": {
                "dnn": {
                    "type": "string"
                },
                "dns-primary": {
                    "type": "string"
                },
                "dns-secondary": {
                    "type": "string"
                },
                "mtu": {
                    "type": "integer"
                },
                "ue-dnn-qos": {
                    "$ref": "#/definitions/configmodels.DeviceGroupsIpDomainExpandedUeDnnQos"
                },
                "ue-ip-pool": {
                    "type": "string"
                }
            }
        },
        "configmodels.DeviceGroupsIpDomainExpandedUeDnnQos": {
            "type": "object",
            "properties": {
                "bitrate-unit": {
                    "description": "data rate unit for uplink and downlink",
                    "type": "string"
                },
                "dnn-mbr-downlink": {
                    "description": "downlink data rate",
                    "type": "integer"
                },
                "dnn-mbr-uplink": {
                    "description": "uplink data rate",
                    "type": "integer"
                },
                "traffic-class": {
                    "description": "QCI/QFI for the traffic",
                    "allOf": [
                        {
                            "$ref": "#/definitions/configmodels.TrafficClassInfo"
                        }
                    ]
                }
            }
        },
        "configmodels.GetUserAccountResponse": {
            "type": "object",
            "properties": {
                "role": {
                    "type": "integer"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "configmodels.Slice": {
            "type": "object",
            "properties": {
                "application-filtering-rules": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/configmodels.SliceApplicationFilteringRules"
                    }
                },
                "site-device-group": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "site-info": {
                    "$ref": "#/definitions/configmodels.SliceSiteInfo"
                },
                "slice-id": {
                    "$ref": "#/definitions/configmodels.SliceSliceId"
                },
                "sliceName": {
                    "type": "string"
                }
            }
        },
        "configmodels.SliceApplicationFilteringRules": {
            "type": "object",
            "properties": {
                "action": {
                    "description": "action",
                    "type": "string"
                },
                "app-mbr-downlink": {
                    "type": "integer"
                },
                "app-mbr-uplink": {
                    "type": "integer"
                },
                "bitrate-unit": {
                    "description": "data rate unit for uplink and downlink",
                    "type": "string"
                },
                "dest-port-end": {
                    "description": "port range end",
                    "type": "integer"
                },
                "dest-port-start": {
                    "description": "port range start",
                    "type": "integer"
                },
                "endpoint": {
                    "description": "Application Desination IP or network",
                    "type": "string"
                },
                "priority": {
                    "description": "priority",
                    "type": "integer"
                },
                "protocol": {
                    "description": "protocol",
                    "type": "integer"
                },
                "rule-name": {
                    "description": "Rule name",
                    "type": "string"
                },
                "rule-trigger": {
                    "type": "string"
                },
                "traffic-class": {
                    "$ref": "#/definitions/configmodels.TrafficClassInfo"
                }
            }
        },
        "configmodels.SliceSiteInfo": {
            "type": "object",
            "properties": {
                "gNodeBs": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/configmodels.SliceSiteInfoGNodeBs"
                    }
                },
                "plmn": {
                    "$ref": "#/definitions/configmodels.SliceSiteInfoPlmn"
                },
                "site-name": {
                    "description": "Unique name per Site.",
                    "type": "string"
                },
                "upf": {
                    "description": "UPF which belong to this slice",
                    "type": "object",
                    "additionalProperties": true
                }
            }
        },
        "configmodels.SliceSiteInfoGNodeBs": {
            "type": "object",
            "properties": {
                "name": {
                    "type": "string"
                },
                "tac": {
                    "description": "unique tac per gNB. This should match gNB configuration.",
                    "type": "integer"
                }
            }
        },
        "configmodels.SliceSiteInfoPlmn": {
            "type": "object",
            "properties": {
                "mcc": {
                    "type": "string"
                },
                "mnc": {
                    "type": "string"
                }
            }
        },
        "configmodels.SliceSliceId": {
            "type": "object",
            "properties": {
                "sd": {
                    "description": "Slice differntiator.",
                    "type": "string"
                },
                "sst": {
                    "description": "Slice Service Type",
                    "type": "string"
                }
            }
        },
        "configmodels.SubsListIE": {
            "type": "object",
            "properties": {
                "plmnID": {
                    "type": "string"
                },
                "ueId": {
                    "type": "string"
                }
            }
        },
        "configmodels.SubsOverrideData": {
            "type": "object",
            "properties": {
                "key": {
                    "type": "string"
                },
                "opc": {
                    "type": "string"
                },
                "plmnID": {
                    "type": "string"
                },
                "sequenceNumber": {
                    "type": "string"
                }
            }
        },
        "configmodels.TrafficClassInfo": {
            "type": "object",
            "properties": {
                "arp": {
                    "description": "Traffic class priority",
                    "type": "integer"
                },
                "name": {
                    "description": "Traffic class name",
                    "type": "string"
                },
                "pdb": {
                    "description": "Packet Delay Budget",
                    "type": "integer"
                },
                "pelr": {
                    "description": "Packet Error Loss Rate",
                    "type": "integer"
                },
                "qci": {
                    "description": "QCI/5QI/QFI",
                    "type": "integer"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "description": "Enter the token in the format ` + "`" + `Bearer \u003ctoken\u003e` + "`" + `",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Webconsole API Documentation",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
