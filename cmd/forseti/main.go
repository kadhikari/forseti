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

	"github.com/CanalTP/forseti"
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

	FreeFloatingsURIStr  string        `mapstructure:"free-floatings-uri"`
	FreeFloatingsRefresh time.Duration `mapstructure:"free-floatings-refresh"`
	FreeFloatingsURI     url.URL
	FreeFloatingsToken  string        `mapstructure:"free-floatings-token"`

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
	pflag.String("free-floatings-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("free-floatings-token", "", "token for free floating source")
	pflag.Duration("free-floatings-refresh", 30*time.Second, "time between refresh of vehicles in Fluctéo data")
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

	if noneOf(config.DeparturesURIStr, config.ParkingsURIStr, config.EquipmentsURIStr, config.FreeFloatingsURIStr) {
		return config, errors.New("no data provided at all. Please provide at lease one type of data")
	}

	for configURIStr, configURI := range map[string]*url.URL{
		config.DeparturesURIStr: &config.DeparturesURI,
		config.ParkingsURIStr:   &config.ParkingsURI,
		config.EquipmentsURIStr: &config.EquipmentsURI,
		config.FreeFloatingsURIStr: &config.FreeFloatingsURI,
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

	err = forseti.RefreshFreeFloatings(manager, config.FreeFloatingsURI, config.FreeFloatingsToken, config.ConnectionTimeout)
	if err != nil {
		logrus.Errorf("Impossible to load free_floatings data at startup: %s (%s)", err, config.FreeFloatingsURIStr)
	}

	go RefreshDepartureLoop(manager, config.DeparturesURI, config.DeparturesRefresh, config.ConnectionTimeout)
	go RefreshParkingLoop(manager, config.ParkingsURI, config.ParkingsRefresh, config.ConnectionTimeout)
	go RefreshEquipmentLoop(manager, config.EquipmentsURI, config.EquipmentsRefresh, config.ConnectionTimeout)
	go RefreshFreeFloatingLoop(manager, config.FreeFloatingsURI, config.FreeFloatingsToken, config.FreeFloatingsRefresh, config.ConnectionTimeout)

	err = forseti.SetupRouter(manager, nil).Run()
	if err != nil {
		logrus.Fatalf("Impossible to start gin: %s", err)
	}
}

func RefreshDepartureLoop(manager *forseti.DataManager,
	departuresURI url.URL,
	departuresRefresh, connectionTimeout time.Duration) {
	if (len(departuresURI.String()) == 0 || departuresRefresh.Seconds() <= 0) {
		logrus.Debug("Departure data refreshing is disabled")
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
	if (len(parkingsURI.String()) == 0 || parkingsRefresh.Seconds() <= 0){
		logrus.Debug("Parking data refreshing is disabled")
		return
	}
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
	if (len(equipmentsURI.String()) == 0 || equipmentsRefresh.Seconds() <= 0){
		logrus.Debug("Equipment data refreshing is disabled")
		return
	}
	for {
		err := forseti.RefreshEquipments(manager, equipmentsURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading equipment data: ", err)
		}
		logrus.Debug("Equipment data updated")
		time.Sleep(equipmentsRefresh)
	}
}

func RefreshFreeFloatingLoop(manager *forseti.DataManager,
	freeFloatingsURI url.URL,
	freeFloatingsToken string,
	freeFloatingsRefresh,
	connectionTimeout time.Duration) {
	if (len(freeFloatingsURI.String()) == 0 || freeFloatingsRefresh.Seconds() <= 0){
		logrus.Debug("FreeFloating data refreshing is disabled")
	}
	for {
		err := forseti.RefreshFreeFloatings(manager, freeFloatingsURI, freeFloatingsToken, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading freefloating data: ", err)
		}
		logrus.Debug("Free_floating data updated")
		time.Sleep(freeFloatingsRefresh)
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
