package metrics

import "github.com/Surigoma/SwitchBotExporter/internal/models/reporting"

type projection struct {
	prefix  string
	labels  sample
	samples map[sample]float64
}

func (p projection) Gauge(suffix string, value float64) {
	series := p.labels
	series.metric = p.prefix + suffix
	p.samples[series] = value
}

func (p projection) Enum(suffix, current string, options ...string) {
	for _, option := range options {
		series := p.labels
		series.metric = p.prefix + suffix
		series.state = option
		p.samples[series] = reporting.Number(option == current)
	}
}
