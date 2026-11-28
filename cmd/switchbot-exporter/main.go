package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/app"
	"github.com/Surigoma/SwitchBotExporter/internal/ble"
	"github.com/Surigoma/SwitchBotExporter/internal/config"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var scan bool
	if len(args) > 0 {
		switch args[0] {
		case "version":
			if len(args) != 1 {
				fmt.Fprintln(stderr, "version takes no arguments")
				return 2
			}
			fmt.Fprintf(stdout, "Version: %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\n", version, gitCommit, buildDate, runtime.Version())
			return 0
		case "scan":
			scan = true
			args = args[1:]
		}
	}

	flags := flag.NewFlagSet("switchbot-exporter", flag.ContinueOnError)
	flags.SetOutput(stderr)
	duration, filename := 5*time.Second, "./nametable.json"
	if scan {
		duration, filename = 10*time.Second, ""
	}
	settings := app.Settings{}
	flags.StringVar(&settings.Adapter, "adapter", "hci0", "BlueZ adapter")
	flags.DurationVar(&settings.Duration, "scan-duration", duration, "duration of a BLE scan")
	flags.DurationVar(&settings.CacheTTL, "cache-ttl", 30*time.Second, "expiration of pending manufacturer data")
	flags.StringVar(&filename, "config", filename, "path to the MAC-to-name JSON file")
	logLevel := flags.String("log-level", "info", "debug, info, warn or error")
	var raw bool
	var listen string
	if scan {
		flags.BoolVar(&raw, "raw", false, "emit advertisement observations as JSON Lines")
	} else {
		flags.StringVar(&listen, "listen", ":8000", "HTTP listen address")
		flags.DurationVar(&settings.Interval, "scan-interval", 10*time.Second, "target interval between scan starts")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || !ble.ValidAdapterName(settings.Adapter) || settings.Duration <= 0 || settings.CacheTTL <= 0 || !scan && settings.Interval <= 0 {
		fmt.Fprintln(stderr, "expected valid flags, an hci adapter and positive durations")
		return 2
	}
	if !scan {
		if _, _, err := net.SplitHostPort(listen); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		fmt.Fprintln(stderr, "log level must be debug, info, warn or error")
		return 2
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	if filename != "" || !scan {
		names, err := config.LoadNames(filename)
		if err != nil {
			fmt.Fprintf(stderr, "load name table %q: %v\n", filename, err)
			return 2
		}
		settings.Names = names
	}
	application := app.App{Scan: ble.ScanBlueZ, Logger: logger}
	if scan {
		if err := application.ScanOnce(ctx, settings, raw, stdout); err != nil {
			logger.Error("scan failed", "error", err)
			return 1
		}
		return 0
	}

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		logger.Error("HTTP listen failed", "error", err)
		return 1
	}
	logger.Info("HTTP listening", "address", listener.Addr())
	if err := application.Serve(ctx, listener, settings); err != nil {
		logger.Error("exporter stopped with an error", "error", err)
		return 1
	}
	return 0
}
