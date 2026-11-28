package motionsensor

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type MotionSensor struct {
	ScopeTest       string `json:"scope_test"`
	PIR             string `json:"PIR"`
	Latest          int32  `json:"latest"`
	LEDState        bool   `json:"LED_state"`
	IOTState        bool   `json:"IOT_state"`
	SensingDistance string `json:"sensing_distance"`
	LightIntensity  string `json:"light_intensity"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	distance := [...]string{"Long", "Middle", "Short", "Reserve"}
	light := [...]string{"Reserve", "dark", "bright", "Reserve"}
	reading.Data = MotionSensor{
		ScopeTest:       protocol.Choice(protocol.Bit(p[1], 7), "Not tested", "Tested"),
		PIR:             protocol.Motion(protocol.Bit(p[1], 6)),
		Latest:          protocol.ExtendedTime(p[3:5], protocol.Bit(p[5], 7)),
		LEDState:        protocol.Bit(p[5], 5),
		IOTState:        protocol.Bit(p[5], 4),
		SensingDistance: distance[(p[5]>>2)&3],
		LightIntensity:  light[p[5]&3],
	}
	return reading
}
