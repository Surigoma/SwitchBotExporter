package models_test

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/models"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

const mac = "000000000000"

// Cases follow the OpenWonderLabs/SwitchBotAPI-BLE broadcast tables.
// Manufacturer payloads start after the company ID.
var cases = []struct {
	kind         protocol.DeviceType
	manufacturer bool
	payload      string
	data         map[string]any
}{
	{'i', false, "690064" + "05962d", map[string]any{"temp": 22.5, "hum": 45.0, "group": []any{}, "alert": map[string]any{"temp": "no alert", "hum": "no alert"}, "battery": 100.0}},
	{'4', false, "340050", map[string]any{"battery": 80.0}},
	{'4', true, mac + "0000" + "05962d", map[string]any{"temp": 22.5, "hum": 45.0}},
	{0x14, true, mac + "0000" + "051530", map[string]any{"temp": -21.5, "hum": 48.0}},
	{'5', true, mac + "0000" + "05962d" + "0000" + "0320", map[string]any{"temp": 22.5, "hum": 45.0, "co2": 800.0}},
	{'e', false, "6580000032", map[string]any{"on": true, "mode": "target", "target_humidity": 50.0}},
	{'e', false, "6500000080", map[string]any{"on": false, "mode": "auto", "target_humidity": 0.0}},
	{'E', false, "4580000066", map[string]any{"on": true, "mode": "medium", "target_humidity": 0.0}},
	{'u', true, mac + "01" + "b2a98a14", map[string]any{"sequence_num": 1.0, "on": true, "level": 50.0, "delay": true, "network": "iot connected", "preset": true, "light_state": "white", "bad_rssi": true, "dynamic_rate": 10.0, "loop_index": 5.0}},
	{'r', true, mac + "02" + "6413", map[string]any{"sequence_num": 2.0, "on": false, "level": 100.0, "delay": false, "network": "iot connecting", "mode": "scene"}},
	{'g', true, mac + "10" + "800532812c", map[string]any{"sequence_num": 16.0, "on": true, "delay": true, "timer": false, "synced_utc": true, "wifi_rssi": -50.0, "overload": true, "power": 30.0}},
	{'j', true, mac + "11" + "0000000000", map[string]any{"sequence_num": 17.0, "on": false, "delay": false, "timer": false, "synced_utc": false, "wifi_rssi": 0.0, "overload": false, "power": 0.0}},
	{'x', false, "780064", map[string]any{"battery": 100.0}},
	{'x', true, mac + "03" + "01b250", map[string]any{"sequence_num": 3.0, "calibrated": true, "movement": "movement", "tilt": 50.0, "light_level": 5.0}},
	{'$', true, mac + "04" + "8818d580c0", map[string]any{"sequence_num": 4.0, "calibrated": true, "lock": "Unlocked", "door_open": true, "auto_lock_paused": true, "charging": true, "double_lock_mode": true, "unclosed_door_alarm": true, "unlocked_alarm": true, "battery": 85.0}},
	{'$', true, mac + "05" + "7800000000", map[string]any{"sequence_num": 5.0, "calibrated": false, "lock": "Undefined", "door_open": false, "auto_lock_paused": false, "charging": false, "double_lock_mode": false, "unclosed_door_alarm": false, "unlocked_alarm": false, "battery": 0.0}},
}

func TestNewModels(t *testing.T) {
	for _, test := range cases {
		t.Run(string(rune(test.kind))+"/"+test.payload, func(t *testing.T) {
			payload, err := hex.DecodeString(test.payload)
			if err != nil {
				t.Fatal(err)
			}
			var reading protocol.Reading
			if test.manufacturer {
				if reading, err = models.DecodeManufacturer(test.kind, payload); err != nil {
					t.Fatal(err)
				}
				if _, err := models.DecodeManufacturer(test.kind, payload[:len(payload)-1]); err == nil {
					t.Fatal("accepted truncated manufacturer payload")
				}
			} else {
				kind, parsed, err := models.DecodeService(payload)
				if err != nil || kind != test.kind {
					t.Fatalf("service: kind %q, error %v", kind, err)
				}
				reading = parsed
				if _, _, err := models.DecodeService(payload[:len(payload)-1]); err == nil {
					t.Fatal("accepted truncated service payload")
				}
			}
			value, err := json.Marshal(device.State{Type: test.kind, Reading: reading})
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct{ Data map[string]any }
			if err := json.Unmarshal(value, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded.Data, test.data) {
				t.Fatalf("decoded %v, expected %v", decoded.Data, test.data)
			}
		})
	}
}

func TestModelFamilies(t *testing.T) {
	for _, test := range []struct {
		a, b protocol.DeviceType
		same bool
	}{
		{'4', 0x14, true},
		{'g', 'G', true},
		{'g', 'j', false},
		{'i', 'T', false},
		{'[', '{', true},
		{'c', 'C', false},
	} {
		if got := models.SameModel(test.a, test.b); got != test.same {
			t.Errorf("SameModel(%q, %q) = %v", byte(test.a), byte(test.b), got)
		}
	}
	for _, kind := range []protocol.DeviceType{'&', 0x06, '?', 'n', 'Q', 'V'} {
		if _, ok := models.Lookup(kind); !ok {
			t.Errorf("type %q not registered", byte(kind))
		}
	}
}
