package smartlock

import (
	"github.com/Surigoma/SwitchBotExporter/internal/protocol"
)

type LockStatus struct {
	Lock string `json:"lock"`
	Door string `json:"door"`
}

type LockInfo struct {
	Calibrated   bool       `json:"calibrated"`
	UpdateSource string     `json:"update_source"`
	Status       LockStatus `json:"status"`
}

type LockConfig struct {
	DoubleLockMode           bool   `json:"double_lock_mode"`
	UnclosedDoorTimeoutAlarm bool   `json:"unclosed_door_timeout_alarm"`
	PauseAutomaticLocking    string `json:"pause_automatic_locking"`
}

type SmartLock struct {
	Sequence uint8      `json:"sequence_num"`
	Info     LockInfo   `json:"lock_info"`
	Config   LockConfig `json:"config"`
}

func DecodeManufacturer(p []byte) protocol.Reading {
	return protocol.Reading{Data: SmartLock{
		Sequence: p[6],
		Info: LockInfo{
			Calibrated:   protocol.Bit(p[7], 7),
			UpdateSource: protocol.Choice(protocol.Bit(p[7], 3), "primary_lock", "secondary_lock"),
			Status:       LockStatus{LockStates()[(p[7]>>4)&7], protocol.Choice(protocol.Bit(p[7], 2), "closed", "opened")},
		},
		Config: LockConfig{protocol.Bit(p[8], 7), protocol.Bit(p[8], 6), protocol.Choice(protocol.Bit(p[8], 1), "inactived", "activated")},
	}}
}

func LockStates() [8]string {
	return [8]string{"Locked", "Unlocked", "Locking", "Unlocking", "LockingStop(Stuck)", "UnlockingStop(Stuck)", "Not fully locked(only for EU Lock Type)", "Undefined"}
}
