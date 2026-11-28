package humidifier

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Humidifier struct {
	On   bool   `json:"on"`
	Mode string `json:"mode"`
	// Target is the target humidity in percent; 0 outside the target mode.
	Target uint8 `json:"target_humidity"`
}

func DecodeService(p []byte) protocol.Reading {
	data := Humidifier{On: p[1] != 0, Mode: Mode(p[4])}
	if data.Mode == "target" {
		data.Target = p[4]
	}
	return protocol.Reading{Data: data}
}

func Modes() []string {
	return []string{"auto", "low", "medium", "high", "target", "Undefined"}
}

func Mode(level uint8) string {
	switch {
	case level == 0x80 || level == 0xe4:
		return "auto"
	case level >= 101 && level <= 103:
		return Modes()[level-100]
	case level >= 1 && level <= 100:
		return "target"
	default:
		return "Undefined"
	}
}
