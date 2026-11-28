package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/ble"
)

type outputObservation struct {
	ReceivedAt       time.Time         `json:"receivedAt"`
	Source           string            `json:"source"`
	MAC              string            `json:"mac"`
	RSSI             *int16            `json:"rssi,omitempty"`
	RSSIChanged      bool              `json:"rssiChanged"`
	ServiceData      map[string]string `json:"serviceData,omitempty"`
	ManufacturerData map[string]string `json:"manufacturerData,omitempty"`
	Fields           []outputField     `json:"switchbotFields,omitempty"`
}

type outputField struct {
	ADType     uint8  `json:"adType"`
	ID         string `json:"id"`
	PayloadHex string `json:"payloadHex"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("switchbot-ble-poc", flag.ContinueOnError)
	flags.SetOutput(stderr)
	adapter := flags.String("adapter", "hci0", "BlueZ adapter")
	duration := flags.Duration("duration", 10*time.Second, "length of the observation window")
	mac := flags.String("mac", "", "capture only this Bluetooth MAC address")
	debug := flags.Bool("debug", false, "log malformed advertisement properties")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *duration <= 0 {
		fmt.Fprintln(stderr, "expected flags and a positive --duration")
		return 2
	}
	if _, valid := ble.NormalizeMAC(*mac); *mac != "" && !valid {
		fmt.Fprintln(stderr, "invalid --mac address")
		return 2
	}
	if *debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	encoder := json.NewEncoder(stdout)
	observations := 0
	err := ble.ScanBlueZ(ctx, *adapter, *mac, *duration, func(obs ble.Observation) error {
		line := outputObservation{
			ReceivedAt:       obs.ReceivedAt.UTC(),
			Source:           obs.Source,
			MAC:              obs.MAC,
			RSSI:             obs.RSSI,
			RSSIChanged:      obs.RSSIChanged,
			ServiceData:      obs.ServiceData,
			ManufacturerData: obs.ManufacturerData,
		}
		for _, field := range obs.Fields {
			line.Fields = append(line.Fields, outputField{
				ADType:     field.Type,
				ID:         fmt.Sprintf("%04x", field.ID),
				PayloadHex: hex.EncodeToString(field.Payload),
			})
		}
		if err := encoder.Encode(line); err != nil {
			return err
		}
		observations++
		return nil
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		if errors.Is(err, context.Canceled) {
			return 130
		}
		return 1
	}
	fmt.Fprintf(stderr, "scan completed: %d observations\n", observations)
	return 0
}
