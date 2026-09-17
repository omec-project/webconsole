// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"math"
	"regexp"
	"strconv"
	"strings"
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

// The first digits of an IMSI are its home PLMN (MCC+MNC), so a subscriber whose IMSI does not
// start with a slice's PLMN does not belong to that network and its records would be filed under
// the wrong serving PLMN — an empty PLMN means the slice hasn't been assigned one yet and is not
// something this check can validate against. A PLMN with only one of MCC/MNC set is neither
// unassigned nor complete, so it is rejected rather than treated as a usable prefix.
func isValidImsiForPlmn(imsi, mcc, mnc string) bool {
	if mcc == "" && mnc == "" {
		return true
	}
	if mcc == "" || mnc == "" {
		return false
	}
	return strings.HasPrefix(imsi, mcc+mnc)
}

// A rate is normalised to bps and stored in a signed 32-bit field, so a negative rate is not a
// rate at all and one that does not fit the field cannot be served as configured.
func isValidBitrate(value int32, unit string) bool {
	if value < 0 {
		return false
	}
	multiplier, _ := bitrateMultiplier(unit)
	return int64(value)*multiplier <= math.MaxInt32
}

// A device group rate is normalised to bps and stored in a signed 64-bit field. A negative rate is
// not a rate, and one above what the served string can express cannot be served as configured, so
// the group is refused rather than accepted carrying a rate the operator never asked for.
//
// The bound is checked by dividing rather than multiplying because the product is what overflows:
// the field is wide enough that a Gbps rate can wrap it, and a wrapped product lands back inside
// any range a multiplication would be compared against.
func isValidDeviceGroupBitrate(value int64, unit string) bool {
	if value < 0 {
		return false
	}
	multiplier, _ := bitrateMultiplier(unit)
	return value <= maxDeviceGroupBitrateBps/multiplier
}
