package departures

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DepartureLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "departures",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	DepartureLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "departures",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
