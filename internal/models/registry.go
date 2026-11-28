package models

import (
	"errors"
	"fmt"

	"github.com/Surigoma/SwitchBotExporter/internal/models/bot"
	"github.com/Surigoma/SwitchBotExporter/internal/models/button"
	"github.com/Surigoma/SwitchBotExporter/internal/models/ceilinglight"
	"github.com/Surigoma/SwitchBotExporter/internal/models/contactsensor"
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain"
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain3"
	"github.com/Surigoma/SwitchBotExporter/internal/models/hub2"
	"github.com/Surigoma/SwitchBotExporter/internal/models/hubmini"
	"github.com/Surigoma/SwitchBotExporter/internal/models/meterth"
	"github.com/Surigoma/SwitchBotExporter/internal/models/motionsensor"
	"github.com/Surigoma/SwitchBotExporter/internal/models/outdoormeter"
	"github.com/Surigoma/SwitchBotExporter/internal/models/reporting"
	"github.com/Surigoma/SwitchBotExporter/internal/models/smartlock"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Definition struct {
	Name         string
	Prefix       string
	Publish      func(reporting.Output, any)
	service      decoder
	manufacturer decoder
}

type decoder struct {
	min  int
	read func([]byte) protocol.Reading
}

func publishModel[T any](publish func(reporting.Output, T)) func(reporting.Output, any) {
	return func(output reporting.Output, data any) {
		if value, ok := data.(T); ok {
			publish(output, value)
		}
	}
}

var definitions = map[protocol.DeviceType]Definition{
	'H': {Name: "SwitchBot Bot (WoHand)", service: decoder{3, bot.DecodeService}, Prefix: "switchbot_bot"},
	'B': {Name: "WoButton [Add Mode]"},
	'b': {Name: "WoButton", service: decoder{3, button.DecodeService}, Prefix: "switchbot_button"},
	'L': {Name: "SwitchBot Hub (WoLink) [Add Mode]"},
	'l': {Name: "SwitchBot Hub (WoLink)"},
	'P': {Name: "SwitchBot Hub Plus (WoLink Plus) [Add Mode]"},
	'p': {Name: "SwitchBot Hub Plus (WoLink Plus)"},
	'F': {Name: "SwitchBot Fan (WoFan) [Add Mode]"},
	'f': {Name: "SwitchBot Fan (WoFan)"},
	't': {Name: "SwitchBot MeterTH (WoSensorTH) [Add Mode]"},
	'T': {Name: "SwitchBot MeterTH (WoSensorTH)", service: decoder{6, meterth.DecodeService}, Prefix: "switchbot_meter_th", Publish: publishModel(meterth.Publish)},
	'M': {Name: "SwitchBot Mini (HubMini) [Add Mode]"},
	'm': {Name: "SwitchBot Mini (HubMini)", manufacturer: decoder{13, hubmini.DecodeManufacturer}},
	'D': {Name: "SwitchBot Contact Sensor [Add Mode]"},
	'd': {Name: "SwitchBot Contact Sensor", service: decoder{9, contactsensor.DecodeService}, Prefix: "switchbot_contact_sensor", Publish: publishModel(contactsensor.Publish)},
	'W': {Name: "SwitchBot Outdoor Meter [Add Mode]"},
	'w': {Name: "SwitchBot Outdoor Meter", service: decoder{3, protocol.BatteryReading}, manufacturer: decoder{11, outdoormeter.DecodeManufacturer}, Prefix: "switchbot_meter_outdoor", Publish: publishModel(outdoormeter.Publish)},
	'o': {Name: "SwitchBot Smart Lock", service: decoder{3, protocol.BatteryReading}, manufacturer: decoder{9, smartlock.DecodeManufacturer}, Prefix: "switchbot_smartlock", Publish: publishModel(smartlock.Publish)},
	'y': {Name: "SwitchBot Keypad touch", service: decoder{3, protocol.BatteryReading}, Prefix: "switchbot_keypad_touch"},
	'C': {Name: "SwitchBot Curtain [Add Mode]"},
	'c': {Name: "SwitchBot Curtain", service: decoder{5, curtain.DecodeService}, Prefix: "switchbot_curtain", Publish: publishModel(curtain.Publish)},
	'[': {Name: "SwitchBot Curtain 3 [Const Adv]", service: decoder{5, curtain3.DecodeService}, Prefix: "switchbot_curtain3", Publish: publishModel(curtain3.Publish)},
	'{': {Name: "SwitchBot Curtain 3 [Pair]", service: decoder{5, curtain3.DecodeService}, Prefix: "switchbot_curtain3", Publish: publishModel(curtain3.Publish)},
	'q': {Name: "SwitchBot Ceiling light", manufacturer: decoder{11, ceilinglight.DecodeManufacturer}, Prefix: "switchbot_ceiling_light", Publish: publishModel(ceilinglight.Publish)},
	's': {Name: "SwitchBot Motion Sensor", service: decoder{6, motionsensor.DecodeService}, Prefix: "switchbot_motion_sensor", Publish: publishModel(motionsensor.Publish)},
	'v': {Name: "SwitchBot Hub 2", manufacturer: decoder{16, hub2.DecodeManufacturer}, Prefix: "switchbot_hub_2", Publish: publishModel(hub2.Publish)},
}

func Lookup(kind protocol.DeviceType) (Definition, bool) {
	value, ok := definitions[kind]
	return value, ok
}

func SameModel(a, b protocol.DeviceType) bool {
	return a == b || a == '[' && b == '{' || a == '{' && b == '['
}

func DecodeService(p []byte) (protocol.DeviceType, protocol.Reading, error) {
	if len(p) == 0 {
		return 0, protocol.Reading{}, errors.New("empty service data")
	}
	kind := protocol.DeviceType(p[0] & 0x7f)
	reading, err := definitions[kind].service.decode(kind, "service", p)
	return kind, reading, err
}

func DecodeManufacturer(kind protocol.DeviceType, p []byte) (protocol.Reading, error) {
	return definitions[kind].manufacturer.decode(kind, "manufacturer", p)
}

func (d decoder) decode(kind protocol.DeviceType, field string, p []byte) (protocol.Reading, error) {
	if d.read == nil {
		return protocol.Reading{}, nil
	}
	if len(p) < d.min {
		return protocol.Reading{}, fmt.Errorf("%q %s data: need %d bytes, got %d", byte(kind), field, d.min, len(p))
	}
	return d.read(p), nil
}
