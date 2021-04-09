package vehicleoccupancies

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CanalTP/forseti/internal/data"
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

func VehicleOccupanciesHandler(context *VehicleOccupanciesContext) gin.HandlerFunc {
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

func AddVehicleOccupanciesEntryPoint(r *gin.Engine, context *VehicleOccupanciesContext) {
	if r == nil {
		r = gin.New()
	}
	r.GET("/vehicle_occupancies", VehicleOccupanciesHandler(context))
}

// Structure and Consumer to creates StopPoint objects based on a line read from a CSV
type StopPoint struct {
	Id        string // Stoppoint uri from navitia
	Name      string // StopPoint name (same for forward and backward directions)
	Direction int    // A StopPoint id is unique for each direction
	// (forward: Direction=0 and backwward: Direction=1) in navitia
}

func NewStopPoint(record []string) (*StopPoint, error) {
	if len(record) < 4 {
		return nil, fmt.Errorf("Missing field in StopPoint record")
	}

	direction, err := strconv.Atoi(record[3])
	if err != nil {
		return nil, err
	}
	if direction != 0 && direction != 1 {
		return nil, fmt.Errorf("only 0 or 1 is permitted as sens ")
	}
	return &StopPoint{
		Id:        fmt.Sprintf("%s:%s", "stop_point", record[2]),
		Name:      record[1],
		Direction: direction,
	}, nil
}

type StopPointLineConsumer struct {
	stopPoints map[string]StopPoint
}

func makeStopPointLineConsumer() *StopPointLineConsumer {
	return &StopPointLineConsumer{
		stopPoints: make(map[string]StopPoint),
	}
}

func (c *StopPointLineConsumer) Consume(line []string, loc *time.Location) error {
	stopPoint, err := NewStopPoint(line)
	if err != nil {
		return err
	}
	key := stopPoint.Name + strconv.Itoa(stopPoint.Direction)
	c.stopPoints[key] = *stopPoint
	return nil
}
func (c *StopPointLineConsumer) Terminate() {}

// Structure and Consumer to creates Course objects based on a line read from a CSV
type Course struct {
	LineCode  string
	Course    string
	DayOfWeek int
	FirstDate time.Time
	FirstTime time.Time
}

func NewCourse(record []string, location *time.Location) (*Course, error) {
	if len(record) < 9 {
		return nil, fmt.Errorf("Missing field in Course record")
	}
	dow, err := strconv.Atoi(record[2])
	if err != nil {
		return nil, err
	}
	firstDate, err := time.ParseInLocation("2006-01-02", record[6], location)
	if err != nil {
		return nil, err
	}
	firstTime, err := time.ParseInLocation("15:04:05", record[3], location)
	if err != nil {
		return nil, err
	}
	return &Course{
		LineCode:  record[0],
		Course:    record[1],
		DayOfWeek: dow,
		FirstDate: firstDate,
		FirstTime: firstTime,
	}, nil
}

type CourseLineConsumer struct {
	courses map[string][]Course
}

func makeCourseLineConsumer() *CourseLineConsumer {
	return &CourseLineConsumer{make(map[string][]Course)}
}

func (c *CourseLineConsumer) Consume(line []string, loc *time.Location) error {
	course, err := NewCourse(line, loc)
	if err != nil {
		return err
	}

	c.courses[course.LineCode] = append(c.courses[course.LineCode], *course)
	return nil
}
func (p *CourseLineConsumer) Terminate() {}

// Structure for prediction
type Prediction struct {
	LineCode  string
	Order     int
	Direction int
	Date      time.Time
	Course    string
	StopName  string
	Occupancy int
	CreatedAt time.Time
}

func NewPrediction(p data.PredictionNode, location *time.Location) *Prediction {
	date, err := time.ParseInLocation("2006-01-02T15:04:05", p.Date, location)
	if err != nil {
		return &Prediction{}
	}

	return &Prediction{
		LineCode:  p.Line,
		Direction: p.Sens,
		Order:     p.Order,
		Date:      date,
		Course:    p.Course,
		StopName:  p.StopName,
		Occupancy: int(p.Charge),
	}
}

// Structure for route_schedules
type RouteSchedule struct {
	Id               int
	LineCode         string
	VehicleJourneyId string
	StopId           string
	Direction        int
	Departure        bool
	DateTime         time.Time
}

func NewRouteSchedule(lineCode, stopId, vjId, dateTime string, sens, Id int, depart bool,
	location *time.Location) (*RouteSchedule, error) {
	date, err := time.ParseInLocation("20060102T150405", dateTime, location)
	if err != nil {
		fmt.Println("Error on: ", dateTime)
		return nil, err
	}
	return &RouteSchedule{
		Id:               Id,
		LineCode:         lineCode,
		VehicleJourneyId: vjId,
		StopId:           stopId,
		Direction:        sens,
		Departure:        depart,
		DateTime:         date,
	}, nil
}
