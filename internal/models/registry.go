package models

import (
	"errors"
	"fmt"

	"github.com/Surigoma/SwitchBotExporter/internal/models/blindtilt"
	"github.com/Surigoma/SwitchBotExporter/internal/models/bot"
	"github.com/Surigoma/SwitchBotExporter/internal/models/button"
	"github.com/Surigoma/SwitchBotExporter/internal/models/ceilinglight"
	"github.com/Surigoma/SwitchBotExporter/internal/models/colorbulb"
	"github.com/Surigoma/SwitchBotExporter/internal/models/contactsensor"
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain"
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain3"
	"github.com/Surigoma/SwitchBotExporter/internal/models/hub2"
	"github.com/Surigoma/SwitchBotExporter/internal/models/hubmini"
	"github.com/Surigoma/SwitchBotExporter/internal/models/humidifier"
	"github.com/Surigoma/SwitchBotExporter/internal/models/ledstrip"
	"github.com/Surigoma/SwitchBotExporter/internal/models/lockpro"
	"github.com/Surigoma/SwitchBotExporter/internal/models/meterpro"
	"github.com/Surigoma/SwitchBotExporter/internal/models/meterth"
	"github.com/Surigoma/SwitchBotExporter/internal/models/motionsensor"
	"github.com/Surigoma/SwitchBotExporter/internal/models/outdoormeter"
	"github.com/Surigoma/SwitchBotExporter/internal/models/plugmini"
	"github.com/Surigoma/SwitchBotExporter/internal/models/reporting"
	"github.com/Surigoma/SwitchBotExporter/internal/models/smartlock"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type Definition struct {
	Name    string
	Prefix  string
	Publish func(reporting.Output, any)
	// family groups device types of one model, such as pair and normal mode.
	family       string
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

func init() {
	meterPlus := Definition{Name: "SwitchBot Meter Plus", service: decoder{6, meterth.DecodeService}, Prefix: "switchbot_meter_plus", Publish: publishModel(meterth.Publish)}
	meterPro := Definition{Name: "SwitchBot Meter Pro", service: decoder{3, protocol.BatteryReading}, manufacturer: decoder{11, meterpro.DecodeManufacturer}, Prefix: "switchbot_meter_pro", Publish: publishModel(meterpro.Publish)}
	meterProCO2 := Definition{Name: "SwitchBot Meter Pro CO2", service: decoder{3, protocol.BatteryReading}, manufacturer: decoder{15, meterpro.DecodeCO2Manufacturer}, Prefix: "switchbot_meter_pro_co2", Publish: publishModel(meterpro.PublishCO2)}
	humid := Definition{Name: "SwitchBot Humidifier", service: decoder{5, humidifier.DecodeService}, Prefix: "switchbot_humidifier", Publish: publishModel(humidifier.Publish)}
	bulb := Definition{Name: "SwitchBot Color Bulb", manufacturer: decoder{11, colorbulb.DecodeManufacturer}, Prefix: "switchbot_color_bulb", Publish: publishModel(colorbulb.Publish)}
	strip := Definition{Name: "SwitchBot LED Strip Light", manufacturer: decoder{9, ledstrip.DecodeManufacturer}, Prefix: "switchbot_led_strip", Publish: publishModel(ledstrip.Publish)}
	plugUS := Definition{Name: "SwitchBot Plug Mini (US)", manufacturer: decoder{12, plugmini.DecodeManufacturer}, Prefix: "switchbot_plug_mini", Publish: publishModel(plugmini.Publish)}
	plugJP := plugUS
	plugJP.Name = "SwitchBot Plug Mini (JP)"
	lockPro := Definition{Name: "SwitchBot Lock Pro", manufacturer: decoder{12, lockpro.DecodeManufacturer}, Prefix: "switchbot_lock_pro", Publish: publishModel(lockpro.Publish)}
	tilt := Definition{Name: "SwitchBot Blind Tilt", service: decoder{3, protocol.BatteryReading}, manufacturer: decoder{10, blindtilt.DecodeManufacturer}, Prefix: "switchbot_blind_tilt", Publish: publishModel(blindtilt.Publish)}
	for _, pair := range []struct {
		ids   [2]protocol.DeviceType
		model Definition
	}{
		{[2]protocol.DeviceType{'i', 'I'}, meterPlus},
		{[2]protocol.DeviceType{'4', 0x14}, meterPro},
		{[2]protocol.DeviceType{'5', 0x15}, meterProCO2},
		{[2]protocol.DeviceType{'e', 'E'}, humid},
		{[2]protocol.DeviceType{'u', 'U'}, bulb},
		{[2]protocol.DeviceType{'r', 'R'}, strip},
		{[2]protocol.DeviceType{'g', 'G'}, plugUS},
		{[2]protocol.DeviceType{'j', 'J'}, plugJP},
		{[2]protocol.DeviceType{'$', 0x04}, lockPro},
		{[2]protocol.DeviceType{'x', 'X'}, tilt},
	} {
		register(pair.ids, pair.model)
	}

	// Known types without a documented broadcast layout are identified only.
	for _, pair := range []struct {
		ids  [2]protocol.DeviceType
		name string
	}{
		{[2]protocol.DeviceType{'n', 'N'}, "SwitchBot Ceiling Light Pro"},
		{[2]protocol.DeviceType{'#', 0x03}, "SwitchBot Evaporative Humidifier"},
		{[2]protocol.DeviceType{'&', 0x06}, "SwitchBot Water Leak Detector"},
		{[2]protocol.DeviceType{'<', 0x1c}, "SwitchBot Relay Switch 1PM"},
		{[2]protocol.DeviceType{';', 0x1b}, "SwitchBot Relay Switch 1"},
		{[2]protocol.DeviceType{'=', 0x1d}, "SwitchBot Relay Switch 2PM"},
		{[2]protocol.DeviceType{'\'', 0x07}, "SwitchBot Universal Remote"},
		{[2]protocol.DeviceType{',', 0x0c}, "SwitchBot Roller Shade"},
		{[2]protocol.DeviceType{'%', 0x05}, "SwitchBot Hub Mini Matter"},
		{[2]protocol.DeviceType{'~', '^'}, "SwitchBot Circulator Fan"},
		{[2]protocol.DeviceType{'.', 0x0f}, "SwitchBot Robot Vacuum K20+ Pro"},
		{[2]protocol.DeviceType{'z', 'Z'}, "SwitchBot Floor Cleaning Robot S10"},
		{[2]protocol.DeviceType{'3', 0x13}, "SwitchBot Robot Vacuum K10+ Pro Combo"},
		{[2]protocol.DeviceType{'}', ']'}, "SwitchBot Mini Robot Vacuum K10+"},
		{[2]protocol.DeviceType{'(', 0x08}, "SwitchBot Mini Robot Vacuum K10+ Pro"},
		{[2]protocol.DeviceType{'*', 0x0a}, "SwitchBot Air Purifier (US)"},
		{[2]protocol.DeviceType{'+', 0x0b}, "SwitchBot Air Purifier (JP)"},
		{[2]protocol.DeviceType{'7', 0x17}, "SwitchBot Air Purifier Table (US)"},
		{[2]protocol.DeviceType{'8', 0x18}, "SwitchBot Air Purifier Table (JP)"},
		{[2]protocol.DeviceType{'-', 0x0d}, "SwitchBot Lock Lite"},
		{[2]protocol.DeviceType{'>', 0x1e}, "SwitchBot Garage Door Opener"},
		{[2]protocol.DeviceType{'?', 0x1f}, "SwitchBot Plug Mini (EU)"},
	} {
		register(pair.ids, Definition{Name: pair.name})
	}
	for id, name := range map[protocol.DeviceType]string{
		'Q': "SwitchBot Ceiling light [Add Mode]",
		'O': "SwitchBot Smart Lock [Add Mode]",
		'Y': "SwitchBot Keypad touch [Add Mode]",
		'S': "SwitchBot Motion Sensor [Add Mode]",
		'V': "SwitchBot Hub 2 [Add Mode]",
	} {
		register([2]protocol.DeviceType{id}, Definition{Name: name})
	}
}

// register adds one model under each of its device types.
func register(ids [2]protocol.DeviceType, model Definition) {
	model.family = string(rune(ids[0]))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := definitions[id]; exists {
			panic(fmt.Sprintf("device type %q registered twice", byte(id)))
		}
		definitions[id] = model
	}
}

func Lookup(kind protocol.DeviceType) (Definition, bool) {
	value, ok := definitions[kind]
	return value, ok
}

func SameModel(a, b protocol.DeviceType) bool {
	if a == b || a == '[' && b == '{' || a == '{' && b == '[' {
		return true
	}
	x, y := definitions[a], definitions[b]
	return x.family != "" && x.family == y.family
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
