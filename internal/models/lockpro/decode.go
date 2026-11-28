package lockpro

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type LockPro struct {
	Sequence       uint8  `json:"sequence_num"`
	Calibrated     bool   `json:"calibrated"`
	Lock           string `json:"lock"`
	DoorOpen       bool   `json:"door_open"`
	AutoLockPaused bool   `json:"auto_lock_paused"`
	Charging       bool   `json:"charging"`
	DoubleLockMode bool   `json:"double_lock_mode"`
	UnclosedAlarm  bool   `json:"unclosed_door_alarm"`
	UnlockedAlarm  bool   `json:"unlocked_alarm"`
}

// DecodeManufacturer reads the Lock Pro payload after the company ID. Battery
// is carried here rather than in the service data.
func DecodeManufacturer(p []byte) protocol.Reading {
	battery := p[9] & 0x7f
	return protocol.Reading{Battery: &battery, Data: LockPro{
		Sequence:       p[6],
		Calibrated:     protocol.Bit(p[7], 7),
		Lock:           LockState((p[7] >> 3) & 0x0f),
		DoorOpen:       protocol.Bit(p[8], 4),
		AutoLockPaused: protocol.Bit(p[8], 3),
		Charging:       protocol.Bit(p[9], 7),
		DoubleLockMode: protocol.Bit(p[10], 7),
		UnclosedAlarm:  protocol.Bit(p[11], 7),
		UnlockedAlarm:  protocol.Bit(p[11], 6),
	}}
}

func LockStates() []string {
	return []string{"Locked", "Unlocked", "Locking", "Unlocking", "LockingStop(Stuck)", "UnlockingStop(Stuck)", "Not fully locked(only for EU Lock Type)", "Half locked(only for Lock Ultra EU)", "Undefined"}
}

func LockState(value uint8) string {
	states := LockStates()
	if int(value) >= len(states)-1 {
		return "Undefined"
	}
	return states[value]
}
