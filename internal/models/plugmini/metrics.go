package plugmini

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data PlugMini) {
	p.Gauge("_sequence", float64(data.Sequence))
	p.Gauge("_on", reporting.Number(data.On))
	p.Gauge("_overload", reporting.Number(data.Overload))
	p.Gauge("_wifi_rssi", float64(data.WiFiRSSI))
	p.Gauge("_power", data.Power)
}
