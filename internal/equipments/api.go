package equipments

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// EquipmentsResponse defines the structure returned by the /equipments endpoint
type EquipmentsApiResponse struct {
	Equipments []EquipmentDetail `json:"equipments_details,omitempty"`
	Error      string            `json:"errors,omitempty"`
}

func EquipmentsApiHandler(context *EquipmentsContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := EquipmentsApiResponse{}

		equipments, err := context.GetEquipments()
		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.Equipments = equipments
		c.JSON(http.StatusOK, response)
	}
}

func AddEquipmentsEntryPoint(r *gin.Engine, context *EquipmentsContext) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/equipments", EquipmentsApiHandler(context))
}

// EquipmentDetail defines how a equipment object is represented in a response
type EquipmentDetail struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	EmbeddedType        string              `json:"embedded_type"`
	CurrentAvailability CurrentAvailability `json:"current_availaibity"`
}

type CurrentAvailability struct {
	Status    string    `json:"status"`
	Cause     Cause     `json:"cause"`
	Effect    Effect    `json:"effect"`
	Periods   []Period  `json:"periods"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cause struct {
	Label string `json:"label"`
}

type Effect struct {
	Label string `json:"label"`
}

type Period struct {
	Begin time.Time `json:"begin"`
	End   time.Time `json:"end"`
}

func EmbeddedType(s string) (string, error) {
	switch {
	case s == "ASCENSEUR":
		return "elevator", nil
	case s == "ESCALIER":
		return "escalator", nil
	default:
		return "", fmt.Errorf("Unsupported EmbeddedType %s", s)
	}
}
