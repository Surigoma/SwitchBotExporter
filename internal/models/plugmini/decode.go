package plugmini

import (
	"encoding/binary"

	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type PlugMini struct {
	Sequence  uint8   `json:"sequence_num"`
	On        bool    `json:"on"`
	Delay     bool    `json:"delay"`
	Timer     bool    `json:"timer"`
	SyncedUTC bool    `json:"synced_utc"`
	WiFiRSSI  int16   `json:"wifi_rssi"`
	Overload  bool    `json:"overload"`
	Power     float64 `json:"power"`
}

// DecodeManufacturer reads the payload after the company ID. The specification
// does not state units; WiFi RSSI is negated and power is divided by 10 (W) as
// in pySwitchbot.
func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: PlugMini{
		Sequence:  p[6],
		On:        p[7] == 0x80,
		Delay:     protocol.Bit(p[8], 0),
		Timer:     protocol.Bit(p[8], 1),
		SyncedUTC: protocol.Bit(p[8], 2),
		WiFiRSSI:  -int16(p[9]),
		Overload:  protocol.Bit(p[10], 7),
		Power:     float64(binary.BigEndian.Uint16(p[10:12])&0x7fff) / 10,
	}}
}
