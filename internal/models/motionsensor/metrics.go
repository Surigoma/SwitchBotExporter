package motionsensor

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data MotionSensor) {
	p.Gauge("_latest", float64(data.Latest))
	p.Gauge("_led_status", reporting.Number(data.LEDState))
	p.Gauge("_iot_status", reporting.Number(data.IOTState))
	p.Enum("_PIR", data.PIR, "No one moves", "Someone is moving")
	p.Enum("_light_intensity", data.LightIntensity, "Reserve", "dark", "bright")
}
