package vehicleoccupancies

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

var spFileName = "mapping_stops.csv"
var courseFileName = "extraction_courses.csv"

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies ODITI objects
--------------------------------------------------------------------- */
type VehicleOccupanciesOditiContext struct {
	voContext      *VehicleOccupanciesContext
	stopPoints     *map[string]StopPoint
	courses        *map[string][]Course
	routeSchedules *[]RouteSchedule
	mutex          sync.RWMutex
}

func (d *VehicleOccupanciesOditiContext) GetVehicleOccupanciesContext() *VehicleOccupanciesContext {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.voContext == nil {
		d.voContext = &VehicleOccupanciesContext{}
	}
	return d.voContext
}

func (d *VehicleOccupanciesOditiContext) InitStopPoint(stopPoints map[string]StopPoint) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.stopPoints = &stopPoints
	logrus.Info("*** stopPoints size: ", len(*d.stopPoints))
	d.voContext.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesOditiContext) GetStopId(name string, sens int) (id string) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	key := name + strconv.Itoa(sens)
	return (*d.stopPoints)[key].Id
}

func (d *VehicleOccupanciesOditiContext) InitCourse(courses map[string][]Course) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.courses = &courses
	logrus.Info("*** courses size: ", len(*d.courses))
	d.voContext.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesOditiContext) InitRouteSchedule(routeSchedules []RouteSchedule) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.routeSchedules = &routeSchedules
	logrus.Info("*** routeSchedules size: ", len(*d.routeSchedules))
	d.voContext.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesOditiContext) GetStopPoints() map[string]StopPoint {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return *d.stopPoints
}

func (d *VehicleOccupanciesOditiContext) GetCourses() map[string][]Course {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return *d.courses
}

func (d *VehicleOccupanciesOditiContext) GetCourseFirstTime(prediction Prediction) (date_time time.Time, e error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	for _, course := range (*d.courses)[prediction.LineCode] {
		if prediction.Course == course.Course && int(prediction.Date.Weekday()) == course.DayOfWeek {
			return course.FirstTime, nil
		}
	}
	return time.Now(), fmt.Errorf("no corresponding data found")
}

func (d *VehicleOccupanciesOditiContext) GetVehicleJourneyId(predict Prediction, dataTime time.Time) (vj string) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
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

func (d *VehicleOccupanciesOditiContext) GetRouteSchedule(vjId, stopId string, direction int) (
	routeSchedule *RouteSchedule) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	for _, rs := range *d.routeSchedules {
		if rs.VehicleJourneyId == vjId && rs.StopId == stopId && rs.Direction == direction {
			return &rs
		}
	}
	return nil
}

func (d *VehicleOccupanciesOditiContext) GetRouteSchedules() (routeSchedules []RouteSchedule) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return *d.routeSchedules
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleOccupanciesOditiContext) InitContext(filesURI, externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location, occupancyActive bool) {

	d.voContext = &VehicleOccupanciesContext{}
	d.voContext.ManageVehicleOccupancyStatus(occupancyActive)

	err := LoadAllForVehicleOccupancies(d, filesURI, navitiaURI, externalURI, navitiaToken,
		externalToken, connectionTimeout, location)
	if err != nil {
		logrus.Errorf("Impossible to load data at startup: %s", err)
	}
}

// main loop to refresh vehicle_occupancies from ODITI
func (d *VehicleOccupanciesOditiContext) RefreshVehicleOccupanciesLoop(externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location) {
	if len(externalURI.String()) == 0 || loadExternalRefresh.Seconds() <= 0 {
		logrus.Debug("VehicleOccupancy data refreshing is disabled")
		return
	}

	if (d.courses == nil || d.stopPoints == nil) ||
		(len(*d.courses) == 0 || len(*d.stopPoints) == 0) {
		logrus.Error("VEHICLE_OCCUPANCIES: routine Vehicle_occupancies stopped, no stopPoints or courses loaded at start")
	} else {
		// Wait 10 seconds before reloading vehicleoccupacy informations
		time.Sleep(10 * time.Second)
		for {
			err := RefreshVehicleOccupancies(d, externalURI, externalToken, connectionTimeout, location)
			if err != nil {
				logrus.Error("Error while reloading VehicleOccupancy data: ", err)
			} else {
				logrus.Debug("vehicle_occupancies data updated")
			}
			time.Sleep(loadExternalRefresh)
		}
	}
}

func (d *VehicleOccupanciesOditiContext) RefreshDataFromNavitia(navitiaURI url.URL, navitiaToken string,
	routeScheduleRefresh, connectionTimeout time.Duration, location *time.Location) {

	if len(navitiaURI.String()) == 0 || routeScheduleRefresh.Seconds() <= 0 {
		logrus.Debug("RouteSchedule data refreshing is disabled")
		return
	}

	if (d.courses == nil || d.stopPoints == nil) ||
		(len(*d.courses) == 0 || len(*d.stopPoints) == 0) {
		logrus.Error("VEHICLE_OCCUPANCIES: routine Route_schedule stopped, no stopPoints or courses loaded at start")
	} else {
		for {
			err := LoadRoutesForAllLines(d, navitiaURI, navitiaToken, connectionTimeout, location)
			if err != nil {
				logrus.Error("Error while reloading RouteSchedule data: ", err)
			} else {
				logrus.Debug("RouteSchedule data updated")
			}
			time.Sleep(routeScheduleRefresh)
		}
	}
}

