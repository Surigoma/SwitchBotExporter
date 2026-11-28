package curtain3

import (
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Curtain3 struct {
	curtain.PositionData
	Direction string `json:"direction"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	reading.Data = Curtain3{
		PositionData: curtain.DecodePosition(p),
		Direction:    protocol.Choice(protocol.Bit(p[1], 7), "left", "right"),
	}
	return reading
}
