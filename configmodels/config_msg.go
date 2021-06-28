package configmodels

type ConfigMessage struct {
	MsgType      int
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
