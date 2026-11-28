package curtain

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Curtain struct {
	PositionData
	AllowConnection bool `json:"allow_connection"`
}

type PositionData struct {
	Calibration string `json:"calibration"`
	Movement    string `json:"movement"`
	Position    uint8  `json:"position"`
	LightLevel  uint8  `json:"light_level"`
	DeviceChain uint8  `json:"device_chain"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	reading.Data = Curtain{PositionData: DecodePosition(p), AllowConnection: protocol.Bit(p[1], 7)}
	return reading
}

func DecodePosition(p []byte) PositionData {
	return PositionData{
		Calibration: protocol.Choice(protocol.Bit(p[1], 6), "not calibrated", "calibrated"),
		Movement:    protocol.Choice(protocol.Bit(p[3], 7), "stationary", "movement"),
		Position:    p[3] & 0x7f,
		LightLevel:  p[4] >> 4,
		DeviceChain: p[4] & 0x0f,
	}
}
