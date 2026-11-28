package ble

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	bluezName           = "org.bluez"
	adapterInterface    = "org.bluez.Adapter1"
	deviceInterface     = "org.bluez.Device1"
	objectManager       = "org.freedesktop.DBus.ObjectManager"
	propertiesInterface = "org.freedesktop.DBus.Properties"
	busInterface        = "org.freedesktop.DBus"
	signalCapacity      = 256
	callTimeout         = 3 * time.Second
)

var adapterName = regexp.MustCompile("^hci[0-9]+$")
var errSignalOverflow = errors.New("BlueZ notification queue overflow")

func ValidAdapterName(adapter string) bool {
	return adapterName.MatchString(adapter)
}

// ScanBlueZ opens one private discovery session. Closing the connection also
// releases the session if a start or stop reply is lost.
func ScanBlueZ(ctx context.Context, adapter, mac string, duration time.Duration, emit func(Observation) error) (scanErr error) {
	if !ValidAdapterName(adapter) || duration <= 0 || emit == nil {
		return errors.New("expected an hci adapter, a positive scan duration and an observation handler")
	}
	if mac != "" {
		var ok bool
		mac, ok = NormalizeMAC(mac)
		if !ok {
			return errors.New("invalid target MAC address")
		}
	}
	path := dbus.ObjectPath("/org/bluez/" + adapter)
	queue := signalQueue{events: make(chan queuedSignal, signalCapacity), overflow: make(chan struct{})}
	conn, err := connect(ctx, &queue)
	if err != nil {
		return fmt.Errorf("connect to system D-Bus: %w", err)
	}
	defer conn.Close()

	owner, devices, err := prepare(ctx, conn, path)
	if err != nil {
		return err
	}

	object := conn.Object(owner, path)
	options := properties{
		"Transport":     dbus.MakeVariant("le"),
		"DuplicateData": dbus.MakeVariant(true),
	}
	if err := call(ctx, object, adapterInterface+".SetDiscoveryFilter", options).Err; err != nil {
		return fmt.Errorf("set LE discovery filter: %w", err)
	}
	queue.arm()
	if err := call(ctx, object, adapterInterface+".StartDiscovery").Err; err != nil {
		return fmt.Errorf("start LE discovery: %w", err)
	}
	defer func() {
		if err := call(context.Background(), object, adapterInterface+".StopDiscovery").Err; err != nil {
			scanErr = errors.Join(scanErr, fmt.Errorf("stop LE discovery: %w", err))
		}
	}()

	deadline := time.Now().Add(duration)
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	handle := func(event queuedSignal) error {
		if !event.at.Before(deadline) {
			return nil
		}
		return devices.handle(ctx, conn, owner, mac, event, emit)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-conn.Context().Done():
			return errors.New("system D-Bus disconnected")
		case <-queue.overflow:
			return errSignalOverflow
		case event := <-queue.events:
			if err := handle(event); err != nil {
				return err
			}
		case <-timer.C:
			queue.seal()
			select {
			case <-queue.overflow:
				return errSignalOverflow
			default:
			}
			for event := range queue.events {
				if err := handle(event); err != nil {
					return err
				}
			}
			return ctx.Err()
		}
	}
}

func connect(ctx context.Context, queue *signalQueue) (*dbus.Conn, error) {
	setup, cancelSetup := context.WithTimeout(ctx, callTimeout)
	defer cancelSetup()
	// A separate lifetime lets cancellation stop the handshake without
	// shutting the bus before StopDiscovery can run.
	lifetime, cancelLifetime := context.WithCancel(context.Background())
	stop := context.AfterFunc(setup, cancelLifetime)
	conn, err := dbus.ConnectSystemBus(dbus.WithContext(lifetime), dbus.WithSignalHandler(queue))
	if err != nil {
		stop()
		cancelLifetime()
		if setup.Err() != nil {
			return nil, setup.Err()
		}
		return nil, err
	}
	if !stop() || setup.Err() != nil {
		conn.Close()
		cancelLifetime()
		return nil, setup.Err()
	}
	// Close cancels conn.Context(), which in turn releases the lifetime.
	context.AfterFunc(conn.Context(), cancelLifetime)
	return conn, nil
}

