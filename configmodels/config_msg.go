// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package configmodels

const (
	Post_op = iota
	Put_op
	Delete_op
)

type ConfigMessage struct {
	MsgType      int
	MsgMethod    int
	DevGroup     *DeviceGroups
	Slice        *Slice
	DevGroupName string
	SliceName    string
}

// Slice + attached device group
type SliceConfigSnapshot struct {
	SliceMsg *Slice
	DevGroup []*DeviceGroups
}

// DevGroup + slice name
type DevGroupConfigSnapshot struct {
	SliceName string
	DevGroup  *DeviceGroups
}
