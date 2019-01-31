package main

import (
	"fmt"
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
	DeparturesURI     url.URL

	ParkingsURIStr  string        `mapstructure:"parkings-uri"`
	ParkingsRefresh time.Duration `mapstructure:"parkings-refresh"`
	ParkingsURI     url.URL

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
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return config, errors.Wrap(err, "Impossible to parse flags")
	}
	viper.SetEnvPrefix("SYTRALRT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.Unmarshal(&config); err != nil {
		return config, errors.Wrap(err, "Unmarshalling of flag failed")
	}

	if config.DeparturesURIStr == "" {
		return config, fmt.Errorf("departures-uri is required")
	}

	if config.ParkingsURIStr == "" {
		return config, fmt.Errorf("parkings-uri is required")
	}

	departuresUri, err := url.Parse(config.DeparturesURIStr)
	if err != nil {
		return config, errors.Wrap(err, "Impossible to parse URL")
	}

	parkingsUri, err := url.Parse(config.ParkingsURIStr)
	if err != nil {
		return config, errors.Wrap(err, "Impossible to parse URL")
	}

	config.DeparturesURI = *departuresUri
	config.ParkingsURI = *parkingsUri
	return config, nil
}

func main() {
	config, err := GetConfig()
	if err != nil {
		logrus.Fatalf("Impossible to load data at startup: %s", err)
	}

	initLog(config.JSONLog, config.LogLevel)
	manager := &sytralrt.DataManager{}

	if err = sytralrt.RefreshDepartures(manager, config.DeparturesURI); err != nil {
		//TODO start without data
		logrus.Errorf("Impossible to load departures data at startup: %s", err)
	}

	if err = sytralrt.RefreshParkings(manager, config.ParkingsURI); err != nil {
		//TODO start without data
		logrus.Errorf("Impossible to load parkings data at startup: %s", err)
	}

	r := sytralrt.SetupRouter(manager, nil)

	go RefreshDepartureLoop(manager, config.DeparturesURI, config.DeparturesRefresh)
	go RefreshParkingLoop(manager, config.ParkingsURI, config.ParkingsRefresh)

	err = r.Run()
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
