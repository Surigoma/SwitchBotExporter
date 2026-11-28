package ble

import (
	"encoding/hex"
	"log/slog"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
	"github.com/godbus/dbus/v5"
)

type properties = map[string]dbus.Variant

// Observation includes the original BlueZ keys for comparison with HCI traces.
type Observation struct {
	protocol.Advertisement
	Source           string
	RSSIChanged      bool
	ServiceData      map[string]string
	ManufacturerData map[string]string
}

type tracker struct {
	adapter dbus.ObjectPath
	// targets limits observations to these MAC addresses. Nil allows all.
	targets map[string]struct{}
	devices map[dbus.ObjectPath]device
}

type device struct {
	mac     string
	adapter dbus.ObjectPath
	rssi    *int16
}

func NormalizeMAC(address string) (string, bool) {
	mac, err := net.ParseMAC(address)
	if err != nil || len(mac) != 6 {
		return "", false
	}
	return mac.String(), true
}

// remember reads only device identity. Initial or fallback property snapshots
// may contain advertising properties from outside this scan window.
func (t *tracker) remember(path dbus.ObjectPath, fields properties) {
	address, _ := fields["Address"].Value().(string)
	mac, _ := NormalizeMAC(address)
	adapter, _ := fields["Adapter"].Value().(dbus.ObjectPath)
	if mac != "" && adapter == t.adapter {
		t.devices[path] = device{mac: mac, adapter: adapter}
	}
}

func (t *tracker) observe(path dbus.ObjectPath, fields properties, invalidated []string, at time.Time, source string) (Observation, bool) {
	info := t.devices[path]
	for _, key := range invalidated {
		switch key {
		case "Address":
			info.mac = ""
		case "Adapter":
			info.adapter = ""
		case "RSSI":
			info.rssi = nil
		}
	}
	if value, ok := fields["Address"]; ok {
		address, _ := value.Value().(string)
		mac, _ := NormalizeMAC(address)
		if mac != info.mac {
			info.rssi = nil
		}
		info.mac = mac
	}
	if value, ok := fields["Adapter"]; ok {
		info.adapter, _ = value.Value().(dbus.ObjectPath)
	}
	if info.mac == "" || info.adapter != t.adapter {
		delete(t.devices, path)
		return Observation{}, false
	}
	if _, wanted := t.targets[info.mac]; t.targets != nil && !wanted {
		t.devices[path] = info
		return Observation{}, false
	}

	rssi, rssiChanged := fields["RSSI"].Value().(int16)
	if rssiChanged {
		info.rssi = &rssi
	}
	t.devices[path] = info

	service := readData[string]("ServiceData", fields)
	manufacturer := readData[uint16]("ManufacturerData", fields)
	if !rssiChanged && len(service) == 0 && len(manufacturer) == 0 {
		return Observation{}, false
	}

	obs := Observation{
		Advertisement:    protocol.Advertisement{MAC: info.mac, ReceivedAt: at},
		Source:           source,
		RSSIChanged:      rssiChanged,
		ServiceData:      make(map[string]string, len(service)),
		ManufacturerData: make(map[string]string, len(manufacturer)),
	}
	if info.rssi != nil {
		strength := *info.rssi
		obs.RSSI = &strength
	}
	for id, payload := range manufacturer {
		obs.ManufacturerData[hex.EncodeToString([]byte{byte(id >> 8), byte(id)})] = hex.EncodeToString(payload)
	}

	// Sorting makes the original UUID selection deterministic if a dictionary
	// happens to contain aliases that differ only by case.
	serviceKeys := make([]string, 0, len(service))
	for key := range service {
		serviceKeys = append(serviceKeys, key)
		obs.ServiceData[key] = hex.EncodeToString(service[key])
	}
	sort.Strings(serviceKeys)
	for _, candidate := range []struct {
		uuid string
		id   uint16
	}{
		{"0000fd3d-0000-1000-8000-00805f9b34fb", 0xfd3d},
		{"00000d00-0000-1000-8000-00805f9b34fb", 0x0d00},
	} {
		for _, key := range serviceKeys {
			if strings.EqualFold(key, candidate.uuid) {
				obs.Fields = append(obs.Fields, protocol.ADField{Type: protocol.ServiceData16, ID: candidate.id, Payload: service[key]})
				break
			}
		}
	}
	for _, id := range []uint16{0x0969, 0x0059} {
		if payload, ok := manufacturer[id]; ok {
			obs.Fields = append(obs.Fields, protocol.ADField{Type: protocol.ManufacturerData, ID: id, Payload: payload})
		}
	}
	return obs, true
}

func readData[K comparable](property string, fields properties) map[K][]byte {
	value, exists := fields[property]
	if !exists {
		return nil
	}
	entries, ok := value.Value().(map[K]dbus.Variant)
	if !ok {
		slog.Debug("invalid BlueZ property type", "property", property, "signature", value.Signature().String())
		return nil
	}
	data := make(map[K][]byte, len(entries))
	for key, entry := range entries {
		payload, ok := entry.Value().([]byte)
		if !ok {
			slog.Debug("invalid BlueZ advertisement value", "property", property, "signature", entry.Signature().String())
			continue
		}
		data[key] = append([]byte(nil), payload...)
	}
	return data
}
