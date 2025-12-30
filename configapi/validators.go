// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"errors"
	"regexp"
	"strconv"

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
