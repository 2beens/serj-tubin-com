package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func SetupPrometheus() *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()

	// Add Go module build info, runtime metrics and process collectors.
	promRegistry.MustRegister(
		collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return promRegistry
}
