package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/models"
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

const (
	maxPending          = 256
	maxPendingPerDevice = 8
	maxPayload          = 255
)

var ErrCacheFull = errors.New("manufacturer data cache limit reached")

type State struct {
	Type      protocol.DeviceType
	RSSI      *int16
	UpdatedAt time.Time
	protocol.Reading
}

func (s State) MarshalJSON() ([]byte, error) {
	name := "Undefined"
	if info, ok := models.Lookup(s.Type); ok {
		name = info.Name
	}
	var rssi int16 = -999
	if s.RSSI != nil {
		rssi = *s.RSSI
	}
	var update int64 = -1
	if !s.UpdatedAt.IsZero() {
		update = s.UpdatedAt.Unix()
	}
	data := make(map[string]json.RawMessage)
	if s.Data != nil {
		value, err := json.Marshal(s.Data)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(value, &data); err != nil {
			return nil, err
		}
	}
	if s.Battery != nil {
		value, err := json.Marshal(*s.Battery)
		if err != nil {
			return nil, err
		}
		data["battery"] = value
	}
	return json.Marshal(struct {
		Type     protocol.DeviceType        `json:"type"`
		TypeName string                     `json:"typeStr"`
		RSSI     int16                      `json:"rssi"`
		Update   int64                      `json:"update"`
		Data     map[string]json.RawMessage `json:"data"`
	}{s.Type, name, rssi, update, data})
}

type pendingField struct {
	mac     string
	at      time.Time
	payload []byte
}

type Store struct {
	mu      sync.Mutex
	ttl     time.Duration
	devices map[string]State
	pending []pendingField
}

func NewStore(ttl time.Duration) *Store {
	return &Store{ttl: ttl, devices: make(map[string]State)}
}

// Snapshot returns read-only observations. Future updates replace their
// readings without mutating data referenced by this snapshot.
func (s *Store) Snapshot() map[string]State {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := make(map[string]State, len(s.devices))
	for mac, state := range s.devices {
		snapshot[mac] = state
	}
	return snapshot
}

func (s *Store) Expire(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire(now)
}

func (s *Store) expire(now time.Time) {
	kept := s.pending[:0]
	for _, entry := range s.pending {
		if now.Before(entry.at.Add(s.ttl)) {
			kept = append(kept, entry)
		}
	}
	clear(s.pending[len(kept):])
	s.pending = kept
}

// Observe associates fields with a device and reports whether this observation
// contained a known type or a valid reading for one.
func (s *Store) Observe(ad protocol.Advertisement) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expire(ad.ReceivedAt)

	state, exists := s.devices[ad.MAC]
	if ad.RSSI != nil {
		rssi := *ad.RSSI
		state.RSSI = &rssi
	}

	var matched bool
	var problems []error
	// Resolve all service data before consuming any manufacturer data from
	// this observation. Across notifications the original order is preserved.
	for _, field := range ad.Fields {
		if field.Type != protocol.ServiceData16 || !field.IsSwitchBot() {
			continue
		}
		exists = true
		kind, reading, err := models.DecodeService(field.Payload)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		_, known := models.Lookup(kind)
		_, previous := models.Lookup(state.Type)
		if !known {
			if !previous {
				state.Type = kind
			}
			continue
		}
		if !models.SameModel(state.Type, kind) {
			state.Reading = protocol.Reading{}
			state.UpdatedAt = time.Time{}
		}
		state.Type = kind
		matched = true
		state.apply(reading, ad.ReceivedAt)
		problems = append(problems, s.replay(ad.MAC, &state)...)
	}

	for _, field := range ad.Fields {
		if field.Type != protocol.ManufacturerData || !field.IsSwitchBot() {
			continue
		}
		exists = true
		if len(field.Payload) > maxPayload {
			problems = append(problems, fmt.Errorf("manufacturer data exceeds %d bytes", maxPayload))
			continue
		}
		if _, known := models.Lookup(state.Type); !known {
			if s.cache(ad.MAC, ad.ReceivedAt, field.Payload) {
				problems = append(problems, ErrCacheFull)
			}
			continue
		}
		reading, err := models.DecodeManufacturer(state.Type, field.Payload)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if !reading.Empty() {
			matched = true
			state.apply(reading, ad.ReceivedAt)
		}
	}

	if exists {
		s.devices[ad.MAC] = state
	}
	return matched, errors.Join(problems...)
}

func (s *State) apply(reading protocol.Reading, at time.Time) {
	if reading.Empty() {
		return
	}
	if reading.Battery != nil {
		s.Battery = reading.Battery
	}
	if reading.Data != nil {
		s.Data = reading.Data
	}
	if at.After(s.UpdatedAt) {
		s.UpdatedAt = at
	}
}

func (s *Store) replay(mac string, state *State) []error {
	kept := s.pending[:0]
	var problems []error
	for _, entry := range s.pending {
		if entry.mac != mac {
			kept = append(kept, entry)
			continue
		}
		reading, err := models.DecodeManufacturer(state.Type, entry.payload)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		state.apply(reading, entry.at)
	}
	clear(s.pending[len(kept):])
	s.pending = kept
	return problems
}

func (s *Store) cache(mac string, at time.Time, payload []byte) bool {
	first := -1
	count := 0
	for i, entry := range s.pending {
		if entry.mac == mac {
			if first < 0 {
				first = i
			}
			count++
		}
	}
	drop := -1
	switch {
	case count >= maxPendingPerDevice:
		drop = first
	case len(s.pending) >= maxPending:
		drop = 0
	}
	if drop >= 0 {
		copy(s.pending[drop:], s.pending[drop+1:])
		s.pending = s.pending[:len(s.pending)-1]
	}
	s.pending = append(s.pending, pendingField{mac, at, append([]byte(nil), payload...)})
	return drop >= 0
}
