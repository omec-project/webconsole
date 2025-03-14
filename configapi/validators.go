// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"regexp"
	"strconv"
)

const (
	NAME_PATTERN = "^[a-zA-Z][a-zA-Z0-9-_]+$"
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

func isValidGnbTac(tac string) bool {
	tacNum, err := strconv.Atoi(tac)
	if err != nil {
		return false
	}
	return tacNum >= 1 && tacNum <= 16777215
}
