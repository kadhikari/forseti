package parkings

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

func ParkingsApiHandler(context *ParkingsContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			parkings []Parking
			errStr   []string
		)

		if ids, ok := c.GetQueryArray("ids[]"); ok {
			// Only query parkings with a specific id
			var errs []error
			parkings, errs = context.GetParkingsByIds(ids)
			for _, e := range errs {
				errStr = append(errStr, e.Error())
			}
		} else {
			// Query all parkings !
			var err error
			parkings, err = context.GetParkings()
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

func AddParkingsEntryPoint(r *gin.Engine, context *ParkingsContext) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/parkings/P+R", ParkingsApiHandler(context))
}
