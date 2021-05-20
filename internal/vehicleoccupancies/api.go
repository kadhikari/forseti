package vehicleoccupancies

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var location = "Europe/Paris"

// VehicleOccupanciesResponse defines the structure returned by the /vehicle_occupancies endpoint
type VehicleOccupanciesResponse struct {
	VehicleOccupancies []VehicleOccupancy `json:"vehicle_occupancies,omitempty"`
	Error              string             `json:"error,omitempty"`
}

// Structures and functions to read files for vehicle_occupancies are here
type VehicleOccupancy struct {
	Id               int `json:"_"`
	LineCode         string
	VehicleJourneyId string `json:"vehiclejourney_id,omitempty"`
	StopId           string `json:"stop_id,omitempty"`
	Direction        int
	DateTime         time.Time `json:"date_time,omitempty"`
	Occupancy        int       `json:"occupancy"`
}

type VehicleOccupancyRequestParameter struct {
	StopId           string
	VehicleJourneyId string
	Date             time.Time
}

func InitVehicleOccupanyrequestParameter(c *gin.Context) (param *VehicleOccupancyRequestParameter) {
	p := VehicleOccupancyRequestParameter{}
	p.StopId = c.Query("stop_id")
	p.VehicleJourneyId = c.Query("vehiclejourney_id")
	loc, _ := time.LoadLocation(location)
	// We accept two date formats in the parameter
	date, err := time.ParseInLocation("20060102", c.Query("date"), loc)
	if err != nil {
		date, err = time.ParseInLocation("2006-01-02", c.Query("date"), loc)
	}
	if err != nil {
		p.Date = time.Now().Truncate(24 * time.Hour)
	} else {
		p.Date = date
	}
	return &p
}

func VehicleOccupanciesHandler(context IVehicleOccupancy) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := VehicleOccupanciesResponse{}
		parameter := InitVehicleOccupanyrequestParameter(c)
		vehicleOccupancies, err := context.GetVehicleOccupancies(parameter)

		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.VehicleOccupancies = vehicleOccupancies
		c.JSON(http.StatusOK, response)
	}
}

func AddVehicleOccupanciesEntryPoint(r *gin.Engine, context IVehicleOccupancy) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/vehicle_occupancies", VehicleOccupanciesHandler(context))
}
