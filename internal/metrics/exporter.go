package metrics

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Surigoma/SwitchBotExporter/internal/device"
	"github.com/Surigoma/SwitchBotExporter/internal/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Exporter struct {
	registry    *prometheus.Registry
	devices     *collector
	success     prometheus.Gauge
	errors      prometheus.Counter
	duration    prometheus.Histogram
	seen        prometheus.Gauge
	lastSuccess prometheus.Gauge
}

func New() *Exporter {
	e := &Exporter{
		registry: prometheus.NewRegistry(),
		devices:  &collector{samples: make(map[sample]float64)},
		success:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "switchbot_scan_success", Help: "Whether the latest completed scan succeeded."}),
		errors:   prometheus.NewCounter(prometheus.CounterOpts{Name: "switchbot_scan_errors_total", Help: "Number of failed scan attempts."}),
		duration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "switchbot_scan_duration_seconds", Help: "Time spent on a scan attempt, including setup and cleanup.",
			Buckets: []float64{0.5, 1, 2.5, 5, 7.5, 10, 15, 30},
		}),
		seen:        prometheus.NewGauge(prometheus.GaugeOpts{Name: "switchbot_devices_seen", Help: "Number of distinct devices seen in the latest successful scan."}),
		lastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{Name: "switchbot_last_success_timestamp", Help: "Unix timestamp of the latest successful scan."}),
	}
	e.registry.MustRegister(e.devices, e.success, e.errors, e.duration, e.seen, e.lastSuccess)
	return e
}

func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{})
}

func (e *Exporter) RecordScan(start, end time.Time, err error, seen int) {
	e.duration.Observe(end.Sub(start).Seconds())
	if err != nil {
		e.success.Set(0)
		e.errors.Inc()
		return
	}
	e.success.Set(1)
	e.seen.Set(float64(seen))
	e.lastSuccess.Set(float64(end.Unix()))
}

type sample struct {
	metric string
	mac    string
	name   string
	key    string
	state  string
}

type collector struct {
	// Only the scan loop writes samples. Concurrent scrapes read published
	// immutable metrics, including a complete set of options for each enum.
	samples   map[sample]float64
	published atomic.Pointer[[]prometheus.Metric]
}

func (*collector) Describe(chan<- *prometheus.Desc) {}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	if values := c.published.Load(); values != nil {
		for _, value := range *values {
			ch <- value
		}
	}
}

func (e *Exporter) Publish(states map[string]device.State, names map[string]string) {
	for mac, reading := range states {
		name, configured := names[mac]
		model, supported := models.Lookup(reading.Type)
		if !configured || !supported || model.Prefix == "" {
			continue
		}
		labels := sample{mac: mac, name: name}
		common := func(metric string, value float64) {
			series := labels
			series.metric = metric
			series.key = model.Prefix
			e.devices.samples[series] = value
		}
		if reading.Battery != nil {
			common("switchbot_batt", float64(*reading.Battery))
		}
		if reading.RSSI != nil {
			common("switchbot_rssi", float64(*reading.RSSI))
		}
		if !reading.UpdatedAt.IsZero() {
			common("switchbot_update", float64(reading.UpdatedAt.Unix()))
		}
		if model.Publish != nil {
			model.Publish(projection{model.Prefix, labels, e.devices.samples}, reading.Data)
		}
	}
	values := make([]prometheus.Metric, 0, len(e.devices.samples))
	descs := make(map[string]*prometheus.Desc)
	for labels, value := range e.devices.samples {
		names := []string{"mac", "name"}
		valuesForLabels := []string{labels.mac, labels.name}
		if labels.key != "" {
			names = append(names, "key")
			valuesForLabels = append(valuesForLabels, labels.key)
		}
		if labels.state != "" {
			names = append(names, labels.metric)
			valuesForLabels = append(valuesForLabels, labels.state)
		}
		desc, ok := descs[labels.metric]
		if !ok {
			desc = prometheus.NewDesc(labels.metric, "Switch Bot", names, nil)
			descs[labels.metric] = desc
		}
		values = append(values, prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, valuesForLabels...))
	}
	e.devices.published.Store(&values)
}
