package humidifier

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data Humidifier) {
	p.Gauge("_on", reporting.Number(data.On))
	p.Enum("_mode", data.Mode, Modes()...)
	if data.Mode == "target" {
		p.Gauge("_target_humid", float64(data.Target))
	}
}
