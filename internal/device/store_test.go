package device

import (
	"errors"
	"testing"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/models/outdoormeter"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

func outdoor(value byte) protocol.ADField {
	return protocol.ADField{Type: protocol.ManufacturerData, ID: 0x0969, Payload: []byte{0, 0, 0, 0, 0, 0, value, 0, 0, 0x95, 0x32}}
}

func service(kind byte, rest ...byte) protocol.ADField {
	return protocol.ADField{Type: protocol.ServiceData16, ID: 0xfd3d, Payload: append([]byte{kind}, rest...)}
}

func TestJoinAndExpire(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewStore(30 * time.Second)
	apply := func(mac string, at time.Time, fields ...protocol.ADField) {
		t.Helper()
		if _, err := store.Observe(protocol.Advertisement{MAC: mac, ReceivedAt: at, Fields: fields}); err != nil {
			t.Fatal(err)
		}
	}

	apply("first", now, outdoor(1))
	apply("expired", now, outdoor(3))
	apply("first", now.Add(29*time.Second), outdoor(2), service('w', 0, 0))
	apply("expired", now.Add(30*time.Second), service('w', 0, 7))
	apply("service-first", now.Add(30*time.Second), service('w', 0, 8))
	apply("service-first", now.Add(31*time.Second), outdoor(4))
	snapshot := store.Snapshot()
	first := snapshot["first"]
	if first.Data.(outdoormeter.OutdoorMeter).Sequence != 2 || *first.Battery != 0 || first.UpdatedAt != now.Add(29*time.Second) {
		t.Fatalf("first observation: %+v", first)
	}
	if expired := snapshot["expired"]; expired.Data != nil || *expired.Battery != 7 {
		t.Fatalf("expired manufacturer data applied: %+v", expired)
	}
	if snapshot["service-first"].Data.(outdoormeter.OutdoorMeter).Sequence != 4 {
		t.Fatal("manufacturer data after service was lost")
	}
	apply("first", now.Add(32*time.Second), service('y', 0, 6))
	changed := store.Snapshot()["first"]
	if changed.Type != 'y' || changed.Data != nil || *changed.Battery != 6 || first.Data.(outdoormeter.OutdoorMeter).Sequence != 2 {
		t.Fatalf("type changed inconsistently: %+v", changed)
	}
}

func TestCacheBoundsAndInvalidPayload(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewStore(time.Minute)
	observe := func(fields ...protocol.ADField) error {
		_, err := store.Observe(protocol.Advertisement{MAC: "outdoor", ReceivedAt: now, Fields: fields})
		return err
	}
	for i := range maxPendingPerDevice + 1 {
		err := observe(outdoor(byte(i)))
		if (i == maxPendingPerDevice && !errors.Is(err, ErrCacheFull)) || (i < maxPendingPerDevice && err != nil) {
			t.Fatalf("unexpected cache result for %d: %v", i, err)
		}
	}
	if err := observe(service('w', 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := observe(service('H')); err == nil {
		t.Fatal("didn't flag truncated service packet")
	}
	state := store.Snapshot()["outdoor"]
	if state.Type != 'w' || state.Data.(outdoormeter.OutdoorMeter).Sequence != maxPendingPerDevice {
		t.Fatalf("invalid data overwrote reading: %+v", state)
	}
}
