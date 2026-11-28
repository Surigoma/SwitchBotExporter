package ceilinglight

import (
	"encoding/binary"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type CeilingLight struct {
	Sequence         uint8  `json:"sequence_num"`
	Level            uint8  `json:"level"`
	Status           bool   `json:"status"`
	SmallStatus      bool   `json:"small_status"`
	ColorTemperature uint16 `json:"color_temp"`
	Channel          uint8  `json:"channel"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: CeilingLight{
		Sequence: p[6], Level: p[7] & 0x7f,
		Status: protocol.Bit(p[10], 7), SmallStatus: protocol.Bit(p[10], 6),
		ColorTemperature: binary.BigEndian.Uint16(p[8:10]), Channel: 1 + (p[10] & 1),
	}}
}
