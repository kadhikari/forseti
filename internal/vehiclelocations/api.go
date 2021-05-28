package vehiclelocations

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var location = "Europe/Paris"

// Structures and functions to read files for vehicle_locations are here
type VehicleLocation struct {
	Id               int       `json:"_"`
	VehicleJourneyId string    `json:"vehiclejourney_id,omitempty"`
	DateTime         time.Time `json:"date_time,omitempty"`
	Latitude         float32   `json:"latitude"`
	Longitude        float32   `json:"longitude"`
	Bearing          float32   `json:"bearing"`
	Speed            float32   `json:"speed"`
}

// VehicleLocationsResponse defines the structure returned by the /vehicle_locations endpoint
type VehicleLocationsResponse struct {
	VehicleLocations []VehicleLocation `json:"vehicle_locations,omitempty"`
	Error            string            `json:"error,omitempty"`
}

type VehicleLocationRequestParameter struct {
	VehicleJourneyId string
	Date             time.Time
}

func AddVehicleLocationsEntryPoint(r *gin.Engine, context IConnectors) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/vehicle_locations", VehicleLocationsHandler(context))
}

func VehicleLocationsHandler(context IConnectors) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := VehicleLocationsResponse{}
		parameter := InitVehicleLocationrequestParameter(c)
		vehicleLocations, err := context.GetVehicleLocations(parameter)

		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.VehicleLocations = vehicleLocations
		c.JSON(http.StatusOK, response)
	}
}

func InitVehicleLocationrequestParameter(c *gin.Context) (param *VehicleLocationRequestParameter) {
	p := VehicleLocationRequestParameter{}
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
