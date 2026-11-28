package smartlock

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data SmartLock) {
	states := LockStates()
	p.Gauge("_sequence", float64(data.Sequence))
	p.Enum("_lock", data.Info.Status.Lock, states[:]...)
}
