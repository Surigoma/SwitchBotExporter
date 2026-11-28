package metrics

import (
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

// TestNewModelSeries checks series of models beyond the Python exporter.
// Manufacturer data arrives before the service data that identifies the type.
func TestNewModelSeries(t *testing.T) {
	const mac = "00:11:22:33:44:55"
	store := device.NewStore(time.Minute)
	now := time.Unix(100, 0)
	for _, field := range []protocol.ADField{
		{Type: protocol.ManufacturerData, ID: 0x0969, Payload: decodeHex(t, "000000000000000005962d00000320")},
		{Type: protocol.ServiceData16, ID: 0xfd3d, Payload: decodeHex(t, "350050")},
	} {
		if _, err := store.Observe(protocol.Advertisement{MAC: mac, ReceivedAt: now, Fields: []protocol.ADField{field}}); err != nil {
			t.Fatal(err)
		}
	}
	exporter := New()
	exporter.Publish(store.Snapshot(), map[string]string{mac: "co2"})
	gathered, err := exporter.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]float64{
		"switchbot_batt":                80,
		"switchbot_update":              100,
		"switchbot_meter_pro_co2_temp":  22.5,
		"switchbot_meter_pro_co2_humid": 45,
		"switchbot_meter_pro_co2_co2":   800,
	}
	for _, family := range gathered {
		want, ok := expected[family.GetName()]
		if !ok {
			continue
		}
		if len(family.Metric) != 1 || math.Abs(family.Metric[0].GetGauge().GetValue()-want) > 1e-9 {
			t.Errorf("%s: got %v, expected %v", family.GetName(), family.Metric, want)
		}
		delete(expected, family.GetName())
	}
	if len(expected) != 0 {
		t.Errorf("missing series: %v", expected)
	}
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
