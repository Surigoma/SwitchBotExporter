package protocol_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/models"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

func TestAgainstPythonDecoder(t *testing.T) {
	paths := []string{"../../testdata/advertisements/python-synthetic.json"}
	if path := os.Getenv("SWITCHBOT_EXTRA_DECODER_CASES"); path != "" {
		paths = append(paths, path)
	}
	for index, path := range paths {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var cases []struct {
				Kind    string
				ADType  uint8
				Payload string
				Data    map[string]any
			}
			if err := json.Unmarshal(raw, &cases); err != nil {
				t.Fatal(err)
			}
			for _, test := range cases {
				t.Run(test.Kind+"/"+test.Payload, func(t *testing.T) {
					payload, err := hex.DecodeString(test.Payload)
					if err != nil {
						t.Fatal(err)
					}
					kind := protocol.DeviceType(test.Kind[0])
					var reading protocol.Reading
					if test.ADType == protocol.ServiceData16 {
						got, parsed, err := models.DecodeService(payload)
						if err != nil || got != kind {
							t.Fatalf("service: kind %q, error %v", got, err)
						}
						reading = parsed
						if index == 0 {
							if _, _, err := models.DecodeService(payload[:len(payload)-1]); err == nil {
								t.Fatal("accepted truncated service payload")
							}
						}
					} else {
						reading, err = models.DecodeManufacturer(kind, payload)
						if err != nil {
							t.Fatal(err)
						}
						if index == 0 {
							if _, err := models.DecodeManufacturer(kind, payload[:len(payload)-1]); err == nil {
								t.Fatal("accepted truncated manufacturer payload")
							}
						}
					}
					value, err := json.Marshal(device.State{Type: kind, Reading: reading})
					if err != nil {
						t.Fatal(err)
					}
					var decoded struct{ Data map[string]any }
					if err := json.Unmarshal(value, &decoded); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(decoded.Data, test.Data) {
						t.Fatalf("decoded %v, Python expected %v", decoded.Data, test.Data)
					}
				})
			}
		})
	}
}
