package vehicleoccupancies

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/sirupsen/logrus"
)

/* -------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies objects
------------------------------------------------------------- */
type VehicleOccupanciesContext struct {
	VehicleOccupancies           *map[int]VehicleOccupancy
	VehicleOccupanciesGtfsRt     map[int]*VehicleOccupancy
	lastVehicleOccupanciesUpdate time.Time
	vehicleOccupanciesMutex      sync.RWMutex
	loadOccupancyData            bool

	stopPoints     *map[string]StopPoint
	courses        *map[string][]Course
	routeSchedules *[]RouteSchedule
}

func (d *VehicleOccupanciesContext) ManageVehicleOccupancyStatus(activate bool) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.loadOccupancyData = activate
}

func (d *VehicleOccupanciesContext) LoadOccupancyData() bool {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	return d.loadOccupancyData
}

func (d *VehicleOccupanciesContext) InitStopPoint(stopPoints map[string]StopPoint) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.stopPoints = &stopPoints
	logrus.Info("*** stopPoints size: ", len(*d.stopPoints))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) GetStopId(name string, sens int) (id string) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	key := name + strconv.Itoa(sens)
	return (*d.stopPoints)[key].Id
}

func (d *VehicleOccupanciesContext) InitCourse(courses map[string][]Course) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.courses = &courses
	logrus.Info("*** courses size: ", len(*d.courses))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) InitRouteSchedule(routeSchedules []RouteSchedule) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.routeSchedules = &routeSchedules
	logrus.Info("*** routeSchedules size: ", len(*d.routeSchedules))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) UpdateVehicleOccupancies(vehicleOccupancies map[int]VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.VehicleOccupancies = &vehicleOccupancies
	logrus.Info("*** vehicleOccupancies size: ", len(*d.VehicleOccupancies))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) CleanListVehicleOccupancies() {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	if d.VehicleOccupanciesGtfsRt != nil {
		for k := range d.VehicleOccupanciesGtfsRt {
			delete(d.VehicleOccupanciesGtfsRt, k)
		}
	}
	logrus.Info("*** Clean list VehicleOccupancies")
}
func (d *VehicleOccupanciesContext) AddVehicleOccupancy(vehicleoccupancy *VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	if d.VehicleOccupanciesGtfsRt == nil {
		d.VehicleOccupanciesGtfsRt = map[int]*VehicleOccupancy{}
	}

	d.VehicleOccupanciesGtfsRt[vehicleoccupancy.Id] = vehicleoccupancy
	logrus.Debug("*** Vehicle Occupancies size: ", len(d.VehicleOccupanciesGtfsRt))
}

func (d *VehicleOccupanciesContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *VehicleOccupanciesContext) GetStopPoints() map[string]StopPoint {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.stopPoints
}

func (d *VehicleOccupanciesContext) GetCourses() map[string][]Course {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.courses
}

func (d *VehicleOccupanciesContext) GetCourseFirstTime(prediction Prediction) (date_time time.Time, e error) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, course := range (*d.courses)[prediction.LineCode] {
		if prediction.Course == course.Course && int(prediction.Date.Weekday()) == course.DayOfWeek {
			return course.FirstTime, nil
		}
	}
	return time.Now(), fmt.Errorf("no corresponding data found")
}

func (d *VehicleOccupanciesContext) GetVehicleJourneyId(predict Prediction, dataTime time.Time) (vj string) {
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

func (d *VehicleOccupanciesContext) GetRouteSchedule(vjId, stopId string, direction int) (
	routeSchedule *RouteSchedule) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, rs := range *d.routeSchedules {
		if rs.VehicleJourneyId == vjId && rs.StopId == stopId && rs.Direction == direction {
			return &rs
		}
	}
	return nil
}

func (d *VehicleOccupanciesContext) GetRouteSchedules() (routeSchedules []RouteSchedule) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.routeSchedules
}

func (d *VehicleOccupanciesContext) GetVehiclesOccupancies() (vehicleOccupancies map[int]VehicleOccupancy) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.VehicleOccupancies
}

func (d *VehicleOccupanciesContext) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	var occupancies []VehicleOccupancy
	{
		d.vehicleOccupanciesMutex.RLock()
		defer d.vehicleOccupanciesMutex.RUnlock()

		if d.VehicleOccupancies == nil {
			e = fmt.Errorf("no vehicle_occupancies in the data")
			return
		}

		// Implement filter on parameters
		for _, vo := range *d.VehicleOccupancies {
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

func NewVehicleOccupancy(voId int, lineCode, vjId, stopId string, direction int, date time.Time,
	occupancy int) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		Id:               voId,
		LineCode:         lineCode,
		VehicleJourneyId: vjId,
		StopId:           stopId,
		Direction:        direction,
		DateTime:         date,
		Occupancy:        occupancy,
	}, nil
}

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates StopPoint objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type StopPoint struct {
	Id        string // Stoppoint uri from navitia
	Name      string // StopPoint name (same for forward and backward directions)
	Direction int    // A StopPoint id is unique for each direction
	// (forward: Direction=0 and backwward: Direction=1) in navitia
}

func NewStopPoint(record []string) (*StopPoint, error) {
	if len(record) < 4 {
		return nil, fmt.Errorf("missing field in StopPoint record")
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

/* --------------------------------------------------------------------------------
// Structure and Consumer to creates Course objects based on a line read from a CSV
-------------------------------------------------------------------------------- */
type Course struct {
	LineCode  string
	Course    string
	DayOfWeek int
	FirstDate time.Time
	FirstTime time.Time
}

func NewCourse(record []string, location *time.Location) (*Course, error) {
	if len(record) < 9 {
		return nil, fmt.Errorf("missing field in Course record")
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

/* -----------------------------------------------------
// Structure and Consumer to creates Course Line objects
----------------------------------------------------- */
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

/* ----------------------------------------------------
// Structure and Consumer to creates Prediction objects
---------------------------------------------------- */
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

/* ---------------------------------------------------------
// Structure and Consumer to creates Route_schedules objects
--------------------------------------------------------- */
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
