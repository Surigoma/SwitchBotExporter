package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/ble"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

const testMAC = "00:11:22:33:44:55"

func testAdvertisement() ble.Observation {
	rssi := int16(-50)
	return ble.Observation{Advertisement: protocol.Advertisement{
		MAC: testMAC, RSSI: &rssi, ReceivedAt: time.Unix(100, 0),
		Fields: []protocol.ADField{{Type: protocol.ServiceData16, ID: 0xfd3d, Payload: []byte{'T', 0, 0, 0, 0x95, 0x32}}},
	}, ServiceData: map[string]string{"0000fd3d-0000-1000-8000-00805f9b34fb": "540000009532"}}
}

func TestScanOnce(t *testing.T) {
	a := App{
		Scan: func(_ context.Context, _ string, _ string, _ time.Duration, emit func(ble.Observation) error) error {
			return emit(testAdvertisement())
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	settings := Settings{Duration: time.Second, CacheTTL: time.Minute}
	var output bytes.Buffer
	if err := a.ScanOnce(t.Context(), settings, false, &output); err != nil {
		t.Fatal(err)
	}
	// Use raw JSON for the heterogeneous data fields (including group/alert).
	var records map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(records[testMAC]), `"temp":21`) || !strings.Contains(string(records[testMAC]), `"type":84`) {
		t.Fatalf("unexpected JSON: %s", output.String())
	}
	output.Reset()
	if err := a.ScanOnce(t.Context(), settings, true, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"adType":22`) || !strings.Contains(output.String(), `"payloadHex":"540000009532"`) {
		t.Fatalf("unexpected raw output: %s", output.String())
	}
}

func TestHTTPRecoveryAndGracefulShutdown(t *testing.T) {
	stopped := make(chan struct{})
	attempt := 0
	a := App{
		Scan: func(_ context.Context, _ string, _ string, _ time.Duration, emit func(ble.Observation) error) error {
			attempt++
			if attempt == 1 {
				return errors.New("adapter temporarily unavailable")
			}
			return emit(testAdvertisement())
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	settings := Settings{Duration: time.Second, Interval: 50 * time.Millisecond, CacheTTL: time.Minute, Names: map[string]string{testMAC: "test"}}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		defer close(stopped)
		if err := a.Serve(ctx, listener, settings); err != nil {
			t.Error(err)
		}
	}()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	sawFailure := false
	for {
		response, err := client.Get("http://" + listener.Addr().String() + "/metrics")
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("metrics endpoint returned %s", response.Status)
		}
		output := string(body)
		sawFailure = sawFailure || strings.Contains(output, "switchbot_scan_success 0\n") && strings.Contains(output, "switchbot_scan_errors_total 1\n")
		if strings.Contains(output, "switchbot_scan_success 1\n") && strings.Contains(output, `switchbot_meter_th_temp{mac="00:11:22:33:44:55",name="test"} 21`) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("metric never published: %s", body)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !sawFailure {
		t.Error("failed scan was not observable before recovery")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down")
	}
}
