package app

import (
	"sort"
	"strings"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/ble"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type rawField struct {
	ADType     uint8  `json:"adType"`
	Key        string `json:"key"`
	PayloadHex string `json:"payloadHex"`
}

type rawObservation struct {
	MAC        string     `json:"mac"`
	RSSI       *int16     `json:"rssi,omitempty"`
	ReceivedAt time.Time  `json:"receivedAt"`
	Fields     []rawField `json:"fields"`
}

func toRawObservation(obs ble.Observation) rawObservation {
	observation := rawObservation{
		MAC: obs.MAC, RSSI: obs.RSSI, ReceivedAt: obs.ReceivedAt.UTC(),
		Fields: make([]rawField, 0, len(obs.ServiceData)+len(obs.ManufacturerData)),
	}
	for uuid, payload := range obs.ServiceData {
		observation.Fields = append(observation.Fields, rawField{serviceADType(uuid), uuid, payload})
	}
	for company, payload := range obs.ManufacturerData {
		observation.Fields = append(observation.Fields, rawField{protocol.ManufacturerData, company, payload})
	}
	sort.Slice(observation.Fields, func(i, j int) bool {
		a, b := observation.Fields[i], observation.Fields[j]
		if a.ADType == b.ADType {
			return a.Key < b.Key
		}
		return a.ADType < b.ADType
	})
	return observation
}

// BlueZ uses full UUIDs as keys. This returns the shortest Bluetooth UUID
// representation, since the original wire width is unavailable here.
func serviceADType(uuid string) uint8 {
	prefix, base := strings.CutSuffix(strings.ToLower(uuid), "-0000-1000-8000-00805f9b34fb")
	if !base || len(prefix) != 8 {
		return 0x21
	}
	if prefix[:4] == "0000" {
		return protocol.ServiceData16
	}
	return 0x20
}
