package lockpro

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data LockPro) {
	p.Gauge("_sequence", float64(data.Sequence))
	p.Gauge("_door_open", reporting.Number(data.DoorOpen))
	p.Gauge("_charging", reporting.Number(data.Charging))
	p.Enum("_lock", data.Lock, LockStates()...)
}
