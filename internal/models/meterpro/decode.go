package meterpro

import (
	"encoding/binary"

	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

// MeterPro is the manufacturer data of Meter Pro. Battery arrives in the
// service data.
type MeterPro struct {
	protocol.Climate
}

type MeterProCO2 struct {
	protocol.Climate
	CO2 uint16 `json:"co2"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: MeterPro{protocol.DecodeClimate(p[8], p[9], p[10])}}
}

func DecodeCO2Manufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: MeterProCO2{protocol.DecodeClimate(p[8], p[9], p[10]), binary.BigEndian.Uint16(p[13:15])}}
}
