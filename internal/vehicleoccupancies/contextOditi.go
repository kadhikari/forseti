package vehicleoccupancies

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hove-io/forseti/google_transit"
	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/data"
	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

var SpFileName = "mapping_stops.csv"
var CourseFileName = "extraction_courses.csv"

const LINE_CODE_ODITI_40 = "line:IDFM:C00048"
const LINE_CODE_ODITI_45 = "line:IDFM:C00051"

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies ODITI objects
--------------------------------------------------------------------- */
type VehicleOccupanciesOditiContext struct {
	voContext       *VehicleOccupanciesContext
	stopPoints      *map[string]StopPoint
	courses         *map[string][]Course
	vehicleJourneys []VehicleJourney
	connector       *connectors.Connector
	//routeSchedules *[]RouteSchedule
	mutex sync.RWMutex
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

func (d *VehicleOccupanciesOditiContext) InitVehicleJourneys(vehicleJourneys []VehicleJourney) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.vehicleJourneys = vehicleJourneys
	logrus.Info("*** vehicleJourneys size: ", len(d.vehicleJourneys))
}

func (d *VehicleOccupanciesOditiContext) GetStopId(name string, sens int) (id string, direction int) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	key := name + strconv.Itoa(sens)
	return (*d.stopPoints)[key].Id, (*d.stopPoints)[key].Direction
}

func (d *VehicleOccupanciesOditiContext) InitCourse(courses map[string][]Course) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.courses = &courses
	logrus.Info("*** courses size: ", len(*d.courses))
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

