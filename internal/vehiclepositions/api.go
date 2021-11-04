package vehiclepositions

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var location = "Europe/Paris"

// Structures and functions to read files for vehicle_locations are here
type VehiclePosition struct {
	Id                 int       `json:"_"`
	VehicleJourneyCode string    `json:"vehicle_journey_code"`
	DateTime           time.Time `json:"date_time,omitempty"`
	Latitude           float32   `json:"latitude"`
	Longitude          float32   `json:"longitude"`
	Bearing            float32   `json:"bearing"`
	Speed              float32   `json:"speed"`
	Occupancy          string    `json:"occupancy,omitempty"`
	FeedCreatedAt      time.Time `json:"feed_created_at,omitempty"`
}

// VehiclePositionsResponse defines the structure returned by the /vehicle_locations endpoint
type VehiclePositionsResponse struct {
	VehiclePositions []VehiclePosition `json:"vehicle_positions,omitempty"`
	Error            string            `json:"error,omitempty"`
}

type VehiclePositionRequestParameter struct {
	VehicleJourneyCodes []string
	Date                time.Time
}

func AddVehiclePositionsEntryPoint(r *gin.Engine, context IConnectors) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/vehicle_positions", VehiclePositionsHandler(context))
}

func VehiclePositionsHandler(context IConnectors) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := VehiclePositionsResponse{}
		parameter := InitVehiclePositionrequestParameter(c)
		vehiclePositions, err := context.GetVehiclePositions(parameter)

		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.VehiclePositions = vehiclePositions
		c.JSON(http.StatusOK, response)
	}
}

func InitVehiclePositionrequestParameter(c *gin.Context) (param *VehiclePositionRequestParameter) {
	p := VehiclePositionRequestParameter{}
	p.VehicleJourneyCodes = c.Request.URL.Query()["vehicle_journey_code[]"]

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
