package ceilinglight

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data CeilingLight) {
	p.Gauge("_sequence", float64(data.Sequence))
	p.Gauge("_level", float64(data.Level))
	p.Gauge("_status", reporting.Number(data.Status))
	p.Gauge("_small_status", reporting.Number(data.SmallStatus))
	p.Gauge("_color_temp", float64(data.ColorTemperature))
}
