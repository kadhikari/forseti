package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

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

func init() {
	prometheus.MustRegister(httpDurations)
	prometheus.MustRegister(httpInFlight)
}

type Config struct {
	DeparturesURIStr  string        `mapstructure:"departures-uri"`
	DeparturesRefresh time.Duration `mapstructure:"departures-refresh"`
	DeparturesURI     url.URL
	JSONLog           bool   `mapstructure:"json-log"`
	LogLevel          string `mapstructure:"log-level"`
}

func InstrumentGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		begin := time.Now()
		httpInFlight.Inc()
		c.Next()
		httpInFlight.Dec()
		observer := httpDurations.With(prometheus.Labels{"handler": c.HandlerName(), "code": strconv.Itoa(c.Writer.Status())})
		observer.Observe(time.Since(begin).Seconds())
	}
}

func GetConfig() (Config, error) {
	pflag.String("departures-uri", "", "format: [scheme:][//[userinfo@]host][/]path \nexample: sftp://sytral:pass@172.17.0.3:22/extract_edylic.txt")
	pflag.Duration("departures-refresh", 30*time.Second, "time between refresh of departures data")
	pflag.Bool("json-log", false, "enable json logging")
	pflag.String("log-level", "debug", "log level: debug, info, warn, error")
	var config Config
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return config, errors.Wrap(err, "Impossible to parse flags")
	}
	viper.SetEnvPrefix("SYTRALRT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	err = viper.Unmarshal(&config)
	if err != nil {
		return config, errors.Wrap(err, "Unmarshalling of flag failed")
	}
	if config.DeparturesURIStr == "" {
		return config, fmt.Errorf("departures-uri is required")
	}
	uri, err := url.Parse(config.DeparturesURIStr)
	if err != nil {
		return config, errors.Wrap(err, "Impossible to parse URL")
	}
	config.DeparturesURI = *uri
	return config, nil
}

func main() {
	config, err := GetConfig()
	if err != nil {
		logrus.Fatalf("Impossible to load data at startup: %s", err)
	}
	initLog(config.JSONLog, config.LogLevel)
	manager := &DataManager{}
	err = RefreshDepartures(manager, config.DeparturesURI)
	if err != nil {
		//TODO start without data
		logrus.Errorf("Impossible to load data at startup: %s", err)
	}

	r := setupRouter()
	r.GET("/departures", DeparturesHandler(manager))

	go RefreshLoop(manager, config.DeparturesURI, config.DeparturesRefresh)
	r.Run()

}

func RefreshLoop(manager *DataManager, departuresURI url.URL, departuresRefresh time.Duration) {
	if departuresRefresh.Seconds() < 1 {
		logrus.Info("data refreshing is disabled")
		return
	}
	for {
		logrus.Debug("refreshing of departures data")
		err := RefreshDepartures(manager, departuresURI)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		}
		logrus.Debug("Data updated")
		time.Sleep(departuresRefresh)
	}
}

func DeparturesHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		stopID := c.Query("stop_id")
		departures, err := manager.GetDeparturesByStop(stopID)
		if err != nil {
			c.JSON(503, departures) //TODO: return a pretty error response
			return
		}
		c.JSON(200, departures)
	}
}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, false))
	r.Use(InstrumentGin())
	r.Use(gin.Recovery())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}

func initLog(jsonLog bool, logLevel string) {
	if jsonLog {
		// Log as JSON instead of the default ASCII formatter.
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(level)
}
