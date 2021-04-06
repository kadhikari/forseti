package forseti

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/parkings"
)

var spFileName = "mapping_stops.csv"
var courseFileName = "extraction_courses.csv"
var location = "Europe/Paris"

type VehicleOccupancyRequestParameter struct {
	StopId           string
	VehicleJourneyId string
	Date             time.Time
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

func NewVehicleOccupancy(rs RouteSchedule, occupancy int) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		Id:               rs.Id,
		LineCode:         rs.LineCode,
		VehicleJourneyId: rs.VehicleJourneyId,
		StopId:           rs.StopId,
		Direction:        rs.Direction,
		DateTime:         rs.DateTime,
		Occupancy:        occupancy,
	}, nil
}

// Data manager for all apis
type DataManager struct {
	freeFloatingsContext *freefloatings.FreeFloatingsContext

	stopPoints                   *map[string]StopPoint
	courses                      *map[string][]Course
	routeSchedules               *[]RouteSchedule
	vehicleOccupancies           *map[int]VehicleOccupancy
	lastVehicleOccupanciesUpdate time.Time
	vehicleOccupanciesMutex      sync.RWMutex
	loadOccupancyData            bool

	equipmentsContext *equipments.EquipmentsContext

	departuresContext *departures.DeparturesContext

	parkingsContext *parkings.ParkingsContext
}

func (d *DataManager) SetEquipmentsContext(equipmentsContext *equipments.EquipmentsContext) {
	d.equipmentsContext = equipmentsContext
}

func (d *DataManager) GetEquipmentsContext() *equipments.EquipmentsContext {
	return d.equipmentsContext
}

func (d *DataManager) SetFreeFloatingsContext(freeFloatingsContext *freefloatings.FreeFloatingsContext) {
	d.freeFloatingsContext = freeFloatingsContext
}

func (d *DataManager) GetFreeFloatingsContext() *freefloatings.FreeFloatingsContext {
	return d.freeFloatingsContext
}

func (d *DataManager) SetDeparturesContext(departuresContext *departures.DeparturesContext) {
	d.departuresContext = departuresContext
}

func (d *DataManager) GetDeparturesContext() *departures.DeparturesContext {
	return d.departuresContext
}

func (d *DataManager) SetParkingsContext(parkingsContext *parkings.ParkingsContext) {
	d.parkingsContext = parkingsContext
}

func (d *DataManager) GetParkingsContext() *parkings.ParkingsContext {
	return d.parkingsContext
}

func (d *DataManager) ManageVehicleOccupancyStatus(activate bool) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.loadOccupancyData = activate
}

func (d *DataManager) LoadOccupancyData() bool {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	return d.loadOccupancyData
}

func (d *DataManager) InitStopPoint(stopPoints map[string]StopPoint) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.stopPoints = &stopPoints
	fmt.Println("*** stopPoints size: ", len(*d.stopPoints))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *DataManager) GetStopId(name string, sens int) (id string) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	key := name + strconv.Itoa(sens)
	return (*d.stopPoints)[key].Id
}

func (d *DataManager) InitCourse(courses map[string][]Course) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.courses = &courses
	fmt.Println("*** courses size: ", len(*d.courses))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *DataManager) InitRouteSchedule(routeSchedules []RouteSchedule) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.routeSchedules = &routeSchedules
	fmt.Println("*** routeSchedules size: ", len(*d.routeSchedules))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *DataManager) UpdateVehicleOccupancies(vehicleOccupancies map[int]VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.vehicleOccupancies = &vehicleOccupancies
	fmt.Println("*** vehicleOccupancies size: ", len(*d.vehicleOccupancies))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *DataManager) GetLastVehicleOccupanciesDataUpdate() time.Time {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *DataManager) GetCourseFirstTime(prediction Prediction) (date_time time.Time, e error) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, course := range (*d.courses)[prediction.LineCode] {
		if prediction.Course == course.Course && int(prediction.Date.Weekday()) == course.DayOfWeek {
			return course.FirstTime, nil
		}
	}
	return time.Now(), fmt.Errorf("No corresponding data found")
}

func (d *DataManager) GetVehicleJourneyId(predict Prediction, dataTime time.Time) (vj string) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	minDiff := math.Inf(1)
	result := ""
	for _, rs := range *d.routeSchedules {
		if rs.Departure &&
			predict.LineCode == rs.LineCode &&
			predict.Direction == rs.Direction &&
			math.Abs(rs.DateTime.Sub(dataTime).Seconds()) < minDiff {
			minDiff = math.Abs(rs.DateTime.Sub(dataTime).Seconds())
			result = rs.VehicleJourneyId
		}
	}
	return result
}

func (d *DataManager) GetRouteSchedule(vjId, stopId string, direction int) (routeSchedule *RouteSchedule) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, rs := range *d.routeSchedules {
		if rs.VehicleJourneyId == vjId && rs.StopId == stopId && rs.Direction == direction {
			return &rs
		}
	}
	return nil
}

func (d *DataManager) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	var occupancies []VehicleOccupancy
	{
		d.vehicleOccupanciesMutex.RLock()
		defer d.vehicleOccupanciesMutex.RUnlock()

		if d.vehicleOccupancies == nil {
			e = fmt.Errorf("No vehicle_occupancies in the data")
			return
		}

		// Implement filter on parameters
		for _, vo := range *d.vehicleOccupancies {
			// Filter on stop_id
			if len(param.StopId) > 0 && param.StopId != vo.StopId {
				continue
			}
			// Filter on vehiclejourney_id
			if len(param.VehicleJourneyId) > 0 && param.VehicleJourneyId != vo.VehicleJourneyId {
				continue
			}
			//Fileter on datetime (default value Now)
			if vo.DateTime.Before(param.Date) {
				continue
			}
			occupancies = append(occupancies, vo)
		}
		return occupancies, nil
	}
}
