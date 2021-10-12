package vehiclepositions

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	VehiclePositionsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_positions",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	VehiclePositionsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_positions",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
