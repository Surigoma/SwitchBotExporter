package metrics

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

func TestPythonMetricCompatibility(t *testing.T) {
	var input []struct {
		Kind    string
		ADType  uint8
		Payload string
	}
	var expected []struct {
		Name   string
		Labels map[string]string
		Value  float64
	}
	readFixture(t, "python-synthetic.json", &input)
	readFixture(t, "python-metrics.json", &expected)
	store := device.NewStore(time.Minute)
	now := time.Unix(100, 0)
	rssi := int16(-55)
	names := make(map[string]string)
	withoutService := make(map[string]byte)
	for _, entry := range input {
		mac := fmt.Sprintf("00:11:22:33:44:%02x", entry.Kind[0])
		if _, known := names[mac]; !known {
			withoutService[mac] = entry.Kind[0]
		}
		names[mac] = fmt.Sprintf("example-%d", entry.Kind[0])
		payload, err := hex.DecodeString(entry.Payload)
		if err != nil {
			t.Fatal(err)
		}
		field := protocol.ADField{Type: entry.ADType, ID: 0xfd3d, Payload: payload}
		if entry.ADType == protocol.ManufacturerData {
			field.ID = 0x0969
		} else {
			delete(withoutService, mac)
		}
		if _, err := store.Observe(protocol.Advertisement{MAC: mac, RSSI: &rssi, ReceivedAt: now, Fields: []protocol.ADField{field}}); err != nil {
			t.Fatal(err)
		}
	}
	for mac, kind := range withoutService {
		field := protocol.ADField{Type: protocol.ServiceData16, ID: 0xfd3d, Payload: []byte{kind}}
		if _, err := store.Observe(protocol.Advertisement{MAC: mac, RSSI: &rssi, ReceivedAt: now, Fields: []protocol.ADField{field}}); err != nil {
			t.Fatal(err)
		}
	}

	exporter := New()
	exporter.Publish(store.Snapshot(), names)
	exporter.Publish(nil, names) // devices absent from a later scan retain their series
	gathered, err := exporter.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range gathered {
		if family.GetHelp() != "Switch Bot" {
			continue
		}
		for _, actual := range family.GetMetric() {
			found := false
			for i, want := range expected {
				if want.Name != family.GetName() || len(want.Labels) != len(actual.Label) {
					continue
				}
				equal := true
				for _, label := range actual.Label {
					value, ok := want.Labels[label.GetName()]
					if !ok || value != label.GetValue() {
						equal = false
						break
					}
				}
				if !equal {
					continue
				}
				if actual.Gauge == nil || math.Abs(actual.Gauge.GetValue()-want.Value) > 1e-9 {
					t.Errorf("%s %v: got %v, expected gauge %v", want.Name, want.Labels, actual.Gauge, want.Value)
				}
				expected = append(expected[:i], expected[i+1:]...)
				found = true
				break
			}
			if !found {
				t.Errorf("unexpected series %s %v", family.GetName(), actual.Label)
			}
		}
	}
	if len(expected) != 0 {
		t.Errorf("missing %d legacy series: %v", len(expected), expected)
	}
}

func readFixture(t *testing.T, name string, result any) {
	t.Helper()
	data, err := os.ReadFile("../../testdata/advertisements/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, result); err != nil {
		t.Fatal(err)
	}
}

func TestMonitoring(t *testing.T) {
	exporter := New()
	time0 := time.Unix(100, 0)
	exporter.RecordScan(time0, time0.Add(5*time.Second), nil, 2)
	exporter.RecordScan(time0, time0.Add(2*time.Second), fmt.Errorf("test error"), 0)
	values, err := exporter.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]float64{
		"switchbot_scan_success": 0, "switchbot_scan_errors_total": 1,
		"switchbot_devices_seen": 2, "switchbot_last_success_timestamp": 105,
		"switchbot_scan_duration_seconds": 2,
	}
	for _, family := range values {
		name := family.GetName()
		if !strings.HasPrefix(name, "switchbot_") {
			continue
		}
		want, ok := expected[name]
		if !ok || len(family.Metric) != 1 {
			t.Fatalf("unexpected monitoring metric %v", family)
		}
		m := family.Metric[0]
		got := m.Gauge.GetValue() + m.Counter.GetValue() + float64(m.Histogram.GetSampleCount())
		if got != want {
			t.Errorf("%s: got %f, wanted %f", name, got, want)
		}
		delete(expected, name)
	}
	if len(expected) != 0 {
		t.Errorf("missing monitoring metrics: %v", expected)
	}
}
