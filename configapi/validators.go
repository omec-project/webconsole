// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omec-project/webconsole/configmodels"
)

const (
	NAME_PATTERN = "^[a-zA-Z][a-zA-Z0-9-_]{1,255}$"
	FQDN_PATTERN = "^([a-zA-Z0-9][a-zA-Z0-9-]+\\.){2,}([a-zA-Z]{2,6})$"
)

func isValidName(name string) bool {
	nameMatch, err := regexp.MatchString(NAME_PATTERN, name)
	if err != nil {
		return false
	}
	return nameMatch
}

func isValidFQDN(fqdn string) bool {
	fqdnMatch, err := regexp.MatchString(FQDN_PATTERN, fqdn)
	if err != nil {
		return false
	}
	return fqdnMatch
}

func isValidUpfPort(port string) bool {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return portNum >= 0 && portNum <= 65535
}

func isValidGnbTac(tac int32) bool {
	return tac >= 1 && tac <= 16777215
}

func parseAndValidateSliceRequest(c *gin.Context, sliceName string) (configmodels.Slice, error) {
	var request configmodels.Slice

	ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
	if ct != "application/json" {
		return request, fmt.Errorf("unsupported content-type: %s", ct)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		return request, fmt.Errorf("JSON bind error: %+v", err)
	}

	for i, gnb := range request.SiteInfo.GNodeBs {
		if !isValidName(gnb.Name) {
			return request, fmt.Errorf("invalid gNodeBs[%d].name `%s` in Network Slice %s", i, gnb.Name, sliceName)
		}
		if !isValidGnbTac(gnb.Tac) {
			return request, fmt.Errorf("invalid gNodeBs[%d].tac %d for gNB %s in Network Slice %s", i, gnb.Tac, gnb.Name, sliceName)
		}
	}

	request.SliceName = sliceName
	// Validate required fields are not empty
	if strings.TrimSpace(request.SliceName) == "" {
		return request, fmt.Errorf("slice-name cannot be empty")
	}
	if strings.TrimSpace(request.SliceId.Sst) == "" {
		return request, fmt.Errorf("slice-id.sst cannot be empty")
	}
	if strings.TrimSpace(request.SliceId.Sd) == "" {
		return request, fmt.Errorf("slice-id.sd cannot be empty")
	}
	if len(request.SiteDeviceGroup) == 0 {
		return request, fmt.Errorf("site-device-group cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.SiteName) == "" {
		return request, fmt.Errorf("site-info.site-name cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.Plmn.Mcc) == "" {
		return request, fmt.Errorf("site-info.plmn.mcc cannot be empty")
	}
	if strings.TrimSpace(request.SiteInfo.Plmn.Mnc) == "" {
		return request, fmt.Errorf("site-info.plmn.mnc cannot be empty")
	}
	if request.SiteInfo.Upf == nil {
		return request, fmt.Errorf("site-info.upf cannot be empty")
	}
	if len(request.SiteInfo.GNodeBs) == 0 {
		return request, fmt.Errorf("site-info.gNodeBs cannot be empty")
	}
	for i, gnodeb := range request.SiteInfo.GNodeBs {
		if strings.TrimSpace(gnodeb.Name) == "" {
			return request, fmt.Errorf("site-info.gNodeBs[%d].name cannot be empty", i)
		}
		if gnodeb.Tac <= 0 {
			return request, fmt.Errorf("site-info.gNodeBs[%d].tac must be > 0", i)
		}
	}

	// Validate ApplicationFilteringRules
	// Si no hay reglas de filtrado, agrega una por defecto
	if len(request.ApplicationFilteringRules) == 0 {
		request.ApplicationFilteringRules = append(request.ApplicationFilteringRules, configmodels.SliceApplicationFilteringRules{
			RuleName:       "default",
			Action:         "permit",
			Endpoint:       "any",
			Protocol:       0,
			StartPort:      0,
			EndPort:        65535,
			AppMbrUplink:   0,
			AppMbrDownlink: 0,
			BitrateUnit:    "bps",
			TrafficClass: &configmodels.TrafficClassInfo{
				Name: "default",
				Qci:  9,
				Arp:  8,
				Pdb:  100,
				Pelr: 6,
			},
		})
	} else {
		for i, rule := range request.ApplicationFilteringRules {
			if strings.TrimSpace(rule.RuleName) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: rule-name cannot be empty", i)
			}
			if strings.TrimSpace(rule.Action) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: action cannot be empty", i)
			}
			if strings.TrimSpace(rule.Endpoint) == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: endpoint cannot be empty", i)
			}
			if rule.Protocol < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: protocol must be >= 0", i)
			}
			if rule.StartPort < 0 || rule.EndPort < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: port values must be >= 0", i)
			}
			if rule.EndPort < rule.StartPort {
				return request, fmt.Errorf("application-filtering-rules[%d]: dest-port-end must be >= dest-port-start", i)
			}
			if rule.AppMbrUplink < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: app-mbr-uplink must be >= 0", i)
			}
			if rule.AppMbrDownlink < 0 {
				return request, fmt.Errorf("application-filtering-rules[%d]: app-mbr-downlink must be >= 0", i)
			}
			if rule.BitrateUnit == "" {
				return request, fmt.Errorf("application-filtering-rules[%d]: bitrate-unit cannot be empty", i)
			}
			if rule.TrafficClass != nil {
				if strings.TrimSpace(rule.TrafficClass.Name) == "" {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.name cannot be empty", i)
				}
				if rule.TrafficClass.Qci < 1 || rule.TrafficClass.Qci > 9 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.qci must be between 1 and 9", i)
				}
				if rule.TrafficClass.Arp < 1 || rule.TrafficClass.Arp > 15 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.arp must be between 1 and 15", i)
				}
				if rule.TrafficClass.Pdb < 0 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.pdb must be >= 0", i)
				}
				if rule.TrafficClass.Pelr < 1 || rule.TrafficClass.Pelr > 8 {
					return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class.pelr must be between 1 and 8", i)
				}
			}
			if rule.TrafficClass == nil {
				return request, fmt.Errorf("application-filtering-rules[%d]: traffic-class cannot be empty", i)
			}
		}
	}

	slices.Sort(request.SiteDeviceGroup)
	request.SiteDeviceGroup = slices.Compact(request.SiteDeviceGroup)

	return request, nil
}

