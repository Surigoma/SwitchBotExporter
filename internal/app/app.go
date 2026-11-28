package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/ble"
	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/metrics"
)

type ScanFunc func(context.Context, string, string, time.Duration, func(ble.Observation) error) error

type Settings struct {
	Adapter  string
	Duration time.Duration
	Interval time.Duration
	CacheTTL time.Duration
	Names    map[string]string
}

type App struct {
	Scan   ScanFunc
	Logger *slog.Logger
}

func (a App) Serve(ctx context.Context, listener net.Listener, settings Settings) error {
	exporter := metrics.New()
	routes := http.NewServeMux()
	routes.Handle("/metrics", exporter.Handler())
	server := &http.Server{Handler: routes, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	httpDone := make(chan error, 1)
	go func() {
		httpDone <- server.Serve(listener)
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	store := device.NewStore(settings.CacheTTL)
	scansDone := make(chan struct{})
	go func() {
		a.runScans(ctx, settings, store, exporter)
		close(scansDone)
	}()
	expireDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		defer close(expireDone)
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				store.Expire(now)
			}
		}
	}()

	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-httpDone:
	}
	cancel()
	<-scansDone
	<-expireDone

	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	shutdownErr := server.Shutdown(shutdown)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr)
}

func (a App) runScans(ctx context.Context, settings Settings, store *device.Store, exporter *metrics.Exporter) {
	next := time.Now()
	backoff := time.Second
	for ctx.Err() == nil {
		if !wait(ctx, time.Until(next)) {
			return
		}
		start := time.Now()
		seen := make(map[string]struct{})
		err := a.Scan(ctx, settings.Adapter, "", settings.Duration,
			a.observer(settings.Names, store, seen))
		if ctx.Err() != nil {
			return
		}
		now := time.Now()
		if err == nil {
			exporter.Publish(store.Snapshot(), settings.Names)
		}
		exporter.RecordScan(start, now, err, len(seen))

		earliest := now
		if err == nil {
			backoff = time.Second
		} else {
			a.Logger.Warn("BLE scan failed", "error", err, "backoff", backoff)
			earliest = now.Add(backoff)
			backoff = min(2*backoff, time.Minute)
		}
		// Keep the same slot grid and skip starts that fell during this scan
		// or during recovery.
		next = next.Add(settings.Interval)
		if next.Before(earliest) {
			late := earliest.Sub(next)
			next = earliest.Add((settings.Interval - late%settings.Interval) % settings.Interval)
		}
	}
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

func (a App) ScanOnce(ctx context.Context, settings Settings, raw bool, stdout io.Writer) error {
	encoder := json.NewEncoder(stdout)
	store := device.NewStore(settings.CacheTTL)
	observe := a.observer(settings.Names, store, nil)
	if raw {
		observe = func(obs ble.Observation) error {
			if allowed(settings.Names, obs.MAC) {
				return encoder.Encode(toRawObservation(obs))
			}
			return nil
		}
	}
	if err := a.Scan(ctx, settings.Adapter, "", settings.Duration, observe); err != nil {
		return err
	}
	if raw {
		return nil
	}
	return encoder.Encode(store.Snapshot())
}

func (a App) observer(names map[string]string, store *device.Store, seen map[string]struct{}) func(ble.Observation) error {
	var lastWarning time.Time
	return func(obs ble.Observation) error {
		if !allowed(names, obs.MAC) {
			return nil
		}
		recognized, err := store.Observe(obs.Advertisement)
		if recognized && seen != nil {
			seen[obs.MAC] = struct{}{}
		}
		if err != nil && (lastWarning.IsZero() || obs.ReceivedAt.Sub(lastWarning) >= time.Second) {
			lastWarning = obs.ReceivedAt
			a.Logger.Warn("BLE advertisement partially rejected", "error", err)
		}
		return nil
	}
}

func allowed(names map[string]string, mac string) bool {
	if names == nil {
		return true
	}
	_, ok := names[mac]
	return ok
}
