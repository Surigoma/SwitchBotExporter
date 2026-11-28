//go:build pprof

package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
)

// Profiling builds serve net/http/pprof on SWITCHBOT_PPROF, for example
// 127.0.0.1:6060. Build with -tags pprof.
func init() {
	address := os.Getenv("SWITCHBOT_PPROF")
	if address == "" {
		return
	}
	go func() {
		slog.Info("pprof listening", "address", address)
		if err := http.ListenAndServe(address, nil); err != nil {
			slog.Error("pprof stopped", "error", err)
		}
	}()
}
