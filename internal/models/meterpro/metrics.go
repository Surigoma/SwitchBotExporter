package meterpro

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data MeterPro) {
	reporting.Climate(p, data.Climate)
}

func PublishCO2(p reporting.Output, data MeterProCO2) {
	reporting.Climate(p, data.Climate)
	p.Gauge("_co2", float64(data.CO2))
}
