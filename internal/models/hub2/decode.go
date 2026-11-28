package hub2

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Hub2 struct {
	protocol.Climate
	Sequence   uint8 `json:"sequence_num"`
	LightLevel uint8 `json:"light_level"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: Hub2{Climate: protocol.DecodeClimate(p[13], p[14], p[15]), Sequence: p[11], LightLevel: p[12] & 0xf}}
}
