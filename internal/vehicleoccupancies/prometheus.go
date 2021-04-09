package vehicleoccupancies

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	VehicleOccupanciesLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_occupancies",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	VehicleOccupanciesLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_occupancies",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)
