package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// http://prometheus.serj-tubin.com/
// http://grafana.serj-tubin.com/

type Instrumentation struct {
	CounterRequests     prometheus.Counter
	CounterNetlogVisits prometheus.Counter
	GaugeRequests       prometheus.Gauge
	GaugeLifeSignal     prometheus.Gauge
}

func NewInstrumentation(namespace, subsystem string) *Instrumentation {
	counterRequests := promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request",
		Help:      "The total number of incoming requests",
	})
	counterNetlogVisits := promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "netlog_visits",
		Help:      "The total number of netlog visits",
	})

	gaugeRequests := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "current_requests",
		Help:        "Current number of requests served",
		ConstLabels: nil,
	})
	gaugeLifeSignal := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "life_signal",
		Help:        "Shows whether the service is alive",
		ConstLabels: nil,
	})

	return &Instrumentation{
		CounterRequests:     counterRequests,
		CounterNetlogVisits: counterNetlogVisits,
		GaugeRequests:       gaugeRequests,
		GaugeLifeSignal:     gaugeLifeSignal,
	}
}
