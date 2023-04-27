package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func SetupPrometheus(additionalCollectors ...prometheus.Collector) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()

	// Add Go module build info, runtime metrics and process collectors.
	allCollectors := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),
	}

	promRegistry.MustRegister(append(allCollectors, additionalCollectors...)...)

	return promRegistry
}