func (d *VehicleOccupanciesOditiContext) ManageVehicleOccupancyStatus(vehicleOccupanciesActive bool) {
	d.voContext.ManageVehicleOccupancyStatus(vehicleOccupanciesActive)
}

func (d *VehicleOccupanciesOditiContext) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	return d.voContext.GetVehicleOccupancies(param)
}

func (d *VehicleOccupanciesOditiContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	return d.voContext.GetLastVehicleOccupanciesDataUpdate()
}

func (d *VehicleOccupanciesOditiContext) LoadOccupancyData() bool {
	return d.voContext.LoadOccupancyData()
}

/********* PRIVATE FUNCTIONS *********/

func RefreshVehicleOccupancies(context *VehicleOccupanciesOditiContext, predict_url url.URL, predict_token string,
	connectionTimeout time.Duration, location *time.Location) error {
	// Continue using last loaded data if loading is deactivated
	if !context.voContext.LoadOccupancyData() {
		return nil
	}
	if context.routeSchedules == nil {
		return errors.New("RouteSchedule contains no data")
	}
	begin := time.Now()
	predictions, err := LoadPredictions(predict_url, predict_token, connectionTimeout, location)
	if len(predictions) == 0 {
		return fmt.Errorf("Predictions contains no data: %s", err)
	}
	occupanciesWithCharge := CreateOccupanciesFromPredictions(context, predictions)

	context.voContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func LoadStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]StopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, spFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	stopPointsConsumer := makeStopPointLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, stopPointsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return stopPointsConsumer.stopPoints, nil
}

func LoadCourses(uri url.URL, connectionTimeout time.Duration) (map[string][]Course, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, courseFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	courseLineConsumer := makeCourseLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, courseLineConsumer, loadDataOptions)
	if err != nil {
		fmt.Println("LoadCourses Error: ", err.Error())
		return nil, err
	}
	return courseLineConsumer.courses, nil
}

func LoadRoutesForAllLines(context *VehicleOccupanciesOditiContext, navitia_url url.URL, navitia_token string,
	connectionTimeout time.Duration, location *time.Location) error {
	// Load Forward RouteSchedules (sens=0) for lines 40 and 45
	startIndex := 1
	routeSchedules, err := LoadRoutesWithDirection(
		startIndex,
		navitia_url,
		navitia_token,
		"forward",
		connectionTimeout,
		location)
	if err != nil {
		return err
	}
	// Load Backward RouteSchedules (sens=1) for lines 40 and 45
	startIndex = len(routeSchedules) + 1
	backward, err := LoadRoutesWithDirection(
		startIndex,
		navitia_url,
		navitia_token,
		"backward",
		connectionTimeout,
		location)
	if err != nil {
		return err
	}

	// Concat two arrays
	for i := range backward {
		routeSchedules = append(routeSchedules, backward[i])
	}

	context.InitRouteSchedule(routeSchedules)
	return nil
}

func CreateOccupanciesFromPredictions(context *VehicleOccupanciesOditiContext,
	predictions []Prediction) map[int]*VehicleOccupancy {
	// create vehicleOccupancy with "Charge" using StopPoints and Courses in the manager for each element in Prediction
	occupanciesWithCharge := make(map[int]*VehicleOccupancy)
	var vehicleJourneyId = ""
	for _, predict := range predictions {
		if predict.Order == 0 {
			firstTime, err := context.GetCourseFirstTime(predict)
			if err != nil {
				continue
			}
			// Schedule datettime = predict.Date + firstTime
			dateTime := utils.AddDateAndTime(predict.Date, firstTime)
			vehicleJourneyId = context.GetVehicleJourneyId(predict, dateTime)
		}

		if len(vehicleJourneyId) > 0 {
			// Fetch StopId from manager corresponding to predict.StopName and predict.Direction
			stopId := context.GetStopId(predict.StopName, predict.Direction)
			rs := context.GetRouteSchedule(vehicleJourneyId, stopId, predict.Direction)
			if rs != nil {
				poCalc := utils.CalculateOccupancy(predict.Occupancy)
				vo, err := NewVehicleOccupancy(rs.Id, rs.LineCode, rs.VehicleJourneyId, rs.StopId,
					rs.Direction, rs.DateTime, int(GetOccupancyStatusForOditi(poCalc)))

				if err != nil {
					continue
				}
				occupanciesWithCharge[vo.Id] = vo
			}
		}
	}
	return occupanciesWithCharge
}

func LoadAllForVehicleOccupancies(context *VehicleOccupanciesOditiContext, files_uri, navitia_url,
	predict_url url.URL, navitia_token, predict_token string, connectionTimeout time.Duration,
	location *time.Location) error {
	// Load referential Stoppoints file
	stopPoints, err := LoadStopPoints(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	context.InitStopPoint(stopPoints)

	// Load referential course file
	courses, err := LoadCourses(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	context.InitCourse(courses)

	// Load all RouteSchedules for line 40 and 45
	err = LoadRoutesForAllLines(context, navitia_url, navitia_token, connectionTimeout, location)
	if err != nil {
		return err
	}

	// Load predictions for external service and update VehicleOccupancies with charge
	err = RefreshVehicleOccupancies(context, predict_url, predict_token, connectionTimeout, location)
	if err != nil {
		return err
	}
	return nil
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
