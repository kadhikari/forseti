package equipments

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	EquipmentsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "equipments",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	EquipmentsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "equipments",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
