package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// http://prometheus.serj-tubin.com/
// http://grafana.serj-tubin.com/

type Instrumentation struct {
	CounterRequests           prometheus.Counter
	CounterNetlogVisits       prometheus.Counter
	CounterHandleRequestPanic prometheus.Counter
	GaugeRequests             prometheus.Gauge
	GaugeLifeSignal           prometheus.Gauge
	HistRequestDuration       prometheus.Histogram
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
	counterHandleRequestPanic := promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "handle_request_panic",
		Help:      "The total number of serve request panics",
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

	histReqDuration := promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Buckets: []float64{
				0.0000001, 0.0000002, 0.0000003, 0.0000004, 0.0000005,
				0.000001, 0.0000025, 0.000005, 0.0000075, 0.00001,
				0.0001, 0.001, 0.01, 0.1, 1, 10, 60,
			},
			Name: "request_duration_seconds",
			Help: "Total duration of all requests",
		},
	)

	return &Instrumentation{
		CounterRequests:           counterRequests,
		CounterNetlogVisits:       counterNetlogVisits,
		CounterHandleRequestPanic: counterHandleRequestPanic,
		GaugeRequests:             gaugeRequests,
		GaugeLifeSignal:           gaugeLifeSignal,
		HistRequestDuration:       histReqDuration,
	}
}
