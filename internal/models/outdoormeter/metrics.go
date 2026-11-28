package outdoormeter

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data OutdoorMeter) {
	reporting.Climate(p, data.Climate)
}
