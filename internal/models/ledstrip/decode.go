package ledstrip

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type LEDStrip struct {
	Sequence uint8  `json:"sequence_num"`
	On       bool   `json:"on"`
	Level    uint8  `json:"level"`
	Delay    bool   `json:"delay"`
	Network  string `json:"network"`
	Mode     string `json:"mode"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: LEDStrip{
		Sequence: p[6],
		On:       protocol.Bit(p[7], 7),
		Level:    p[7] & 0x7f,
		Delay:    protocol.Bit(p[8], 7),
		Network:  protocol.Network(p[8]),
		Mode:     Mode(p[8] & 0x0f),
	}}
}

func Modes() []string {
	return []string{"Undefined", "color", "scene", "music", "controller"}
}

func Mode(value uint8) string {
	if value < 2 || value > 5 {
		return "Undefined"
	}
	return Modes()[value-1]
}
