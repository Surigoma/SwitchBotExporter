package meterth

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data MeterTH) {
	reporting.Climate(p, data.Climate)
}
