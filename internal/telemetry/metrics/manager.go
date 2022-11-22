package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Manager struct {
	// counters
	CounterRequests           *prometheus.CounterVec
	CounterNetlogVisits       prometheus.Counter
	CounterNotes              prometheus.Counter
	CounterHandleRequestPanic prometheus.Counter
	CounterVisitsBackups      prometheus.Counter

	// gauges
	GaugeRequests   prometheus.Gauge
	GaugeLifeSignal prometheus.Gauge

	// historgrams
	HistRequestDuration      prometheus.Histogram
	HistNetlogBackupDuration prometheus.Histogram
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

	histReqDuration := factory.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Buckets: []float64{
				0.0000001, 0.0000002, 0.0000003, 0.0000004, 0.0000005,
				0.000001, 0.0000025, 0.000005, 0.0000075, 0.00001,
				0.0001, 0.001, 0.01, 0.1, 1, 10, 60,
			},
			Name: "request_duration_seconds",
			Help: "Total duration of requests in seconds",
		},
	)
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

	return &Manager{
		CounterRequests:           counterRequests,
		CounterNetlogVisits:       counterNetlogVisits,
		CounterNotes:              counterNotes,
		CounterHandleRequestPanic: counterHandleRequestPanic,
		CounterVisitsBackups:      counterVisitsBackups,
		GaugeRequests:             gaugeRequests,
		GaugeLifeSignal:           gaugeLifeSignal,
		HistRequestDuration:       histReqDuration,
		HistNetlogBackupDuration:  histNetlogBackupDuration,
	}
}
