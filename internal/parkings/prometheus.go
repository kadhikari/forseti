package parkings

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ParkingsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "parkings",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	ParkingsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "parkings",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
