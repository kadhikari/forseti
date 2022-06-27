package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/hove-io/forseti"
	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/equipments"
	"github.com/hove-io/forseti/internal/freefloatings"
	"github.com/hove-io/forseti/internal/manager"
	"github.com/hove-io/forseti/internal/parkings"
	vehicleoccupancies "github.com/hove-io/forseti/internal/vehicleoccupancies"
	"github.com/hove-io/forseti/internal/vehiclepositions"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type LoadingStatus struct {
	Source        string    `json:"source"`
	RefreshActive bool      `json:"refresh_active"`
	RefreshTime   string    `json:"refresh_data"`
	LastUpdate    time.Time `json:"last_update"`
}

// StatusResponse defines the object returned by the /status endpoint
type StatusResponse struct {
	Status              string        `json:"status,omitempty"`
	Version             string        `json:"version,omitempty"`
	LastDepartureUpdate time.Time     `json:"last_departure_update"`
	LastParkingUpdate   time.Time     `json:"last_parking_update"`
	LastEquipmentUpdate time.Time     `json:"last_equipment_update"`
	FreeFloatings       LoadingStatus `json:"free_floatings,omitempty"`
	VehicleOccupancies  LoadingStatus `json:"vehicle_occupancies,omitempty"`
	VehiclePositions    LoadingStatus `json:"vehicle_positions,omitempty"`
}

var (
	httpDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "http",
		Name:      "durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	},
		[]string{"handler", "code"},
	)

	httpInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "forseti",
		Subsystem: "http",
		Name:      "in_flight",
		Help:      "current number of http request being served",
	},
	)
)

func StatusHandler(manager *manager.DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var lastFreeFloatingsDataUpdate time.Time
		var loadFreeFloatingData bool = false
		var refreshFreeFloatingData string
		var freeFloatingsSource string
		if manager.GetFreeFloatingsContext() != nil {
			// manage freefloating activation /status?free_floatings=true or false
			freeFloatingStatus := c.Query("free_floatings")
			if len(freeFloatingStatus) > 0 {
				toActive, _ := strconv.ParseBool(freeFloatingStatus)
				manager.GetFreeFloatingsContext().ManageFreeFloatingsStatus(toActive)
			}
			freeFloatingsSource = manager.GetFreeFloatingsContext().GetPackageName()
			lastFreeFloatingsDataUpdate = manager.GetFreeFloatingsContext().GetLastFreeFloatingsDataUpdate()
			loadFreeFloatingData = manager.GetFreeFloatingsContext().LoadFreeFloatingsData()
			refreshFreeFloatingData = manager.GetFreeFloatingsContext().GetRereshTime()
		}

		var lastVehicleOccupanciesDataUpdate time.Time
		var loadVehicleOccupanciesData bool = false
		var refreshVehicleOccupanciesData string
		if manager.GetVehicleOccupanciesContext() != nil {
			// manage vehicleoccupancy activation /status?vehicle_occupancies=true or false
			vehicleOccupancyStatus := c.Query("vehicle_occupancies")
			if len(vehicleOccupancyStatus) > 0 {
				toActive, _ := strconv.ParseBool(vehicleOccupancyStatus)
				manager.GetVehicleOccupanciesContext().ManageVehicleOccupancyStatus(toActive)
			}
			lastVehicleOccupanciesDataUpdate = manager.GetVehicleOccupanciesContext().GetLastVehicleOccupanciesDataUpdate()
			loadVehicleOccupanciesData = manager.GetVehicleOccupanciesContext().LoadOccupancyData()
			refreshVehicleOccupanciesData = manager.GetVehicleOccupanciesContext().GetRereshTime()
		}

		var lastVehiclePositionsDataUpdate time.Time
		var loadVehiclePositionsData bool = false
		var refreshVehiclePositionsData string
		if manager.GetVehiclePositionsContext() != nil {
			// manage vehiclepositions activation /status?vehicle_positions=true or false
			vehiclePositionsStatus := c.Query("vehicle_positions")
			if len(vehiclePositionsStatus) > 0 {
				toActive, _ := strconv.ParseBool(vehiclePositionsStatus)
				manager.GetVehiclePositionsContext().ManageVehiclePositionsStatus(toActive)
			}
			lastVehiclePositionsDataUpdate = manager.GetVehiclePositionsContext().GetLastVehiclePositionsDataUpdate()
			loadVehiclePositionsData = manager.GetVehiclePositionsContext().LoadPositionsData()
			refreshVehiclePositionsData = manager.GetVehiclePositionsContext().GetRereshTime()
		}

		var lastEquipmentDataUpdate time.Time
		if manager.GetEquipmentsContext() != nil {
			lastEquipmentDataUpdate = manager.GetEquipmentsContext().GetLastEquipmentsDataUpdate()
		}

		var lastDeparturesDataUpdate time.Time
		if manager.GetDeparturesContext() != nil {
			lastDeparturesDataUpdate = manager.GetDeparturesContext().GetLastDepartureDataUpdate()
		}

		var lastParkingsDataUpdate time.Time
		if manager.GetParkingsContext() != nil {
			lastParkingsDataUpdate = manager.GetParkingsContext().GetLastParkingsDataUpdate()
		}

		c.JSON(http.StatusOK, StatusResponse{
			"ok",
			forseti.ForsetiVersion,
			lastDeparturesDataUpdate,
			lastParkingsDataUpdate,
			lastEquipmentDataUpdate,
			LoadingStatus{freeFloatingsSource, loadFreeFloatingData, refreshFreeFloatingData, lastFreeFloatingsDataUpdate},
			LoadingStatus{"", loadVehicleOccupanciesData, refreshVehicleOccupanciesData, lastVehicleOccupanciesDataUpdate},
			LoadingStatus{"", loadVehiclePositionsData, refreshVehiclePositionsData, lastVehiclePositionsDataUpdate},
		})
	}
}

func SetupRouter(manager *manager.DataManager, r *gin.Engine) *gin.Engine {
	if r == nil {
		r = gin.New()
	}
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, false))
	r.Use(instrumentGin())
	r.Use(gin.Recovery())
	pprof.Register(r)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/status", StatusHandler(manager))

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
	prometheus.MustRegister(departures.DepartureLoadingDuration)
	prometheus.MustRegister(departures.DepartureLoadingErrors)
	prometheus.MustRegister(parkings.ParkingsLoadingDuration)
	prometheus.MustRegister(parkings.ParkingsLoadingErrors)
	prometheus.MustRegister(equipments.EquipmentsLoadingDuration)
	prometheus.MustRegister(equipments.EquipmentsLoadingErrors)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingDuration)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingErrors)
	prometheus.MustRegister(vehicleoccupancies.VehicleOccupanciesLoadingDuration)
	prometheus.MustRegister(vehicleoccupancies.VehicleOccupanciesLoadingErrors)
	prometheus.MustRegister(vehiclepositions.VehiclePositionsLoadingDuration)
	prometheus.MustRegister(vehiclepositions.VehiclePositionsLoadingErrors)
}
