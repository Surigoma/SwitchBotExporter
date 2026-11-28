package protocol

import "time"

const (
	ServiceData16    = 0x16
	ManufacturerData = 0xff
)

// Advertisement is a device observation. BlueZ can report the fields in
// separate notifications.
type Advertisement struct {
	MAC        string
	RSSI       *int16
	ReceivedAt time.Time
	Fields     []ADField
}

type ADField struct {
	Type    uint8
	ID      uint16
	Payload []byte
}

func (f ADField) IsSwitchBot() bool {
	switch f.Type {
	case ServiceData16:
		return f.ID == 0xfd3d || f.ID == 0x0d00
	case ManufacturerData:
		return f.ID == 0x0969 || f.ID == 0x0059
	default:
		return false
	}
}
