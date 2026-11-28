package ble

import (
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
	"github.com/godbus/dbus/v5"
)

func TestObserveBlueZData(t *testing.T) {
	adapter := dbus.ObjectPath("/org/bluez/hci0")
	path := dbus.ObjectPath("/org/bluez/hci0/dev_00_11_22_33_44_55")
	identity := properties{"Address": dbus.MakeVariant("00:11:22:33:44:55"), "Adapter": dbus.MakeVariant(adapter)}
	tracker := tracker{adapter: adapter, devices: make(map[dbus.ObjectPath]device)}
	identity["RSSI"] = dbus.MakeVariant(int16(-100))
	tracker.remember(path, identity)
	now := time.Now()

	payload := []byte{0, 1, 2}
	manufacturer := properties{"ManufacturerData": dbus.MakeVariant(map[uint16]dbus.Variant{
		0x0969: dbus.MakeVariant(payload),
		0x0059: dbus.MakeVariant([]byte{3, 4}),
		0x004c: dbus.MakeVariant([]byte{5}),
		0x1234: dbus.MakeVariant("invalid payload"),
	})}
	first, ok := tracker.observe(path, manufacturer, nil, now, "propertiesChanged")
	if !ok || first.RSSI != nil || first.MAC != "00:11:22:33:44:55" || !first.ReceivedAt.Equal(now) {
		t.Fatalf("manufacturer-only observation or stale RSSI: %+v", first)
	}
	wantMfr := []protocol.ADField{
		{Type: protocol.ManufacturerData, ID: 0x0969, Payload: []byte{0, 1, 2}},
		{Type: protocol.ManufacturerData, ID: 0x0059, Payload: []byte{3, 4}},
	}
	payload[0] = 9
	if !reflect.DeepEqual(first.Fields, wantMfr) || first.ManufacturerData["004c"] != "05" || len(first.ManufacturerData) != 3 {
		t.Fatalf("manufacturer normalization: %+v", first)
	}

	service := properties{
		"RSSI": dbus.MakeVariant(int16(-53)),
		"ServiceData": dbus.MakeVariant(map[string]dbus.Variant{
			"0000FD3D-0000-1000-8000-00805F9B34FB": dbus.MakeVariant([]byte{0x54, 0x00}),
			"00000d00-0000-1000-8000-00805f9b34fb": dbus.MakeVariant([]byte{0x48}),
			"0000180a-0000-1000-8000-00805f9b34fb": dbus.MakeVariant([]byte{0x58}),
		}),
	}
	for i := range 2 {
		next, ok := tracker.observe(path, service, nil, now.Add(time.Duration(i+1)*time.Second), "propertiesChanged")
		want := []protocol.ADField{
			{Type: protocol.ServiceData16, ID: 0xfd3d, Payload: []byte{0x54, 0x00}},
			{Type: protocol.ServiceData16, ID: 0x0d00, Payload: []byte{0x48}},
		}
		if !ok || !next.RSSIChanged || next.RSSI == nil || *next.RSSI != -53 || !reflect.DeepEqual(next.Fields, want) ||
			len(next.ServiceData) != 3 || len(next.ManufacturerData) != 0 {
			t.Fatalf("service observation %d: %+v", i, next)
		}
	}
	withRSSI, ok := tracker.observe(path, manufacturer, nil, now, "propertiesChanged")
	if !ok || withRSSI.RSSIChanged || withRSSI.RSSI == nil || *withRSSI.RSSI != -53 {
		t.Fatalf("latest RSSI: %+v", withRSSI)
	}
	cleared, ok := tracker.observe(path, manufacturer, []string{"RSSI"}, now, "propertiesChanged")
	if !ok || cleared.RSSI != nil {
		t.Fatalf("invalidated RSSI: %+v", cleared)
	}
	tracker.targets = map[string]struct{}{"aa:bb:cc:dd:ee:ff": {}}
	if _, ok := tracker.observe(path, service, nil, now, "propertiesChanged"); ok {
		t.Fatal("reported a device outside the MAC filter")
	}
	tracker.targets = nil
	if !reflect.DeepEqual(first.ManufacturerData["0969"], hex.EncodeToString(wantMfr[0].Payload)) {
		t.Fatal("raw payload differs from normalized payload")
	}
}

