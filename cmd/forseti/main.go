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

	OccupancyFilesURIStr  	string        `mapstructure:"occupancy-files-uri"`
	OccupancyFilesURI		url.URL
	OccupancyNavitiaURIStr  string        `mapstructure:"occupancy-navitia-uri"`
	OccupancyNavitiaURI		url.URL
	OccupancyServiceURIStr  string        `mapstructure:"occupancy-service-uri"`
	OccupancyServiceURI		url.URL
	OccupancyNavitiaToken  	string        `mapstructure:"occupancy-navitia-token"`
	OccupancyServiceToken  	string        `mapstructure:"occupancy-service-token"`
	OccupancyRefresh 		time.Duration `mapstructure:"occupancy-refresh"`
	RouteScheduleRefresh 	time.Duration `mapstructure:"routeschedule-refresh"`
	TimeZoneLocation  		string        `mapstructure:"timezone-location"`

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

	//Passing configurations for vehicle_occupancies
	pflag.String("occupancy-files-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-navitia-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-navitia-token", "", "token for navitia")
	pflag.String("occupancy-service-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-service-token", "", "token for prediction source")
	pflag.Duration("occupancy-refresh", 5*time.Minute, "time between refresh of predictions")
	pflag.Duration("routeschedule-refresh", 24*time.Hour, "time between refresh of RouteSchedules from navitia")
	pflag.String("timezone-location", "Europe/Paris", "timezone location")

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

	if noneOf(config.DeparturesURIStr, config.ParkingsURIStr, config.EquipmentsURIStr, config.FreeFloatingsURIStr, 
		config.OccupancyFilesURIStr, config.OccupancyNavitiaURIStr, config.OccupancyServiceURIStr) {
		return config, errors.New("no data provided at all. Please provide at lease one type of data")
	}

	for configURIStr, configURI := range map[string]*url.URL{
		config.DeparturesURIStr: &config.DeparturesURI,
		config.ParkingsURIStr:   &config.ParkingsURI,
		config.EquipmentsURIStr: &config.EquipmentsURI,
		config.FreeFloatingsURIStr: &config.FreeFloatingsURI,
		config.OccupancyFilesURIStr: &config.OccupancyFilesURI,
		config.OccupancyNavitiaURIStr: &config.OccupancyNavitiaURI,
		config.OccupancyServiceURIStr: &config.OccupancyServiceURI,
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

	location, _ := time.LoadLocation(config.TimeZoneLocation)

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

	err = forseti.LoadAllForVehicleOccupancies(manager, config.OccupancyFilesURI, config.OccupancyNavitiaURI, config.OccupancyServiceURI,  
		config.OccupancyNavitiaToken, config.OccupancyServiceToken, config.ConnectionTimeout, location)
	if err != nil {
		logrus.Errorf("Impossible to load StopPoints data at startup: %s (%s)", err, config.OccupancyFilesURIStr)
	}

	go RefreshDepartureLoop(manager, config.DeparturesURI, config.DeparturesRefresh, config.ConnectionTimeout)
	go RefreshParkingLoop(manager, config.ParkingsURI, config.ParkingsRefresh, config.ConnectionTimeout)
	go RefreshEquipmentLoop(manager, config.EquipmentsURI, config.EquipmentsRefresh, config.ConnectionTimeout)
	go RefreshFreeFloatingLoop(manager, config.FreeFloatingsURI, config.FreeFloatingsToken, config.FreeFloatingsRefresh, config.ConnectionTimeout)
	go RefreshVehicleOccupanciesLoop(manager, config.OccupancyServiceURI, config.OccupancyServiceToken, 
		config.OccupancyRefresh, config.ConnectionTimeout, location)
	go RefreshRouteSchedulesLoop(manager, config.OccupancyNavitiaURI, config.OccupancyNavitiaToken,
		config.RouteScheduleRefresh, config.ConnectionTimeout, location)

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
		return
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

func RefreshVehicleOccupanciesLoop(manager *forseti.DataManager,
	predictionURI url.URL,
	predictionToken string,
	predictionRefresh,
	connectionTimeout time.Duration,
	location *time.Location) {
	if (len(predictionURI.String()) == 0 || predictionRefresh.Seconds() <= 0){
		logrus.Debug("VehicleOccupancy data refreshing is disabled")
		return
	}
	for {
		err := forseti.RefreshVehicleOccupancies(manager, predictionURI, predictionToken, connectionTimeout, location)
		if err != nil {
			logrus.Error("Error while reloading VehicleOccupancy data: ", err)
		}
		logrus.Debug("vehicle_occupancies data updated")
		time.Sleep(predictionRefresh)
	}
}

func RefreshRouteSchedulesLoop(manager *forseti.DataManager,
	navitiaURI url.URL,
	navitiaToken string,
	routeScheduleRefresh,
	connectionTimeout time.Duration,
	location *time.Location) {
	if (len(navitiaURI.String()) == 0 || routeScheduleRefresh.Seconds() <= 0){
		logrus.Debug("RouteSchedule data refreshing is disabled")
		return
	}
	for {
		err := forseti.LoadRoutesForAllLines(manager, navitiaURI, navitiaToken, connectionTimeout, location)
		if err != nil {
			logrus.Error("Error while reloading RouteSchedule data: ", err)
		}
		logrus.Debug("RouteSchedule data updated")
		time.Sleep(routeScheduleRefresh)
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
