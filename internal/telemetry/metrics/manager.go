package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Manager struct {
	// counters
	CounterRequests            *prometheus.CounterVec
	CounterNetlogVisits        prometheus.Counter
	CounterNotes               prometheus.Counter
	CounterHandleRequestPanic  prometheus.Counter
	CounterVisitsBackups       prometheus.Counter
	CounterRateLimitedRequests prometheus.Counter

	// gauges
	GaugeRequests   prometheus.Gauge
	GaugeLifeSignal prometheus.Gauge

	// histograms
	HistNetlogBackupDuration prometheus.Histogram
	HistogramRequestDuration *prometheus.HistogramVec
}

func NewTestManager() *Manager {
	return NewManager("backend", "test_server", prometheus.NewRegistry())
}

func NewTestManagerAndRegistry() (*Manager, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	return NewManager("backend", "test_server", reg), reg
}

func NewManager(namespace, subsystem string, reg prometheus.Registerer) *Manager {
	factory := promauto.With(reg)

	counterRequests := factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request",
		Help:      "The total number of incoming requests",
	}, []string{"method", "status"})
	counterNetlogVisits := factory.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "netlog_visits",
		Help:      "The total number of netlog visits",
	})
	counterNotes := factory.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "notes",
		Help:      "The total number of added notes",
	})
	counterHandleRequestPanic := factory.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "handle_request_panic",
		Help:      "The total number of serve request panics",
	})
	counterVisitsBackups := factory.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "netlog_visits_backed_up",
		Help:      "Number of netlog visits backed up",
	})
	counterRateLimitedRequests := promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "rate_limited_requests",
		Help:      "The total number of rate limited requests",
	})

	gaugeRequests := factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "current_requests",
		Help:        "Current number of requests served",
		ConstLabels: nil,
	})
	gaugeLifeSignal := factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "life_signal",
		Help:        "Shows whether the service is alive",
		ConstLabels: nil,
	})

	histNetlogBackupDuration := factory.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Buckets: []float64{
				0.0001, 0.001, 0.01, 0.1, 1, 10,
				60, 120, 240, 480, 1000, 2000,
				4000, 10000,
			},
			Name: "netlog_backup_duration_seconds",
			Help: "Total duration of a single netlog backup in seconds",
		},
	)

	histogramRequestDuration := factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help:      "Histogram of response time for requests in seconds",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"route", "method", "status_code"})

	return &Manager{
		CounterRequests:            counterRequests,
		CounterNetlogVisits:        counterNetlogVisits,
		CounterNotes:               counterNotes,
		CounterHandleRequestPanic:  counterHandleRequestPanic,
		CounterVisitsBackups:       counterVisitsBackups,
		CounterRateLimitedRequests: counterRateLimitedRequests,
		GaugeRequests:              gaugeRequests,
		GaugeLifeSignal:            gaugeLifeSignal,
		HistNetlogBackupDuration:   histNetlogBackupDuration,
		HistogramRequestDuration:   histogramRequestDuration,
	}
}
