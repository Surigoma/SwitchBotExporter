package contactsensor

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

func Publish(p reporting.Output, data ContactSensor) {
	p.Enum("_state_PIR", data.State.PIR, "No one moves", "Someone is moving")
	p.Enum("_state_HAL", data.State.HAL, "door close", "door open", "timeout not close", "undefined")
	p.Enum("_light_level", data.LightLevel, "dark", "light")
	p.Gauge("_act_entrances", float64(data.Activity.Entrances))
	p.Gauge("_act_go_out", float64(data.Activity.GoOut))
	p.Gauge("_act_button_push", float64(data.Activity.ButtonPush))
}