func prepare(ctx context.Context, conn *dbus.Conn, adapter dbus.ObjectPath) (string, *tracker, error) {
	matches := [][]dbus.MatchOption{
		{dbus.WithMatchSender(bluezName), dbus.WithMatchInterface(objectManager), dbus.WithMatchMember("InterfacesAdded")},
		{dbus.WithMatchSender(bluezName), dbus.WithMatchInterface(objectManager), dbus.WithMatchMember("InterfacesRemoved")},
		{dbus.WithMatchSender(bluezName), dbus.WithMatchInterface(propertiesInterface), dbus.WithMatchMember("PropertiesChanged"), dbus.WithMatchPathNamespace(adapter)},
		{dbus.WithMatchSender(busInterface), dbus.WithMatchInterface(busInterface), dbus.WithMatchMember("NameOwnerChanged"), dbus.WithMatchArg(0, bluezName)},
	}
	setup, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	for _, match := range matches {
		if err := conn.AddMatchSignalContext(setup, match...); err != nil {
			return "", nil, fmt.Errorf("subscribe to BlueZ: %w", err)
		}
	}

	var owner string
	if err := call(setup, conn.BusObject(), busInterface+".GetNameOwner", bluezName).Store(&owner); err != nil {
		return "", nil, fmt.Errorf("find BlueZ service: %w", err)
	}
	var objects map[dbus.ObjectPath]map[string]properties
	if err := call(setup, conn.Object(owner, "/"), objectManager+".GetManagedObjects").Store(&objects); err != nil {
		return "", nil, fmt.Errorf("list BlueZ objects: %w", err)
	}
	fields, exists := objects[adapter][adapterInterface]
	if !exists {
		return "", nil, fmt.Errorf("Bluetooth adapter %s not found", adapter)
	}
	if powered, _ := fields["Powered"].Value().(bool); !powered {
		return "", nil, fmt.Errorf("Bluetooth adapter %s is not powered", adapter)
	}

	devices := &tracker{adapter: adapter, devices: make(map[dbus.ObjectPath]device)}
	for path, interfaces := range objects {
		if fields, exists := interfaces[deviceInterface]; exists {
			devices.remember(path, fields)
		}
	}
	return owner, devices, nil
}

func call(ctx context.Context, object dbus.BusObject, method string, args ...any) *dbus.Call {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	return object.CallWithContext(ctx, method, 0, args...)
}

func (t *tracker) handle(ctx context.Context, conn *dbus.Conn, owner, target string, event queuedSignal, emit func(Observation) error) error {
	signal := event.signal
	if signal.Sender == busInterface && signal.Name == busInterface+".NameOwnerChanged" && len(signal.Body) == 3 {
		name, nameOK := signal.Body[0].(string)
		oldOwner, oldOK := signal.Body[1].(string)
		if nameOK && oldOK && name == bluezName && oldOwner == owner {
			return errors.New("BlueZ service owner changed")
		}
	}
	if signal.Sender != owner {
		return nil
	}

	var path dbus.ObjectPath
	var fields properties
	var invalidated []string
	var source string
	switch signal.Name {
	case objectManager + ".InterfacesAdded":
		var interfaces map[string]properties
		if err := dbus.Store(signal.Body, &path, &interfaces); err != nil {
			return nil
		}
		fields = interfaces[deviceInterface]
		source = "interfacesAdded"
	case objectManager + ".InterfacesRemoved":
		var interfaces []string
		if err := dbus.Store(signal.Body, &path, &interfaces); err != nil {
			return nil
		}
		for _, name := range interfaces {
			if name == adapterInterface && path == t.adapter {
				return errors.New("Bluetooth adapter disappeared")
			}
			if name == deviceInterface {
				delete(t.devices, path)
			}
		}
		return nil
	case propertiesInterface + ".PropertiesChanged":
		var name string
		if err := dbus.Store(signal.Body, &name, &fields, &invalidated); err != nil {
			return nil
		}
		path = signal.Path
		if name == adapterInterface && path == t.adapter {
			if value, present := fields["Powered"]; present && value.Value() != true {
				return errors.New("Bluetooth adapter powered off")
			}
			for _, property := range invalidated {
				if property == "Powered" {
					return errors.New("Bluetooth adapter power state invalidated")
				}
			}
		}
		if name != deviceInterface || !strings.HasPrefix(string(path), string(t.adapter)+"/") {
			return nil
		}
		source = "propertiesChanged"
		if _, known := t.devices[path]; !known {
			var identity properties
			err := call(ctx, conn.Object(owner, path), propertiesInterface+".GetAll", deviceInterface).Store(&identity)
			if err != nil {
				var busErr dbus.Error
				if errors.As(err, &busErr) && (busErr.Name == busInterface+".Error.UnknownObject" || busErr.Name == busInterface+".Error.UnknownMethod") {
					return nil
				}
				return fmt.Errorf("read BlueZ device identity: %w", err)
			}
			t.remember(path, identity)
		}
	default:
		return nil
	}

	if len(fields) == 0 && len(invalidated) == 0 {
		return nil
	}
	if observation, ok := t.observe(path, fields, invalidated, event.at, source, target); ok {
		if err := emit(observation); err != nil {
			return fmt.Errorf("write BLE observation: %w", err)
		}
	}
	return nil
}

type queuedSignal struct {
	at     time.Time
	signal *dbus.Signal
}

type signalQueue struct {
	mu       sync.Mutex
	events   chan queuedSignal
	overflow chan struct{}
	full     bool
	armed    bool
	sealed   bool
}

// DeliverSignal is called by the D-Bus receiver. It must not wait for the
// consumer, which can itself be waiting for a D-Bus method reply.
func (q *signalQueue) DeliverSignal(_, _ string, signal *dbus.Signal) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.armed || q.sealed || q.full {
		return
	}
	select {
	case q.events <- queuedSignal{at: time.Now(), signal: signal}:
	default:
		q.full = true
		close(q.overflow)
	}
}

func (q *signalQueue) arm() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.armed = true
}

func (q *signalQueue) seal() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sealed = true
	close(q.events)
}
