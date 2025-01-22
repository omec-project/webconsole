// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 Canonical Ltd

package configapi

import (
	"regexp"

	"github.com/THREATINT/go-net"
)

const NAME_PATTERN = "^[a-zA-Z0-9-_]+$"

func validateName(name string) bool {
	nameMatch, err := regexp.MatchString(NAME_PATTERN, name)
	if err != nil || !nameMatch {
		return false
	}
	return true
}

func validateDomainName(url string) bool {
	return net.IsDomain(url)
}
