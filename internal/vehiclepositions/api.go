package vehiclepositions

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hove-io/forseti/internal/utils"
)

var location = "Europe/Paris"

// Structures and functions to read files for vehicle_locations are here
type VehiclePosition struct {
	VehicleJourneyCode string    `json:"vehicle_journey_code"`
	DateTime           time.Time `json:"date_time,omitempty"`
	Latitude           float32   `json:"latitude"`
	Longitude          float32   `json:"longitude"`
	Bearing            float32   `json:"bearing"`
	Speed              float32   `json:"speed"`
	Occupancy          string    `json:"occupancy,omitempty"`
	FeedCreatedAt      time.Time `json:"feed_created_at,omitempty"`
	Distance           float64   `json:"distance,omitempty"`
}

// VehiclePositionsResponse defines the structure returned by the /vehicle_locations endpoint
type VehiclePositionsResponse struct {
	VehiclePositions []VehiclePosition `json:"vehicle_positions,omitempty"`
	Error            string            `json:"error,omitempty"`
}

type VehiclePositionRequestParameter struct {
	VehicleJourneyCodes []string
	Date                time.Time
	Distance            int
	Coord               Coord
}

type Coord struct {
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
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
		parameter, err := InitVehiclePositionrequestParameter(c)
		if err != nil {
			response.Error = err.Error()
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}

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

func InitVehiclePositionrequestParameter(c *gin.Context) (param *VehiclePositionRequestParameter, err error) {
	var longitude, latitude float64
	var e error
	p := VehiclePositionRequestParameter{}
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
	p.VehicleJourneyCodes = c.Request.URL.Query()["vehicle_journey_code[]"]
	if len(p.VehicleJourneyCodes) > 0 {
		return &p, nil
	}

	distanceStr := c.DefaultQuery("distance", "")
	p.Distance = utils.StringToInt(distanceStr, 0)
	coordStr := c.Query("coord")

	// If parameter coord is present we should verify that coord is valid
	if len(coordStr) > 0 {
		p.Distance = utils.StringToInt(distanceStr, 500)

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
			p.Coord = Coord{Lat: latitude, Lon: longitude}
		} else {
			err = fmt.Errorf("Bad request: error on coord value")
			return nil, err
		}

	} else if p.Distance > 0 {
		return nil, fmt.Errorf("Bad request: coord is mandatory when distance is present")
	}
	return &p, nil
}
