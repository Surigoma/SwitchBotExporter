package bot

import (
	"github.com/Surigoma/SwitchBotExporter/internal/models/button"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Bot struct {
	button.Button
	Mode       string   `json:"mode"`
	State      string   `json:"state"`
	CommitFlag string   `json:"commit_flag"`
	Group      []string `json:"group"`
	SyncUTC    string   `json:"sync_utc"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	reading.Data = Bot{
		Button:     button.Button{Encryption: button.Encryption(p)},
		Mode:       protocol.Choice(protocol.Bit(p[1], 7), "one state mode", "on/off state mode"),
		State:      protocol.Choice(protocol.Bit(p[1], 6), "on", "off"),
		CommitFlag: protocol.Choice(protocol.Bit(p[1], 4), "service data no data update", "service data has data update"),
		Group:      protocol.Groups(p[1]),
		SyncUTC:    protocol.Choice(protocol.Bit(p[2], 7), "no need to synchronize", "the Device has passed a time synchronization cycle (10 days by default) and requires a time synchronization"),
	}
	return reading
}
