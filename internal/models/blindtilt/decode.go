package blindtilt

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type BlindTilt struct {
	Sequence   uint8  `json:"sequence_num"`
	Calibrated bool   `json:"calibrated"`
	Movement   string `json:"movement"`
	Tilt       uint8  `json:"tilt"`
	LightLevel uint8  `json:"light_level"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: BlindTilt{
		Sequence:   p[6],
		Calibrated: protocol.Bit(p[7], 0),
		Movement:   protocol.Choice(protocol.Bit(p[8], 7), "stationary", "movement"),
		Tilt:       p[8] & 0x7f,
		LightLevel: p[9] >> 4,
	}}
}
