package blindtilt

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data BlindTilt) {
	p.Gauge("_tilt", float64(data.Tilt))
	p.Gauge("_light_level", float64(data.LightLevel))
	p.Enum("_movement", data.Movement, "stationary", "movement")
}
