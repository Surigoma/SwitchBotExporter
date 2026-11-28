package curtain

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data Curtain) {
	PublishPosition(p, data.PositionData)
}

func PublishPosition(p reporting.Output, data PositionData) {
	p.Gauge("_position", float64(data.Position))
	p.Gauge("_light_level", float64(data.LightLevel))
	p.Enum("_movement", data.Movement, "stationary", "movement")
}