func isValidDeviceGroup(deviceGroup *configmodels.DeviceGroups) error {
	if deviceGroup == nil {
		return errors.New("don't find the device group data")
	}
	if deviceGroup.DeviceGroupName == "" {
		return errors.New("don't find the device group DeviceGroupName")
	}
	if deviceGroup.Imsis == nil {
		return errors.New("don't find the device group Imsis")
	}
	if deviceGroup.SiteInfo == "" {
		return errors.New("don't find the device group SiteInfo")
	}
	if deviceGroup.IpDomainName == "" {
		return errors.New("don't find the device group IpDomainName")
	}
	if deviceGroup.IpDomainExpanded.Dnn == "" {
		return errors.New("don't find the device group IpDomainExpanded.Dnn")
	}
	if deviceGroup.IpDomainExpanded.UeIpPool == "" {
		return errors.New("don't find the device group IpDomainExpanded.UeIpPool")
	}
	if deviceGroup.IpDomainExpanded.DnsPrimary == "" {
		return errors.New("don't find the device group IpDomainExpanded.DnsPrimary")
	}
	if deviceGroup.IpDomainExpanded.Mtu <= 0 {
		return errors.New("invalid value for device group IpDomainExpanded.Mtu")
	}
	if deviceGroup.IpDomainExpanded.UeDnnQos == nil {
		return errors.New("don't find the device group IpDomainExpanded.UeDnnQos")
	}
	// Set default for DnnMbrUplink if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.DnnMbrUplink < 0 {
		// Default uplink bitrate: 1000000 (1 Mbps)
		deviceGroup.IpDomainExpanded.UeDnnQos.DnnMbrUplink = 1000000
	}
	// Set default for DnnMbrDownlink if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.DnnMbrDownlink < 0 {
		// Default downlink bitrate: 1000000 (1 Mbps)
		deviceGroup.IpDomainExpanded.UeDnnQos.DnnMbrDownlink = 1000000
	}
	// Set default for BitrateUnit if empty
	if deviceGroup.IpDomainExpanded.UeDnnQos.BitrateUnit == "" {
		// Default bitrate unit: "bps"
		deviceGroup.IpDomainExpanded.UeDnnQos.BitrateUnit = "bps"
	}
	// Set default TrafficClass if nil
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass == nil {
		// Default TrafficClass with typical values
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass = &configmodels.TrafficClassInfo{
			Name: "default",
			Qci:  9,   // Default QCI value
			Arp:  1,   // Default ARP value
			Pdb:  300, // Default PDB value (ms)
			Pelr: 1,   // Default PELR value
		}
	}
	// Set default TrafficClass.Name if empty
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Name == "" {
		// Default traffic class name: "default"
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Name = "default"
	}
	// Set default Qci if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Qci < 0 {
		// Default QCI value: 9
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Qci = 9
	}
	// Set default Arp if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Arp < 0 {
		// Default ARP value: 1
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Arp = 1
	}
	// Set default Pdb if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Pdb < 0 {
		// Default PDB value: 300 (ms)
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Pdb = 300
	}
	// Set default Pelr if negative
	if deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Pelr < 0 {
		// Default PELR value: 1
		deviceGroup.IpDomainExpanded.UeDnnQos.TrafficClass.Pelr = 1
	}
	return nil
}
