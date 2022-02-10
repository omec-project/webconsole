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

package configmodels

// SliceSiteInfo - give details of the site where this device group is activated
type SliceSiteInfo struct {

	// Unique name per Site.
	SiteName string `json:"site-name,omitempty"`

	Plmn SliceSiteInfoPlmn `json:"plmn,omitempty"`

	GNodeBs []SliceSiteInfoGNodeBs `json:"gNodeBs,omitempty"`

	// UPF which belong to this slice
	Upf map[string]interface{} `json:"upf,omitempty"`
}
