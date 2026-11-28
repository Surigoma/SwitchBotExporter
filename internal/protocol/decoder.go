package protocol

import "encoding/binary"

// Reading carries one decoded part of a device advertisement. Battery and
// the model-specific fields can arrive in separate advertisements.
type Reading struct {
	Battery *uint8
	Data    any
}

func (r Reading) Empty() bool {
	return r.Battery == nil && r.Data == nil
}

type DeviceType byte

func BatteryReading(p []byte) Reading {
	battery := p[2] & 0x7f
	return Reading{Battery: &battery}
}

func Bit(value uint8, shift uint) bool {
	return value&(1<<shift) != 0
}

func Choice(flag bool, zero, one string) string {
	if flag {
		return one
	}
	return zero
}

func Motion(flag bool) string {
	return Choice(flag, "No one moves", "Someone is moving")
}

func Groups(value uint8) []string {
	out := make([]string, 0, 4)
	for i, group := range [...]string{"A", "B", "C", "D"} {
		if Bit(value, uint(i)) {
			out = append(out, group)
		}
	}
	return out
}

type Climate struct {
	Temperature float64 `json:"temp"`
	Humidity    uint8   `json:"hum"`
}

func DecodeClimate(fraction, whole, humidity uint8) Climate {
	temperature := float64(whole&0x7f) + float64(fraction&0x0f)/10
	if !Bit(whole, 7) {
		temperature = -temperature
	}
	if Bit(humidity, 7) {
		temperature = (temperature - 32) / 1.8
	}
	return Climate{temperature, humidity & 0x7f}
}

func ExtendedTime(payload []byte, high bool) int32 {
	value := int32(binary.BigEndian.Uint16(payload))
	if high {
		value |= 1 << 16
	}
	return value
}

// Network reads bits 6:4 of the light status byte.
func Network(value uint8) string {
	states := [...]string{"wifi connecting", "iot connecting", "iot connected"}
	if index := (value >> 4) & 7; int(index) < len(states) {
		return states[index]
	}
	return "Undefined"
}
