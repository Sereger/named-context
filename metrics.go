package context

import "github.com/prometheus/client_golang/prometheus"

type metrics interface {
	IncGorutinesAll()
	IncGorutinesCurrent()
	DecGorutinesCurrent()
	IncTimeouts(label string)
	IncCancels(label string)
}

var metricsCollector metrics

func incGorutines() {
	if metricsCollector == nil {
		return
	}

	metricsCollector.IncGorutinesAll()
	metricsCollector.IncGorutinesCurrent()
}

func decGorutines() {
	if metricsCollector == nil {
		return
	}

	metricsCollector.DecGorutinesCurrent()
}

func incTimeouts(label string) {
	if metricsCollector == nil {
		return
	}

	metricsCollector.IncTimeouts(label)
}

func incCancels(label string) {
	if metricsCollector == nil {
		return
	}

	metricsCollector.IncCancels(label)
}

type Metrics struct {
	timeouts         *prometheus.CounterVec
	cancels          *prometheus.CounterVec
	gorutinesAll     prometheus.Counter
	gorutinesCurrent prometheus.Gauge
}

func NewPrometheusMetrics(appName string) *Metrics {
	return &Metrics{
		timeouts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: appName,
			Subsystem: "context",
			Name:      "timeouts",
		}, []string{"context"}),
		cancels: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: appName,
			Subsystem: "context",
			Name:      "cancels",
		}, []string{"context"}),
		gorutinesAll: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: appName,
			Subsystem: "context",
			Name:      "gorutines",
		}),
		gorutinesCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: appName,
			Subsystem: "context",
			Name:      "gorutines_current",
		}),
	}
}

func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.timeouts,
		m.gorutinesAll,
		m.gorutinesCurrent,
	}
}

func InitMetrics(m metrics) {
	metricsCollector = m
}

func (m *Metrics) IncGorutinesAll() {
	m.gorutinesAll.Inc()
}
func (m *Metrics) IncGorutinesCurrent() {
	m.gorutinesCurrent.Inc()
}

func (m *Metrics) DecGorutinesCurrent() {
	m.gorutinesCurrent.Dec()
}

func (m *Metrics) IncTimeouts(label string) {
	m.timeouts.WithLabelValues(label).Inc()
}

func (m *Metrics) IncCancels(label string) {
	m.cancels.WithLabelValues(label).Inc()
}
