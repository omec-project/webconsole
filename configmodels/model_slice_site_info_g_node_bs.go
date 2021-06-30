/*
 * Connectivity Service Configuration
 *
 * APIs to configure connectivity service in Aether Network
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package configmodels

type SliceSiteInfoGNodeBs struct {

	Name string `json:"name,omitempty"`

	// unique tac per gNB. This should match gNB configuration.
	Tac int32 `json:"tac,omitempty"`
}