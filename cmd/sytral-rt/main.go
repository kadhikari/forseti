package main

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/CanalTP/sytralrt"
)

type Config struct {
	DeparturesURIStr  string        `mapstructure:"departures-uri"`
	DeparturesRefresh time.Duration `mapstructure:"departures-refresh"`
	DeparturesURI     *url.URL

	ParkingsURIStr  string        `mapstructure:"parkings-uri"`
	ParkingsRefresh time.Duration `mapstructure:"parkings-refresh"`
	ParkingsURI     *url.URL

	JSONLog  bool   `mapstructure:"json-log"`
	LogLevel string `mapstructure:"log-level"`
}

func GetConfig() (Config, error) {
	pflag.String("departures-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path \nexample: sftp://sytral:pass@172.17.0.3:22/extract_edylic.txt")
	pflag.Duration("departures-refresh", 30*time.Second, "time between refresh of departures data")
	pflag.String("parkings-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path")
	pflag.Duration("parkings-refresh", 30*time.Second, "time between refresh of parkings data")
	pflag.Bool("json-log", false, "enable json logging")
	pflag.String("log-level", "debug", "log level: debug, info, warn, error")
	pflag.Parse()

	var config Config
	var err error
	if err = viper.BindPFlags(pflag.CommandLine); err != nil {
		return config, errors.Wrap(err, "Impossible to parse flags")
	}
	viper.SetEnvPrefix("SYTRALRT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err = viper.Unmarshal(&config); err != nil {
		return config, errors.Wrap(err, "Unmarshalling of flag failed")
	}

	parseURL := func(urlStr string) (*url.URL, error) {
		if urlStr == "" {
			return nil, nil
		}

		if url, err := url.Parse(urlStr); err != nil {
			return nil, errors.Wrapf(err, "Impossible to parse URL: %s", urlStr)
		} else {
			return url, nil
		}
	}

	if config.DeparturesURI, err = parseURL(config.DeparturesURIStr); err != nil {
		err = errors.Wrap(err, "Unable to parse departures data url")
	}

	if config.ParkingsURI, err = parseURL(config.ParkingsURIStr); err != nil {
		err = errors.Wrap(err, "Unable to parse parkings data url")
	}

	return config, err
}

func main() {
	config, err := GetConfig()
	if err != nil {
		logrus.Fatalf("Impossible to load data at startup: %s", err)
	}

	initLog(config.JSONLog, config.LogLevel)
	manager := &sytralrt.DataManager{}

	if config.DeparturesURI != nil {
		if err = sytralrt.RefreshDepartures(manager, *config.DeparturesURI); err != nil {
			logrus.Errorf("Impossible to load departures data at startup: %s", err)
		} else {
			go RefreshDepartureLoop(manager, *config.DeparturesURI, config.DeparturesRefresh)
		}
	}

	if config.ParkingsURI != nil {
		if err = sytralrt.RefreshParkings(manager, *config.ParkingsURI); err != nil {
			logrus.Errorf("Impossible to load parkings data at startup: %s", err)
		} else {
			go RefreshParkingLoop(manager, *config.ParkingsURI, config.ParkingsRefresh)
		}
	}

	err = sytralrt.SetupRouter(manager, nil).Run()
	if err != nil {
		logrus.Fatalf("Impossible to start gin: %s", err)
	}

}

func RefreshDepartureLoop(manager *sytralrt.DataManager, departuresURI url.URL, departuresRefresh time.Duration) {
	if departuresRefresh.Seconds() < 1 {
		logrus.Info("data refreshing is disabled")
		return
	}
	for {
		err := sytralrt.RefreshDepartures(manager, departuresURI)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		}
		logrus.Debug("Departure data updated")
		time.Sleep(departuresRefresh)
	}
}

func RefreshParkingLoop(manager *sytralrt.DataManager, parkingsURI url.URL, parkingsRefresh time.Duration) {
	for {
		err := sytralrt.RefreshParkings(manager, parkingsURI)
		if err != nil {
			logrus.Error("Error while reloading parking data: ", err)
		}
		logrus.Debug("Parking data updated")
		time.Sleep(parkingsRefresh)
	}
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
