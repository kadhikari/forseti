package freefloatings

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CanalTP/forseti/internal/utils"
)

// FreeFloatingsResponse defines the structure returned by the /free_floatings endpoint
type FreeFloatingsResponse struct {
	FreeFloatings []FreeFloating `json:"free_floatings,omitempty"`
	Paginate      utils.Paginate `json:"pagination,omitempty"`
	Error         string         `json:"error,omitempty"`
}

func FreeFloatingsApiHandler(context *FreeFloatingsContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := FreeFloatingsResponse{}
		parameter, err := initFreeFloatingRequestParameter(c)
		if err != nil {
			response.Error = err.Error()
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		freeFloatings, paginate_freefloatings, err := context.GetFreeFloatings(parameter)
		if err != nil {
			response.Error = "No data loaded"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.FreeFloatings = freeFloatings
		response.Paginate = paginate_freefloatings
		c.JSON(http.StatusOK, response)
	}
}

func AddFreeFloatingsEntryPoint(r *gin.Engine, context *FreeFloatingsContext) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/free_floatings", FreeFloatingsApiHandler(context))
}

type FreeFloating struct {
	PublicId     string   `json:"public_id,omitempty"`
	ProviderName string   `json:"provider_name,omitempty"`
	Id           string   `json:"id,omitempty"`
	Type         string   `json:"type,omitempty"`
	Coord        Coord    `json:"coord,omitempty"`
	Propulsion   string   `json:"propulsion,omitempty"`
	Battery      int      `json:"battery,omitempty"`
	Deeplink     string   `json:"deeplink,omitempty"`
	Attributes   []string `json:"attributes,omitempty"`
	Distance     float64  `json:"distance,omitempty"`
}

type FreeFloatingType int

const (
	BikeType FreeFloatingType = iota
	ScooterType
	MotorScooterType
	StationType
	CarType
	OtherType
	UnknownType
)

func (f FreeFloatingType) String() string {
	return [...]string{"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER", "UNKNOWN"}[f]
}

type FreeFloatingRequestParameter struct {
	Distance  int
	Coord     Coord
	Count     int
	Types     []FreeFloatingType
	StartPage int
}

func ParseFreeFloatingTypeFromParam(value string) FreeFloatingType {
	switch strings.ToLower(value) {
	case "bike":
		return BikeType
	case "scooter":
		return ScooterType
	case "motorscooter":
		return MotorScooterType
	case "station":
		return StationType
	case "car":
		return CarType
	case "other":
		return OtherType
	default:
		return UnknownType
	}
}

func keepIt(ff FreeFloating, types []FreeFloatingType) bool {
	if len(types) == 0 {
		return true
	}
	for _, value := range types {
		if strings.EqualFold(ff.Type, value.String()) {
			return true
		}
	}
	return false
}

//FreeFloating defines how the object is represented in a response
type Coord struct {
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
}

func initFreeFloatingRequestParameter(c *gin.Context) (param *FreeFloatingRequestParameter, err error) {
	var longitude, latitude float64
	var e error
	p := FreeFloatingRequestParameter{}
	countStr := c.DefaultQuery("count", "25")
	p.Count = utils.StringToInt(countStr, 10)
	distanceStr := c.DefaultQuery("distance", "500")
	p.Distance = utils.StringToInt(distanceStr, 500)
	startPage := c.DefaultQuery("start_page", "0")
	p.StartPage = utils.StringToInt(startPage, 0)

	types := c.Request.URL.Query()["type[]"]
	UpdateParameterTypes(&p, types)

	coordStr := c.Query("coord")
	if len(coordStr) == 0 {
		return nil, fmt.Errorf("Bad request: coord is mandatory")
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
		p.Coord = Coord{Lat: latitude, Lon: longitude}
	}
	return &p, nil
}

func UpdateParameterTypes(param *FreeFloatingRequestParameter, types []string) {
	for _, value := range types {
		enumType := ParseFreeFloatingTypeFromParam(value)
		if enumType != UnknownType {
			param.Types = append(param.Types, enumType)
		}
	}
}
