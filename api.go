package forseti

import (
	"net/http"
	"strconv"
	"time"
	"strings"
	"fmt"
	"github.com/gin-contrib/pprof"
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

// StatusResponse defines the object returned by the /status endpoint
type StatusResponse struct {
	Status              	string    `json:"status,omitempty"`
	Version             	string    `json:"version,omitempty"`
	LastDepartureUpdate 	time.Time `json:"last_departure_update"`
	LastParkingUpdate   	time.Time `json:"last_parking_update"`
	LastEquipmentUpdate 	time.Time `json:"last_equipment_update"`
	LastFreeFloatingUpdate 	time.Time `json:"last_free_floating_update"`
}

// ParkingResponse defines how a parking object is represent in a response
type ParkingResponse struct {
	ID                        string    `json:"car_park_id"`
	UpdatedTime               time.Time `json:"updated_time"`
	AvailableSpaces           int       `json:"available"`
	OccupiedSpaces            int       `json:"occupied"`
	AvailableAccessibleSpaces int       `json:"available_PRM"`
	OccupiedAccessibleSpaces  int       `json:"occupied_PRM"`
}

// ParkingModelToResponse converts the model of a Parking object into it's view in the response
func ParkingModelToResponse(p Parking) ParkingResponse {
	return ParkingResponse{
		ID:                        p.ID,
		UpdatedTime:               p.UpdatedTime,
		AvailableSpaces:           p.AvailableStandardSpaces,
		OccupiedSpaces:            p.TotalStandardSpaces - p.AvailableStandardSpaces,
		AvailableAccessibleSpaces: p.AvailableAccessibleSpaces,
		OccupiedAccessibleSpaces:  p.TotalAccessibleSpaces - p.AvailableAccessibleSpaces,
	}
}

type ByParkingResponseId []ParkingResponse

func (p ByParkingResponseId) Len() int           { return len(p) }
func (p ByParkingResponseId) Less(i, j int) bool { return p[i].ID < p[j].ID }
func (p ByParkingResponseId) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ParkingsResponse defines the structure returned by the /parkings endpoint
type ParkingsResponse struct {
	Parkings []ParkingResponse `json:"records,omitempty"`
	Errors   []string          `json:"errors,omitempty"`
}

// EquipmentsResponse defines the structure returned by the /equipments endpoint
type EquipmentsResponse struct {
	Equipments []EquipmentDetail `json:"equipments_details,omitempty"`
	Error      string            `json:"errors,omitempty"`
}

// FreeFloatingsResponse defines the structure returned by the /free_floatings endpoint
type FreeFloatingsResponse struct {
	FreeFloatings []FreeFloating `json:"free_floatings,omitempty"`
	Error      string            `json:"error,omitempty"`
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

func DeparturesHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := DeparturesResponse{}
		stopID, found := c.GetQueryArray("stop_id")
		if !found {
			response.Message = "stopID is required"
			c.JSON(http.StatusBadRequest, response)
			return
		}
		directionType, err := ParseDirectionTypeFromNavitia(c.Query("direction_type"))
		if err != nil {
			response.Message = err.Error()
			c.JSON(http.StatusBadRequest, response)
			return
		}
		departures, err := manager.GetDeparturesByStopsAndDirectionType(stopID, directionType)
		if err != nil {
			response.Message = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.Departures = &departures
		c.JSON(http.StatusOK, response)
	}
}

func StatusHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, StatusResponse{
			"ok",
			ForsetiVersion,
			manager.GetLastDepartureDataUpdate(),
			manager.GetLastParkingsDataUpdate(),
			manager.GetLastEquipmentsDataUpdate(),
			manager.GetLastFreeFloatingsDataUpdate(),
		})
	}
}

func ParkingsHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			parkings []Parking
			errStr   []string
		)

		if ids, ok := c.GetQueryArray("ids[]"); ok {
			// Only query parkings with a specific id
			var errs []error
			parkings, errs = manager.GetParkingsByIds(ids)
			for _, e := range errs {
				errStr = append(errStr, e.Error())
			}
		} else {
			// Query all parkings !
			var err error
			parkings, err = manager.GetParkings()
			if err != nil {
				errStr = append(errStr, err.Error())
			}
		}

		// Convert Parkings from the model to a response view
		parkingsResp := make([]ParkingResponse, len(parkings))
		for i, p := range parkings {
			parkingsResp[i] = ParkingModelToResponse(p)
		}
		c.JSON(http.StatusOK, ParkingsResponse{
			Parkings: parkingsResp,
			Errors:   errStr,
		})
	}
}

func EquipmentsHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := EquipmentsResponse{}

		equipments, err := manager.GetEquipments()
		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.Equipments = equipments
		c.JSON(http.StatusOK, response)
	}
}

func updateParameterTypes(param * FreeFloatingRequestParameter, types []string) {
	for _, value := range types {
		if isTypeValid(value) {
			param.types = append(param.types, value)
		}
	}
}

func initParameters(c *gin.Context) (param *FreeFloatingRequestParameter, err error) {
	var longitude, latitude float64
	var e error
	p := FreeFloatingRequestParameter{}
	countStr := c.DefaultQuery("count", "10")
	p.count = stringToInt(countStr, 10)
	distanceStr := c.DefaultQuery("distance", "500")
	p.distance = stringToInt(distanceStr, 500)

	types, _ := c.Request.URL.Query()["type[]"]
	updateParameterTypes(&p, types)

	coordStr := c.Query("coord")
	if len(coordStr) == 0 {
		err = fmt.Errorf("Bad request: coord is mandatory")
		return nil, err
	}
	coord := strings.Split(coordStr, ";")
	if len(coord) == 2 {
		longitudeStr := coord[0]
		latitudeStr := coord[1]
		longitude, e = strconv.ParseFloat(longitudeStr, 32)
		if e != nil {
			err = fmt.Errorf("Bad request: error on coord longitude value")
			return nil, err
		}
		latitude, e = strconv.ParseFloat(latitudeStr, 32)
		if e != nil {
			err = fmt.Errorf("Bad request: error on coord latitude value")
			return nil, err
		}
		p.coord = Coord {Lat: latitude, Lon: longitude}
	}
	return &p, nil
}

func FreeFloatingsHandler(manager *DataManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := FreeFloatingsResponse{}
		parameter, err := initParameters(c)
		if err != nil {
			response.Error = err.Error()
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		freeFloatings, err := manager.GetFreeFloatings(parameter)
		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.FreeFloatings = freeFloatings
		c.JSON(http.StatusOK, response)
	}
}

func SetupRouter(manager *DataManager, r *gin.Engine) *gin.Engine {
	if r == nil {
		r = gin.New()
	}
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, false))
	r.Use(instrumentGin())
	r.Use(gin.Recovery())
	pprof.Register(r)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/departures", DeparturesHandler(manager))
	r.GET("/status", StatusHandler(manager))
	r.GET("/parkings/P+R", ParkingsHandler(manager))
	r.GET("/equipments", EquipmentsHandler(manager))
	r.GET("/free_floatings", FreeFloatingsHandler(manager))

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
