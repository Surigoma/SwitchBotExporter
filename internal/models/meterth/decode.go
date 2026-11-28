package meterth

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type ClimateAlert struct {
	Temperature string `json:"temp"`
	Humidity    string `json:"hum"`
}

type MeterTH struct {
	protocol.Climate
	Group []string     `json:"group"`
	Alert ClimateAlert `json:"alert"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	temperatureAlerts := [...]string{"no alert", "low temp alert", "high temp alert", "temp alert"}
	humidityAlerts := [...]string{"no alert", "low humidity alert", "high humidity alert", "humidity alert"}
	reading.Data = MeterTH{
		Climate: protocol.DecodeClimate(p[3], p[4], p[5]),
		Group:   protocol.Groups(p[1]),
		Alert:   ClimateAlert{temperatureAlerts[p[3]>>6], humidityAlerts[(p[3]>>4)&3]},
	}
	return reading
}
