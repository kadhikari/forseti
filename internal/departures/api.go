package departures

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DeparturesResponse struct {
	Message    string       `json:"message,omitempty"`
	Departures *[]Departure `json:"departures,omitempty"` // the pointer allow us to display an empty array in json
}

func DeparturesApiHandler(context *DeparturesContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := DeparturesResponse{}
		/*
		if context.GetStatus() != "ok" {
			response.Message = context.GetMessage()
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
        */
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
		departures, err := context.GetDeparturesByStopsAndDirectionType(stopID, directionType)
		if err != nil {
			response.Message = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.Departures = &departures
		c.JSON(http.StatusOK, response)
	}
}

func AddDeparturesEntryPoint(r *gin.Engine, context *DeparturesContext) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/departures", DeparturesApiHandler(context))
}
