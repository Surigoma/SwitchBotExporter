package contactsensor

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type ContactState struct {
	PIR string `json:"PIR"`
	HAL string `json:"HAL"`
}

type ContactTrigger struct {
	PIR int32 `json:"PIR"`
	HAL int32 `json:"HAL"`
}

type ContactCounter struct {
	Entrances  uint8 `json:"entrances"`
	GoOut      uint8 `json:"go_out"`
	ButtonPush uint8 `json:"button_push"`
}

type ContactSensor struct {
	Encryption    string         `json:"enc"`
	ScopeTested   string         `json:"scope_tested"`
	State         ContactState   `json:"state"`
	LatestTrigger ContactTrigger `json:"latest_trigger"`
	LightLevel    string         `json:"light_level"`
	Activity      ContactCounter `json:"act_counter"`
}

func DecodeService(p []byte) protocol.Reading {
	reading := protocol.BatteryReading(p)
	hal := [...]string{"door close", "door open", "timeout not close", "undefined"}
	pirTime := protocol.ExtendedTime(p[4:6], protocol.Bit(p[3], 7))
	halTime := protocol.ExtendedTime(p[6:8], protocol.Bit(p[3], 6))
	if pirTime == 0x1ffff {
		pirTime = -1
	}
	if halTime == 0x1ffff {
		halTime = -1
	}
	reading.Data = ContactSensor{
		Encryption:    protocol.Choice(protocol.Bit(p[0], 7), "No encryption", "With encryption, encryption method 1"),
		ScopeTested:   protocol.Choice(protocol.Bit(p[1], 7), "Not tested", "Tested"),
		State:         ContactState{protocol.Motion(protocol.Bit(p[1], 7)), hal[(p[3]>>1)&3]},
		LatestTrigger: ContactTrigger{pirTime, halTime},
		LightLevel:    protocol.Choice(protocol.Bit(p[3], 0), "dark", "light"),
		Activity:      ContactCounter{p[8] >> 6, (p[8] >> 3) & 3, p[8] & 7},
	}
	return reading
}
