package button

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Button struct {
	Encryption string `json:"enc"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	reading.Data = Button{Encryption: Encryption(p)}
	return reading
}

func Encryption(p []byte) string {
	values := [...]string{"no encryption", "encryption algorithm 1 (the same as the Bot)", "encryption algorithm 2", "encryption algorithm 3"}
	return values[(p[0]>>7)*2+((p[1]>>4)&1)]
}
