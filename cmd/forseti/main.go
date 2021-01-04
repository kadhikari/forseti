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

	"github.com/kadhikari/forseti"
)

type Config struct {
	DeparturesURIStr  string        `mapstructure:"departures-uri"`
	DeparturesRefresh time.Duration `mapstructure:"departures-refresh"`
	DeparturesURI     url.URL

	ParkingsURIStr  string        `mapstructure:"parkings-uri"`
	ParkingsRefresh time.Duration `mapstructure:"parkings-refresh"`
	ParkingsURI     url.URL

	EquipmentsURIStr  string        `mapstructure:"equipments-uri"`
	EquipmentsRefresh time.Duration `mapstructure:"equipments-refresh"`
	EquipmentsURI     url.URL

	ConnectionTimeout time.Duration `mapstructure:"connection-timeout"`
	JSONLog           bool          `mapstructure:"json-log"`
	LogLevel          string        `mapstructure:"log-level"`
}

func noneOf(args ...string) bool {
	for _, a := range args {
		if a != "" {
			return false
		}
	}
	return true
}

func GetConfig() (Config, error) {
	pflag.String("departures-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path \nexample: sftp://forseti:pass@172.17.0.3:22/extract_edylic.txt")
	pflag.Duration("departures-refresh", 30*time.Second, "time between refresh of departures data")
	pflag.String("parkings-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path")
	pflag.Duration("parkings-refresh", 30*time.Second, "time between refresh of parkings data")
	pflag.String("equipments-uri", "",
		"format: [scheme:][//[userinfo@]host][/]path")
	pflag.Duration("equipments-refresh", 30*time.Second, "time between refresh of equipments data")
	pflag.Duration("connection-timeout", 10*time.Second, "timeout to establish the ssh connection")
	pflag.Bool("json-log", false, "enable json logging")
	pflag.String("log-level", "debug", "log level: debug, info, warn, error")
	pflag.Parse()

	var config Config
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return config, errors.Wrap(err, "Impossible to parse flags")
	}
	viper.SetEnvPrefix("FORSETI")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.Unmarshal(&config); err != nil {
		return config, errors.Wrap(err, "Unmarshalling of flag failed")
	}

	if noneOf(config.DeparturesURIStr, config.ParkingsURIStr, config.EquipmentsURIStr) {
		return config, errors.New("no data provided at all. Please provide at lease one type of data")
	}

	for configURIStr, configURI := range map[string]*url.URL{
		config.DeparturesURIStr: &config.DeparturesURI,
		config.ParkingsURIStr:   &config.ParkingsURI,
		config.EquipmentsURIStr: &config.EquipmentsURI,
	} {
		if url, err := url.Parse(configURIStr); err != nil {
			logrus.Errorf("Unable to parse data url: %s", configURIStr)
		} else {
			*configURI = *url
		}
	}

	return config, nil
}

func main() {
	config, err := GetConfig()
	if err != nil {
		logrus.Fatalf("Impossible to load data at startup: %s", err)
	}

	initLog(config.JSONLog, config.LogLevel)
	manager := &forseti.DataManager{}

	err = forseti.RefreshDepartures(manager, config.DeparturesURI, config.ConnectionTimeout)
	if err != nil {
		logrus.Errorf("Impossible to load departures data at startup: %s (%s)", err, config.DeparturesURIStr)
	}

	err = forseti.RefreshParkings(manager, config.ParkingsURI, config.ConnectionTimeout)
	if err != nil {
		logrus.Errorf("Impossible to load parkings data at startup: %s (%s)", err, config.ParkingsURIStr)
	}

	err = forseti.RefreshEquipments(manager, config.EquipmentsURI, config.ConnectionTimeout)
	if err != nil {
		logrus.Errorf("Impossible to load equipments data at startup: %s (%s)", err, config.EquipmentsURIStr)
	}

	go RefreshDepartureLoop(manager, config.DeparturesURI, config.DeparturesRefresh, config.ConnectionTimeout)
	go RefreshParkingLoop(manager, config.ParkingsURI, config.ParkingsRefresh, config.ConnectionTimeout)
	go RefreshEquipmentLoop(manager, config.EquipmentsURI, config.EquipmentsRefresh, config.ConnectionTimeout)

	err = forseti.SetupRouter(manager, nil).Run()
	if err != nil {
		logrus.Fatalf("Impossible to start gin: %s", err)
	}
}

func RefreshDepartureLoop(manager *forseti.DataManager,
	departuresURI url.URL,
	departuresRefresh, connectionTimeout time.Duration) {
	if departuresRefresh.Seconds() < 1 {
		logrus.Info("data refreshing is disabled")
		return
	}
	for {
		err := forseti.RefreshDepartures(manager, departuresURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		}
		logrus.Debug("Departure data updated")
		time.Sleep(departuresRefresh)
	}
}

func RefreshParkingLoop(manager *forseti.DataManager,
	parkingsURI url.URL,
	parkingsRefresh, connectionTimeout time.Duration) {
	for {
		err := forseti.RefreshParkings(manager, parkingsURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading parking data: ", err)
		}
		logrus.Debug("Parking data updated")
		time.Sleep(parkingsRefresh)
	}
}

func RefreshEquipmentLoop(manager *forseti.DataManager,
	equipmentsURI url.URL,
	equipmentsRefresh, connectionTimeout time.Duration) {
	for {
		err := forseti.RefreshEquipments(manager, equipmentsURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading equipment data: ", err)
		}
		logrus.Debug("Equipment data updated")
		time.Sleep(equipmentsRefresh)
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
