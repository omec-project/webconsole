// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * Connectivity Service Configuration
 *
 * APIs to configure connectivity service in Aether Network
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package configapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddConfigService(engine *gin.Engine) *gin.RouterGroup {
	return AddService(engine, "config/v1", routes)
}

// Index is the index handler.
func Index(c *gin.Context) {
	c.String(http.StatusOK, "Hello World!")
}

var routes = Routes{
	{
		"Index",
		http.MethodGet,
		"/",
		Index,
	},

	{
		"GetDeviceGroups",
		http.MethodGet,
		"/device-group",
		GetDeviceGroups,
	},

	{
		"GetDeviceGroupByName",
		http.MethodGet,
		"/device-group/:group-name",
		GetDeviceGroupByName,
	},

	{
		"DeviceGroupGroupNameDelete",
		http.MethodDelete,
		"/device-group/:group-name",
		DeviceGroupGroupNameDelete,
	},

	{
		"DeviceGroupGroupNamePatch",
		http.MethodPatch,
		"/device-group/:group-name",
		DeviceGroupGroupNamePatch,
	},

	{
		"DeviceGroupGroupNamePut",
		http.MethodPut,
		"/device-group/:group-name",
		DeviceGroupGroupNamePut,
	},

	{
		"DeviceGroupGroupNamePost",
		http.MethodPost,
		"/device-group/:group-name",
		DeviceGroupGroupNamePost,
	},

	{
		"GetNetworkSlices",
		http.MethodGet,
		"/network-slice",
		GetNetworkSlices,
	},

	{
		"GetNetworkSliceByName",
		http.MethodGet,
		"/network-slice/:slice-name",
		GetNetworkSliceByName,
	},

	{
		"NetworkSliceSliceNameDelete",
		http.MethodDelete,
		"/network-slice/:slice-name",
		NetworkSliceSliceNameDelete,
	},

	{
		"NetworkSliceSliceNamePost",
		http.MethodPost,
		"/network-slice/:slice-name",
		NetworkSliceSliceNamePost,
	},

	{
		"NetworkSliceSliceNamePut",
		http.MethodPut,
		"/network-slice/:slice-name",
		NetworkSliceSliceNamePut,
	},
	{
		"GetGnbs",
		http.MethodGet,
		"/inventory/gnb",
		GetGnbs,
	},
	{
		"PostGnb",
		http.MethodPost,
		"/inventory/gnb/:gnb-name",
		PostGnb,
	},
	{
		"DeleteGnb",
		http.MethodDelete,
		"/inventory/gnb/:gnb-name",
		DeleteGnb,
	},
	{
		"GetUpfs",
		http.MethodGet,
		"/inventory/upf",
		GetUpfs,
	},
	{
		"PostUpf",
		http.MethodPost,
		"/inventory/upf/:upf-hostname",
		PostUpf,
	},
	{
		"DeleteUpf",
		http.MethodDelete,
		"/inventory/upf/:upf-hostname",
		DeleteUpf,
	},
}