func (d *VehicleOccupanciesOditiContext) GetVehicleJourneys() []VehicleJourney {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.vehicleJourneys
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

func (d *VehicleOccupanciesOditiContext) GetVehicleJourneyCode(sourceCode string) (vehicleJourneyCode string) {
	result := ""
	for _, vj := range d.vehicleJourneys {
		if strings.Contains(vj.VehicleJourneyCode, sourceCode) {
			result = vj.VehicleJourneyCode
			break
		}
	}
	return result
}

func (d *VehicleOccupanciesOditiContext) GetDepartureDatetime(vehicleJourneyCode, stopCode string) (date time.Time) {
	for _, vj := range d.vehicleJourneys {
		if vj.VehicleJourneyCode == vehicleJourneyCode {
			for _, stopC := range *vj.StopPoints {
				if stopC.StopPointId == stopCode {
					return stopC.DepartureTime
				}
			}
		}
	}
	return time.Now()
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleOccupanciesOditiContext) InitContext(filesURI, externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ,
	occupancyCleanVO, connectionTimeout time.Duration, location *time.Location, occupancyActive bool) {
	const unusedDuration time.Duration = time.Duration(-1)

	d.connector = connectors.NewConnector(
		filesURI,
		externalURI,
		externalToken,
		loadExternalRefresh,
		unusedDuration,
		connectionTimeout,
	)
	d.voContext = &VehicleOccupanciesContext{}
	d.voContext.ManageVehicleOccupancyStatus(occupancyActive)
	d.voContext.SetRereshTime(loadExternalRefresh)
	d.voContext.SetStatus("ok")
	err := LoadAllForVehicleOccupancies(d, navitiaURI, navitiaToken, location)
	if err != nil {
		logrus.Errorf("Impossible to load data at startup: %s", err)
	}
}

// main loop to refresh vehicle_occupancies from ODITI
func (d *VehicleOccupanciesOditiContext) RefreshVehicleOccupanciesLoop(externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ,
	occupancyCleanVO, connectionTimeout time.Duration, location *time.Location) {

	if len(externalURI.String()) == 0 || loadExternalRefresh.Seconds() <= 0 {
		logrus.Debug("VehicleOccupancy data refreshing is disabled")
		return
	}

	if (d.courses == nil || d.stopPoints == nil) ||
		(len(*d.courses) == 0 || len(*d.stopPoints) == 0) {
		logrus.Error("Vehicle occupancy ODITI: routine Vehicle_occupancies stopped, no stopPoints or courses loaded at start")
	} else {
		// Wait 10 seconds before reloading vehicleoccupacy informations
		time.Sleep(10 * time.Second)
		for {
			err := RefreshVehicleOccupancies(d, occupancyCleanVO, navitiaURI, navitiaToken, location)
			if err != nil {
				logrus.Error("Error while reloading VehicleOccupancy data: ", err)
			} else {
				logrus.Debug("vehicle_occupancies data updated")
			}
			time.Sleep(loadExternalRefresh)
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

func (d *VehicleOccupanciesOditiContext) GetLastStatusUpdate() time.Time {
	return d.voContext.GetLastStatusUpdate()
}

func (d *VehicleOccupanciesOditiContext) LoadOccupancyData() bool {
	return d.voContext.LoadOccupancyData()
}

func (d *VehicleOccupanciesOditiContext) GetRereshTime() string {
	return d.voContext.GetRereshTime()
}

func (d *VehicleOccupanciesOditiContext) GetStatus() string {
	return d.voContext.GetStatus()
}

func (d *VehicleOccupanciesOditiContext) SetStatus(status string) {
	d.voContext.SetStatus(status)
}

func (d *VehicleOccupanciesOditiContext) GetConnectorType() connectors.ConnectorType {
	return connectors.Connector_ODITI
}

/********* PRIVATE FUNCTIONS *********/

func RefreshVehicleOccupancies(context *VehicleOccupanciesOditiContext, occupancyCleanVO time.Duration,
	navitiaURI url.URL, navitiaToken string, location *time.Location) error {
	// Continue using last loaded data if loading is deactivated
	if !context.voContext.LoadOccupancyData() {
		return nil
	}

	begin := time.Now()
	predictions, err := LoadPredictions(context.connector, location)
	if len(predictions) == 0 {
		return fmt.Errorf("Predictions contains no data: %s", err)
	}

	occupanciesWithCharge := CreateOccupanciesFromPredictions(context, predictions)

	context.voContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func CreateOccupanciesFromPredictions(context *VehicleOccupanciesOditiContext,
	predictions []Prediction) map[string]*VehicleOccupancy {
	occupanciesWithCharge := make(map[string]*VehicleOccupancy)
	for _, predict := range predictions {
		vehicleJourneyCode := context.GetVehicleJourneyCode(predict.Course)
		if len(vehicleJourneyCode) > 0 {
			// Fetch StopId from manager corresponding to predict.StopName and predict.Direction
			stopCode, direction := context.GetStopId(predict.StopName, predict.Direction)
			date := context.GetDepartureDatetime(vehicleJourneyCode, stopCode)
			poCalc := utils.CalculateOccupancy(predict.Occupancy)
			vo, err := NewVehicleOccupancy(vehicleJourneyCode, stopCode, direction, date,
				GetOccupancyStatusForOditi(poCalc))

			if err != nil {
				continue
			}
			key := getKeyVehicleOccupancy()
			occupanciesWithCharge[key] = vo
		}
	}

	return occupanciesWithCharge
}

func LoadAllForVehicleOccupancies(context *VehicleOccupanciesOditiContext, navitiaURI url.URL, navitiaToken string,
	location *time.Location) error {
	// Load referential Stoppoints file
	stopPoints, err := LoadStopPoints(context.connector.GetFilesUri(), context.connector.GetConnectionTimeout())
	if err != nil {
		return err
	}
	context.InitStopPoint(stopPoints)

	// Load referential course file
	courses, err := LoadCourses(context.connector.GetFilesUri(), context.connector.GetConnectionTimeout())
	if err != nil {
		return err
	}
	context.InitCourse(courses)

	vehicleJourneys, err := LoadVehicleJourneysFromNavitia(context, navitiaURI, navitiaToken, location)
	if err != nil {
		return err
	}
	context.InitVehicleJourneys(vehicleJourneys)

	return nil
}

func LoadStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]StopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, SpFileName)
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
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, CourseFileName)
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

func LoadVehicleJourneysFromNavitia(context *VehicleOccupanciesOditiContext, navitiaURI url.URL, navitiaToken string,
	location *time.Location) ([]VehicleJourney, error) {
	vjs, err := GetVehiclesJourneysWithLine(LINE_CODE_ODITI_40, navitiaURI, navitiaToken,
		context.connector.GetConnectionTimeout(), location)
	if err != nil {
		fmt.Println("LoadVjLine40 Error: ", err.Error())
		return nil, err
	}
	vjs45, err := GetVehiclesJourneysWithLine(LINE_CODE_ODITI_45, navitiaURI, navitiaToken,
		context.connector.GetConnectionTimeout(), location)
	if err != nil {
		fmt.Println("LoadVjLine45 Error: ", err.Error())
		return nil, err
	}
	vjs = append(vjs, vjs45...)

	return vjs, nil
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

/* ----------------------------------------------
// Equivalence of the occupancy value Oditi vs GTFS-RT
---------------------------------------------- */

var OditiMatchMatrixGtfsRT = [4][2]int{
	// EMPTY for value equal 0
	{0, 25},  // MANY_SEATS_AVAILABLE for value between > 0 and 25
	{25, 50}, // FEW_SEATS_AVAILABLE for value between > 25 and 50
	{50, 75}, // STANDING_ROOM_ONLY for value between > 50 and 50
	{75, 99}, // CRUSHED_STANDING_ROOM_ONLY for value between > 75 and 50
	// FULL for value equal or better than 100
}

func GetOccupancyStatusForOditi(Oditi_charge int) string {
	var s int32 = 1
	if Oditi_charge == 0 {
		return google_transit.VehiclePosition_OccupancyStatus_name[0]
	}
	if Oditi_charge >= 100 {
		return google_transit.VehiclePosition_OccupancyStatus_name[5]
	}

	for idx, p := range OditiMatchMatrixGtfsRT {
		if InBetween(Oditi_charge, p[0], p[1]) {
			s = s + int32(idx)
			break
		}
	}
	return google_transit.VehiclePosition_OccupancyStatus_name[s]
}

func InBetween(charge, min, max int) bool {
	if (charge > min) && (charge <= max) {
		return true
	} else {
		return false
	}
}
