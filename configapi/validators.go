// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"math"
	"regexp"
	"strconv"
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

// A rate is normalised to bps and stored in a signed 32-bit field, so a negative rate is not a
// rate at all and one that does not fit the field cannot be served as configured.
func isValidBitrate(value int32, unit string) bool {
	if value < 0 {
		return false
	}
	multiplier, _ := bitrateMultiplier(unit)
	return int64(value)*multiplier <= math.MaxInt32
}
