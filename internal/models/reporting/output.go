package reporting

import "github.com/Surigoma/SwitchBotExporter/internal/protocol"

// Output is the metric vocabulary used by individual models.
type Output interface {
	Gauge(suffix string, value float64)
	Enum(suffix, current string, options ...string)
}

func Climate(out Output, value protocol.Climate) {
	out.Gauge("_temp", value.Temperature)
	out.Gauge("_humid", float64(value.Humidity))
}

func Number(flag bool) float64 {
	if flag {
		return 1
	}
	return 0
}