func TestBlueZNotificationLifecycle(t *testing.T) {
	adapter := dbus.ObjectPath("/org/bluez/hci0")
	path := dbus.ObjectPath("/org/bluez/hci0/dev_00_11_22_33_44_55")
	tracker := tracker{adapter: adapter, devices: make(map[dbus.ObjectPath]device)}
	source := ":1.2"
	stopper := errors.New("output closed")
	callback := func(record Observation) error {
		if !record.RSSIChanged || record.Source != "interfacesAdded" {
			t.Errorf("unexpected observation: %+v", record)
		}
		return stopper
	}
	added := &dbus.Signal{
		Sender: source, Name: objectManager + ".InterfacesAdded",
		Body: []any{path, map[string]properties{deviceInterface: {
			"Address": dbus.MakeVariant("00:11:22:33:44:55"),
			"Adapter": dbus.MakeVariant(adapter),
			"RSSI":    dbus.MakeVariant(int16(-70)),
		}}},
	}
	if err := tracker.handle(t.Context(), nil, ":1.9", queuedSignal{signal: added}, callback); err != nil {
		t.Fatalf("unexpected sender was handled: %v", err)
	}
	if err := tracker.handle(t.Context(), nil, source, queuedSignal{signal: added}, callback); !errors.Is(err, stopper) {
		t.Fatalf("write failure was not returned: %v", err)
	}
	changes := []*dbus.Signal{
		{Sender: source, Path: adapter, Name: propertiesInterface + ".PropertiesChanged", Body: []any{adapterInterface, properties{"Powered": dbus.MakeVariant(false)}, []string{}}},
		{Sender: source, Name: objectManager + ".InterfacesRemoved", Body: []any{adapter, []string{adapterInterface}}},
		{Sender: busInterface, Name: busInterface + ".NameOwnerChanged", Body: []any{bluezName, source, ""}},
	}
	for _, signal := range changes {
		if err := tracker.handle(t.Context(), nil, source, queuedSignal{signal: signal}, callback); err == nil {
			t.Errorf("ignored a scan failure: %s", signal.Name)
		}
	}
}

func TestDevicePath(t *testing.T) {
	if got := DevicePath("/org/bluez/hci0", "c8:f6:75:d8:c4:83"); got != "/org/bluez/hci0/dev_C8_F6_75_D8_C4_83" {
		t.Fatalf("unexpected device path %s", got)
	}
}

func TestBodyAs(t *testing.T) {
	fields := properties{"RSSI": dbus.MakeVariant(int16(-40))}
	var name string
	var got properties
	var invalidated []string
	if !bodyAs([]any{deviceInterface, fields, []string{"TxPower"}}, &name, &got, &invalidated) ||
		name != deviceInterface || !reflect.DeepEqual(got, fields) || !reflect.DeepEqual(invalidated, []string{"TxPower"}) {
		t.Fatalf("fast path: %q %v %v", name, got, invalidated)
	}
	// dbus.Store converts compatible shapes the fast path does not cover.
	var path dbus.ObjectPath
	var interfaces []string
	if !bodyAs([]any{"/org/bluez/hci0", []string{adapterInterface}}, &path, &interfaces) || path != "/org/bluez/hci0" {
		t.Fatalf("fallback: %q %v", path, interfaces)
	}
	if bodyAs([]any{deviceInterface}, &name, &got) {
		t.Fatal("accepted an invalid body")
	}
}

func TestSignalQueueOverflow(t *testing.T) {
	queue := signalQueue{events: make(chan queuedSignal, 1), overflow: make(chan struct{})}
	queue.DeliverSignal("", "", &dbus.Signal{Name: "before start"})
	queue.arm()
	first := &dbus.Signal{Name: "first"}
	queue.DeliverSignal("", "", first)
	queue.DeliverSignal("", "", &dbus.Signal{Name: "excess"})
	select {
	case <-queue.overflow:
	default:
		t.Fatal("missing overflow notification")
	}
	queue.seal()
	queue.DeliverSignal("", "", &dbus.Signal{Name: "late"})
	if event := <-queue.events; event.signal != first || event.at.IsZero() {
		t.Fatalf("lost the first event: %+v", event)
	}
	if _, ok := <-queue.events; ok {
		t.Fatal("accepted a late event")
	}
}
