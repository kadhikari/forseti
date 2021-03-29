package freefloatings

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	FreeFloatingsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "free_floatings",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	FreeFloatingsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "free_floatings",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
