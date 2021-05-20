package main

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/CanalTP/forseti/api"
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/manager"
	"github.com/CanalTP/forseti/internal/parkings"
	"github.com/CanalTP/forseti/internal/vehicleoccupancies"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	FreeFloatingsToken   string `mapstructure:"free-floatings-token"`

	OccupancyFilesURIStr   string `mapstructure:"occupancy-files-uri"`
	OccupancyFilesURI      url.URL
	OccupancyNavitiaURIStr string `mapstructure:"occupancy-navitia-uri"`
	OccupancyNavitiaURI    url.URL
	OccupancyServiceURIStr string `mapstructure:"occupancy-service-uri"`
	OccupancyServiceURI    url.URL
	OccupancyNavitiaToken  string        `mapstructure:"occupancy-navitia-token"`
	OccupancyServiceToken  string        `mapstructure:"occupancy-service-token"`
	OccupancyRefresh       time.Duration `mapstructure:"occupancy-refresh"`
	RouteScheduleRefresh   time.Duration `mapstructure:"routeschedule-refresh"`
	TimeZoneLocation       string        `mapstructure:"timezone-location"`

	LogLevel            string        `mapstructure:"log-level"`
	ConnectionTimeout   time.Duration `mapstructure:"connection-timeout"`
	JSONLog             bool          `mapstructure:"json-log"`
	FreeFloatingsActive bool          `mapstructure:"free-floatings-refresh-active"`
	OccupancyActive     bool          `mapstructure:"occupancy-service-refresh-active"`
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
	pflag.Bool("free-floatings-refresh-active", false, "activate the periodic refresh of Fluctuo data")
	pflag.Duration("free-floatings-refresh", 30*time.Second, "time between refresh of vehicles in Fluctéo data")

	//Passing configurations for vehicle_occupancies
	pflag.String("occupancy-files-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-navitia-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-navitia-token", "", "token for navitia")
	pflag.String("occupancy-service-uri", "", "format: [scheme:][//[userinfo@]host][/]path")
	pflag.String("occupancy-service-token", "", "token for prediction source")
	pflag.Bool("occupancy-service-refresh-active", false, "activate the periodic refresh of vehicle occupancy data")
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
		config.DeparturesURIStr:       &config.DeparturesURI,
		config.ParkingsURIStr:         &config.ParkingsURI,
		config.EquipmentsURIStr:       &config.EquipmentsURI,
		config.FreeFloatingsURIStr:    &config.FreeFloatingsURI,
		config.OccupancyFilesURIStr:   &config.OccupancyFilesURI,
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
	manager := &manager.DataManager{}

	location, _ := time.LoadLocation(config.TimeZoneLocation)

	// create API router
	router := api.SetupRouter(manager, nil)

	// With equipments
	Equipments(manager, &config, router)

	// With freefloating
	FreeFloating(manager, &config, router)

	// With departures
	Departures(manager, &config, router)

	// With parkings
	Parkings(manager, &config, router)

	// With vehicle occupancies
	VehiculeOccupancies(manager, &config, router, location)

	// start router
	err = router.Run()
	if err != nil {
		logrus.Fatalf("Impossible to start gin: %s", err)
	}
}

func Equipments(manager *manager.DataManager, config *Config, router *gin.Engine) {
	if len(config.EquipmentsURI.String()) == 0 || config.EquipmentsRefresh.Seconds() <= 0 {
		logrus.Debug("Equipments is disabled")
		return
	}
	equipmentsContext := &equipments.EquipmentsContext{}
	manager.SetEquipmentsContext(equipmentsContext)
	go equipments.RefreshEquipmentLoop(equipmentsContext, config.EquipmentsURI,
		config.EquipmentsRefresh, config.ConnectionTimeout)
	equipments.AddEquipmentsEntryPoint(router, equipmentsContext)
}

func FreeFloating(manager *manager.DataManager, config *Config, router *gin.Engine) {
	if len(config.FreeFloatingsURI.String()) == 0 || config.FreeFloatingsRefresh.Seconds() <= 0 {
		logrus.Debug("FreeFloating is disabled")
		return
	}

	freeFloatingsContext := &freefloatings.FreeFloatingsContext{}
	manager.SetFreeFloatingsContext(freeFloatingsContext)

	// Manage activation of the periodic refresh of Fluctuo data
	freefloatings.ManagefreeFloatingActivation(freeFloatingsContext, config.FreeFloatingsActive)

	go freefloatings.RefreshFreeFloatingLoop(freeFloatingsContext,
		config.FreeFloatingsURI,
		config.FreeFloatingsToken,
		config.EquipmentsRefresh,
		config.ConnectionTimeout)
	freefloatings.AddFreeFloatingsEntryPoint(router, freeFloatingsContext)
}

func Departures(manager *manager.DataManager, config *Config, router *gin.Engine) {
	if len(config.DeparturesURI.String()) == 0 || config.DeparturesRefresh.Seconds() <= 0 {
		logrus.Debug("Departures is disabled")
		return
	}
	departuresContext := &departures.DeparturesContext{}
	manager.SetDeparturesContext(departuresContext)
	go departures.RefreshDeparturesLoop(departuresContext, config.DeparturesURI,
		config.DeparturesRefresh, config.ConnectionTimeout)
	departures.AddDeparturesEntryPoint(router, departuresContext)
}

func Parkings(manager *manager.DataManager, config *Config, router *gin.Engine) {
	if len(config.ParkingsURI.String()) == 0 || config.ParkingsRefresh.Seconds() <= 0 {
		logrus.Debug("Parkings is disabled")
		return
	}
	parkingsContext := &parkings.ParkingsContext{}
	manager.SetParkingsContext(parkingsContext)
	go parkings.RefreshParkingsLoop(parkingsContext, config.ParkingsURI,
		config.ParkingsRefresh, config.ConnectionTimeout)
	parkings.AddParkingsEntryPoint(router, parkingsContext)
}

func VehiculeOccupancies(manager *manager.DataManager, config *Config, router *gin.Engine, location *time.Location) {
	if len(config.OccupancyNavitiaURI.String()) == 0 || len(config.OccupancyServiceURI.String()) == 0 {
		logrus.Debug("Vehicle occupancies is disabled")
		return
	}

	var vehiculeOccupanciesContext, err = vehicleoccupancies.VehicleOccupancyFactory("gtfs")
	if err != nil {
		logrus.Error(err)
		return
	}

	manager.SetVehiculeOccupanciesContext(vehiculeOccupanciesContext)

	vehiculeOccupanciesContext.InitContext(config.OccupancyFilesURI, config.OccupancyServiceURI,
		config.OccupancyServiceToken, config.OccupancyNavitiaURI, config.OccupancyNavitiaToken,
		config.OccupancyRefresh, config.ConnectionTimeout, location, config.OccupancyActive)

	go vehiculeOccupanciesContext.RefreshVehicleOccupanciesLoop(config.OccupancyServiceURI,
		config.OccupancyServiceToken, config.OccupancyNavitiaURI, config.OccupancyNavitiaToken,
		config.OccupancyRefresh, config.ConnectionTimeout, location)
	vehicleoccupancies.AddVehicleOccupanciesEntryPoint(router, vehiculeOccupanciesContext)

	if vehicleOccupanciesOditiContext, ok :=
		vehiculeOccupanciesContext.(*vehicleoccupancies.VehicleOccupanciesOditiContext); ok {
		go vehicleOccupanciesOditiContext.RefreshDataFromNavitia(config.OccupancyNavitiaURI,
			config.OccupancyNavitiaToken, config.RouteScheduleRefresh, config.ConnectionTimeout, location)
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
