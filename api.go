package sytralrt

import (
	"strconv"
	"time"

	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type DeparturesResponse struct {
	Message    string       `json:"message,omitempty"`
	Departures *[]Departure `json:"departures,omitempty"` // the pointer allow us to display an empty array in json
}

var (
	httpDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "sytralrt",
		Subsystem: "http",
		Name:      "durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	},
		[]string{"handler", "code"},
	)

	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "sytralrt",
		Subsystem: "http",
		Name:      "in_flight",
		Help:      "current number of http request being served",
	},
	)
)

func DeparturesHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := DeparturesResponse{}
		stopID := c.Query("stop_id")
		if stopID == "" {
			response.Message = "stopID is required"
			c.JSON(400, response)
			return
		}
		departures, err := manager.GetDeparturesByStop(stopID)
		if err != nil {
			response.Message = "No data loaded"
			c.JSON(503, response)
			return
		}
		response.Departures = &departures
		c.JSON(200, response)
	}
}

func SetupRouter(manager *DataManager, r *gin.Engine) *gin.Engine {
	if r == nil {
		r = gin.New()
	}
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, false))
	r.Use(instrumentGin())
	r.Use(gin.Recovery())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/departures", DeparturesHandler(manager))

	return r
}

func instrumentGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		begin := time.Now()
		httpInFlight.Inc()
		c.Next()
		httpInFlight.Dec()
		observer := httpDurations.With(prometheus.Labels{"handler": c.HandlerName(), "code": strconv.Itoa(c.Writer.Status())})
		observer.Observe(time.Since(begin).Seconds())
	}
}

func init() {
	prometheus.MustRegister(httpDurations)
	prometheus.MustRegister(httpInFlight)
}
