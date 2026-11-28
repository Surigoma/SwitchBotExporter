package outdoormeter

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type OutdoorMeter struct {
	protocol.Climate
	Sequence uint8 `json:"sequence_num"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: OutdoorMeter{Climate: protocol.DecodeClimate(p[8], p[9], p[10]), Sequence: p[6]}}
}
