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
	JSONLog           bool   `mapstructure:"json-log"`
	LogLevel          string `mapstructure:"log-level"`
}

func GetConfig() (Config, error) {
	pflag.String("departures-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path \nexample: sftp://sytral:pass@172.17.0.3:22/extract_edylic.txt")
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
	manager := &sytralrt.DataManager{}
	err = sytralrt.RefreshDepartures(manager, config.DeparturesURI)
	if err != nil {
		//TODO start without data
		logrus.Errorf("Impossible to load data at startup: %s", err)
	}

	r := sytralrt.SetupRouter(manager, nil)
	go RefreshLoop(manager, config.DeparturesURI, config.DeparturesRefresh)

	err = r.Run()
	if err != nil {
		logrus.Fatalf("Impossible to start gin: %s", err)
	}

}

func RefreshLoop(manager *sytralrt.DataManager, departuresURI url.URL, departuresRefresh time.Duration) {
	if departuresRefresh.Seconds() < 1 {
		logrus.Info("data refreshing is disabled")
		return
	}
	for {
		logrus.Debug("refreshing of departures data")
		err := sytralrt.RefreshDepartures(manager, departuresURI)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		}
		logrus.Debug("Data updated")
		time.Sleep(departuresRefresh)
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
