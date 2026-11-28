package hub2

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data Hub2) {
	p.Gauge("_sequence", float64(data.Sequence))
	p.Gauge("_light_level", float64(data.LightLevel))
	reporting.Climate(p, data.Climate)
}
