package colorbulb

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type ColorBulb struct {
	Sequence    uint8  `json:"sequence_num"`
	On          bool   `json:"on"`
	Level       uint8  `json:"level"`
	Delay       bool   `json:"delay"`
	Network     string `json:"network"`
	Preset      bool   `json:"preset"`
	LightState  string `json:"light_state"`
	BadRSSI     bool   `json:"bad_rssi"`
	DynamicRate uint8  `json:"dynamic_rate"`
	LoopIndex   uint8  `json:"loop_index"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: ColorBulb{
		Sequence:    p[6],
		On:          protocol.Bit(p[7], 7),
		Level:       p[7] & 0x7f,
		Delay:       protocol.Bit(p[8], 7),
		Network:     protocol.Network(p[8]),
		Preset:      protocol.Bit(p[8], 3),
		LightState:  LightStates()[p[8]&7],
		BadRSSI:     protocol.Bit(p[9], 7),
		DynamicRate: p[9] & 0x7f,
		LoopIndex:   p[10] >> 2,
	}}
}

func LightStates() [8]string {
	return [8]string{"Undefined", "white", "color", "dynamic", "Undefined", "Undefined", "Undefined", "Undefined"}
}
