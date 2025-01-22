// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import "regexp"

const NAME_PATTERN = "^[a-zA-Z0-9-_]+$"
const FQDN_PATTERN = "^([a-zA-Z0-9-]+\\.){2,}([a-zA-Z]{2,6})$"

func ValidateName(name string) bool {
	nameMatch, err := regexp.MatchString(NAME_PATTERN, name)
	if err != nil || !nameMatch {
		return false
	}
	return true
}

func ValidateFQDN(fqdn string) bool {
	fqdnMatch, err := regexp.MatchString(FQDN_PATTERN, fqdn)
	if err != nil || !fqdnMatch {
		return false
	}
	return true
}
