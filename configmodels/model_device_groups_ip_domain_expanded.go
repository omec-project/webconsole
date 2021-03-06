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

// DeviceGroupsIpDomainExpanded - This is APN for device
type DeviceGroupsIpDomainExpanded struct {
	Dnn string `json:"dnn,omitempty"`

	UeIpPool string `json:"ue-ip-pool,omitempty"`

	DnsPrimary string `json:"dns-primary,omitempty"`

	DnsSecondary string `json:"dns-secondary,omitempty"`

	Mtu int32 `json:"mtu,omitempty"`

	UeDnnQos *DeviceGroupsIpDomainExpandedUeDnnQos `json:"ue-dnn-qos,omitempty"`
}
