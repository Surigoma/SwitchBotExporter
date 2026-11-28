package curtain3

import (
	"github.com/Surigoma/SwitchBotExporter/internal/models/curtain"
	"github.com/Surigoma/SwitchBotExporter/internal/models/reporting"
)

func Publish(p reporting.Output, data Curtain3) {
	curtain.PublishPosition(p, data.PositionData)
}
