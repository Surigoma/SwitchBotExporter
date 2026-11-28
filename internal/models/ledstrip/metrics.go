package ledstrip

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data LEDStrip) {
	p.Gauge("_sequence", float64(data.Sequence))
	p.Gauge("_on", reporting.Number(data.On))
	p.Gauge("_level", float64(data.Level))
	p.Enum("_mode", data.Mode, Modes()...)
}
