package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	//"github.com/CanalTP/forseti/internal/manager"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultTimeout time.Duration = time.Second * 10
var fixtureDir string

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestWithOutStopPointFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}

	// No load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s_/", fixtureDir))
	require.Nil(err)
	_, err = LoadStopPoints(*uri, defaultTimeout)
	require.Error(err)
	assert.Nil(vehicleOccupanciesOditiContext.stopPoints, nil)
}

func TestNewStopPoint(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	spLine := []string{"CPC", "Copernic", "0:SP:80:4029", "0"}
	sp, err := NewStopPoint(spLine)
	require.Nil(err)
	require.NotNil(sp)

	assert.Equal(sp.Id, "stop_point:0:SP:80:4029")
	assert.Equal(sp.Name, "Copernic")
	assert.Equal(sp.Direction, 0)
}

func TestNewStoppointWithMissingField(t *testing.T) {
	require := require.New(t)
	spLine := []string{"CPC", "Copernic", "0:SP:80:4029"}
	sp, err := NewStopPoint(spLine)
	require.Error(err)
	require.Nil(sp)
}

func TestStopPointFileWithOutField(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	spFileName = "mapping_stops0.csv"

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}
	vehicleOccupanciesOditiContext.voContext = &VehicleOccupanciesContext{}

	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	sp, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesOditiContext.InitStopPoint(sp)

	assert.Equal(len(vehicleOccupanciesOditiContext.GetStopPoints()), 0)
}

func TestWithOutCoursesFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}

	// No load Courses from file .../extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s_/", fixtureDir))
	require.Nil(err)
	_, err = LoadCourses(*uri, defaultTimeout)
	require.Error(err)

	assert.Nil(vehicleOccupanciesOditiContext.courses, nil)
}

func TestCoursesFileWithOutField(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	courseFileName = "extraction_courses0.csv"

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}
	vehicleOccupanciesOditiContext.voContext = &VehicleOccupanciesContext{}

	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	course, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesOditiContext.InitCourse(course)

	assert.Equal(len(vehicleOccupanciesOditiContext.GetCourses()), 0)
}

func TestNewCourse(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	courseLine := []string{"40", "2774327", "1", "05:47:18", "06:06:16.5",
		"13", "2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	require.NotNil(course)

	assert.Equal(course.LineCode, "40")
	assert.Equal(course.Course, "2774327")
	assert.Equal(course.DayOfWeek, 1)
	assert.Equal(course.FirstDate, time.Date(2020, 9, 21, 0, 0, 0, 0, location))
	firstTime, err := time.ParseInLocation("15:04:05", "05:47:18", location)
	require.Nil(err)
	assert.Equal(course.FirstTime, firstTime)
}

func TestNewCourseWithMissingField(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	courseLine := []string{"40", "2774327", "1", "05:47:18", "06:06:16.5", "13", "2020-09-21", "2021-01-18"}
	course, err := NewCourse(courseLine, location)
	require.Error(err)
	require.Nil(course)
}

func TestNewRouteSchedule(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)
	require.Nil(err)
	require.NotNil(rs)

	assert.Equal(rs.Id, 1)
	assert.Equal(rs.LineCode, "40")
	assert.Equal(rs.VehicleJourneyId, "vj_id_one")
	assert.Equal(rs.StopId, "stop_point:0:SP:80:4029")
	assert.Equal(rs.Direction, 0)
	assert.Equal(rs.Departure, true)
	dateTime, err := time.ParseInLocation("20060102T150405", "20210222T054500", location)
	require.Nil(err)
	assert.Equal(rs.DateTime, dateTime)
}

func TestDataManagerForVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}
	vehicleOccupanciesOditiContext.voContext = &VehicleOccupanciesContext{}
	vehiculeOccupanciesContext := &VehicleOccupanciesContext{}

	// Load StopPoints
	stopPoints := make(map[string]StopPoint)
	sp, err := NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4029", "0"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4028", "1"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4142", "0"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141", "1"})
	require.Nil(err)
	_, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141", "2"})
	assert.NotNil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	vehicleOccupanciesOditiContext.InitStopPoint(stopPoints)
	assert.Equal(len(*vehicleOccupanciesOditiContext.stopPoints), 4)

	// Load Courses
	courses := make(map[string][]Course)
	courseLine := []string{"40", "2774327", "1", "05:45:00", "06:06:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	courseLine = []string{"40", "2774327", "2", "05:46:00", "06:07:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	vehicleOccupanciesOditiContext.InitCourse(courses)
	assert.Equal(len(*vehicleOccupanciesOditiContext.courses), 1)

	// Load RouteSchedules
	routeSchedules := make([]RouteSchedule, 0)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4142", "vj_id_one", "20210222T055000", 0, 2, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	vehicleOccupanciesOditiContext.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*vehicleOccupanciesOditiContext.routeSchedules), 2)

	// Load Predictions
	predictions := make([]Prediction, 0)
	pn := data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    0,
		StopName: "Copernic",
		Charge:   55}
	p := NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Pasteur",
		Charge:   75}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	assert.Equal(len(predictions), 2)

	// Create VehicleOccupancies from existing data
	occupanciesWithCharge := CreateOccupanciesFromPredictions(vehicleOccupanciesOditiContext, predictions)
	assert.Equal(len(occupanciesWithCharge), 2)
	vehiculeOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehiculeOccupanciesContext.VehicleOccupancies), 2)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{StopId: "", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err := vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 2)

	// Call Api with StopId in the paraameter
	param = VehicleOccupancyRequestParameter{
		StopId:           "stop_point:0:SP:80:4029",
		VehicleJourneyId: "",
		Date:             date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Verify attributes
	dateTime, err := time.ParseInLocation("20060102T150405", "20210222T054500", location)
	require.Nil(err)
	assert.Equal(vehicleOccupancies[0].LineCode, "40")
	assert.Equal(vehicleOccupancies[0].VehicleJourneyId, "vj_id_one")
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4029") //Copernic : departure StopPoint
	assert.Equal(vehicleOccupancies[0].Direction, 0)
	assert.Equal(vehicleOccupancies[0].DateTime, dateTime)
	assert.Equal(vehicleOccupancies[0].Occupancy, 3)

	// Call Api with another StopId in the paraameter
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4142", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	dateTime, err = time.ParseInLocation("20060102T150405", "20210222T055000", location)
	require.Nil(err)
	assert.Equal(vehicleOccupancies[0].LineCode, "40")
	assert.Equal(vehicleOccupancies[0].VehicleJourneyId, "vj_id_one")
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4142") //Pasteur
	assert.Equal(vehicleOccupancies[0].Direction, 0)
	assert.Equal(vehicleOccupancies[0].DateTime, dateTime)
	assert.Equal(vehicleOccupancies[0].Occupancy, 3)
}

func TestLoadStopPointsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	spFileName = "mapping_stops.csv"

	// FileName for StopPoints should be mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Equal(len(stopPoints), 25)
	stopPoint := stopPoints["Copernic0"]
	assert.Equal(stopPoint.Name, "Copernic")
	assert.Equal(stopPoint.Direction, 0)
	assert.Equal(stopPoint.Id, "stop_point:0:SP:80:4029")
}

func TestLoadCoursesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	courseFileName = "extraction_courses.csv"

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// FileName for StopPoints should be extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	// It's map size (line_code=40)
	assert.Equal(len(courses), 1)
	course40 := courses["40"]
	assert.Equal(len(course40), 310)
	assert.Equal(course40[0].LineCode, "40")
	assert.Equal(course40[0].Course, "2774327")
	assert.Equal(course40[0].DayOfWeek, 1)
	time, err := time.ParseInLocation("15:04:05", "05:47:18", location)
	require.Nil(err)
	assert.Equal(course40[0].FirstTime, time)
}

func TestLoadRouteSchedulesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	uri, err := url.Parse(fmt.Sprintf("file://%s/route_schedules.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaRoutes := &data.NavitiaRoutes{}
	err = json.Unmarshal([]byte(jsonData), navitiaRoutes)
	require.Nil(err)
	sens := 0
	startIndex := 1
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, location)
	assert.Equal(len(routeSchedules), 141)
	assert.Equal(routeSchedules[0].Id, 1)
	assert.Equal(routeSchedules[0].LineCode, "40")
	assert.Equal(routeSchedules[0].VehicleJourneyId, "vehicle_journey:0:123713787-1")
	assert.Equal(routeSchedules[0].StopId, "stop_point:0:SP:80:4131")
	assert.Equal(routeSchedules[0].Direction, 0)
	assert.Equal(routeSchedules[0].Departure, true)
	assert.Equal(routeSchedules[0].DateTime, time.Date(2021, 1, 18, 06, 0, 0, 0, location))
}

func TestLoadPredictionsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	uri, err := url.Parse(fmt.Sprintf("file://%s/predictions.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &data.PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, location)
	assert.Equal(len(predictions), 65)
	assert.Equal(predictions[0].LineCode, "40")
	assert.Equal(predictions[0].Order, 0)
	assert.Equal(predictions[0].Direction, 0)
	assert.Equal(predictions[0].Date, time.Date(2021, 1, 18, 0, 0, 0, 0, location))
	assert.Equal(predictions[0].Course, "2774327")
	assert.Equal(predictions[0].StopName, "Pont de Sevres")
	assert.Equal(predictions[0].Occupancy, 12)
}

func TestStatusForVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesOditiContext := &VehicleOccupanciesOditiContext{}
	vehicleOccupanciesOditiContext.voContext = &VehicleOccupanciesContext{}
	vehiculeOccupanciesContext := &VehicleOccupanciesContext{}

	// Load StopPoints
	stopPoints := make(map[string]StopPoint)
	sp, _ := NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4029", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, _ = NewStopPoint([]string{"EXL", "Exelmans", "0:SP:80:4056", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, _ = NewStopPoint([]string{"INO", "Inovel Parc Nord", "0:SP:80:4079", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, _ = NewStopPoint([]string{"LVS", "Louvois", "0:SP:80:4100", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, _ = NewStopPoint([]string{"MDT", "Marcel Dassault", "0:SP:80:4109", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, _ = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4142", "0"})
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	vehicleOccupanciesOditiContext.InitStopPoint(stopPoints)
	assert.Equal(len(*vehicleOccupanciesOditiContext.stopPoints), 6)

	// Load Courses
	courses := make(map[string][]Course)
	courseLine := []string{"40", "2774327", "1", "05:45:00", "06:06:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	courseLine = []string{"40", "2774327", "2", "05:46:00", "06:07:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	courseLine = []string{"40", "2774327", "2", "05:47:00", "06:08:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	courseLine = []string{"40", "2774327", "2", "05:48:00", "06:09:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	courseLine = []string{"40", "2774327", "2", "05:49:00", "06:10:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	courseLine = []string{"40", "2774327", "2", "05:50:00", "06:11:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)

	vehicleOccupanciesOditiContext.InitCourse(courses)
	assert.Equal(len(*vehicleOccupanciesOditiContext.courses), 1)

	// Load RouteSchedules
	routeSchedules := make([]RouteSchedule, 0)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4056", "vj_id_one", "20210222T055000", 0, 2, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4079", "vj_id_one", "20210222T055500", 0, 3, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4100", "vj_id_one", "20210222T060000", 0, 4, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4109", "vj_id_one", "20210222T060500", 0, 5, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4142", "vj_id_one", "20210222T061000", 0, 6, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	vehicleOccupanciesOditiContext.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*vehicleOccupanciesOditiContext.routeSchedules), 6)

	// Load Predictions
	predictions := make([]Prediction, 0)
	pn := data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    0,
		StopName: "Copernic",
		Charge:   0}
	p := NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Exelmans",
		Charge:   6}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Inovel Parc Nord",
		Charge:   34}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Louvois",
		Charge:   60}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Marcel Dassault",
		Charge:   76}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Pasteur",
		Charge:   100}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	assert.Equal(len(predictions), 6)

	// Create VehicleOccupancies from existing data
	occupanciesWithCharge := CreateOccupanciesFromPredictions(vehicleOccupanciesOditiContext, predictions)
	assert.Equal(len(occupanciesWithCharge), 6)
	vehiculeOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehiculeOccupanciesContext.VehicleOccupancies), 6)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{StopId: "", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err := vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 6)

	// Call Api with occupancy EMPTY
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4029", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4029") //Copernic : departure StopPoint
	assert.Equal(vehicleOccupancies[0].Occupancy, 0)

	// Call Api with occupancy MANY_SEATS_AVAILABLE
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4056", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4056") //Pasteur
	assert.Equal(vehicleOccupancies[0].Occupancy, 1)

	// Call Api with occupancy FEW_SEATS_AVAILABLE
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4079", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4079") //Pasteur
	assert.Equal(vehicleOccupancies[0].Occupancy, 2)

	// Call Api with occupancy STANDING_ROOM_ONLY
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4100", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4100") //Pasteur
	assert.Equal(vehicleOccupancies[0].Occupancy, 3)

	// Call Api with occupancy CRUSHED_STANDING_ROOM_ONLY
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4109", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4109") //Pasteur
	assert.Equal(vehicleOccupancies[0].Occupancy, 4)

	// Call Api with occupancy FULL
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4142", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = vehiculeOccupanciesContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)
	// Verify occupancy status
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4142") //Pasteur
	assert.Equal(vehicleOccupancies[0].Occupancy, 5)
}

func Test_VehicleOccupanciesAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	//var manager manager.DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehiculeOccupanciesContext := &VehicleOccupanciesContext{}

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	vehiculeOccupanciesContext.InitStopPoint(stopPoints)
	assert.Equal(len(vehiculeOccupanciesContext.GetStopPoints()), 25)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	vehiculeOccupanciesContext.InitCourse(courses)
	assert.Equal(len(vehiculeOccupanciesContext.GetCourses()), 1)
	coursesFor40 := (vehiculeOccupanciesContext.GetCourses())["40"]
	assert.Equal(len(coursesFor40), 310)

	// Load RouteSchedules from file
	uri, err = url.Parse(fmt.Sprintf("file://%s/route_schedules.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaRoutes := &data.NavitiaRoutes{}
	err = json.Unmarshal([]byte(jsonData), navitiaRoutes)
	require.Nil(err)
	sens := 0
	startIndex := 1
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, loc)
	vehiculeOccupanciesContext.InitRouteSchedule(routeSchedules)
	assert.Equal(len(vehiculeOccupanciesContext.GetRouteSchedules()), 141)

	// Load prediction from a file
	uri, err = url.Parse(fmt.Sprintf("file://%s/predictions.json", fixtureDir))
	require.Nil(err)
	reader, err = utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err = ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &data.PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, loc)
	assert.Equal(len(predictions), 65)

	occupanciesWithCharge := CreateOccupanciesFromPredictions(vehiculeOccupanciesContext, predictions)
	vehiculeOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehiculeOccupanciesContext.GetVehiclesOccupancies()), 35)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	AddVehicleOccupanciesEntryPoint(engine, vehiculeOccupanciesContext)
	//engine = SetupRouter(&manager, engine)

	// Request without any parameter (Date with default value = Now().Format("20060102"))
	response := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.Nil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 0)
	assert.Empty(response.Error)

	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 35)
	assert.Empty(response.Error)

	resp := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest(
		"GET",
		"/vehicle_occupancies?date=20210118&vehiclejourney_id=vehicle_journey:0:123713792-1",
		nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	require.NotNil(resp.VehicleOccupancies)
	assert.Len(resp.VehicleOccupancies, 7)
	assert.Empty(resp.Error)

	resp = VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vehiclejourney_id=vehicle_journey:0:123713792-1&stop_id=stop_point:0:SP:80:4121",
		nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	require.NotNil(resp.VehicleOccupancies)
	assert.Len(resp.VehicleOccupancies, 1)
	assert.Empty(resp.Error)

	require.Nil(err)
	assert.Equal(resp.VehicleOccupancies[0].VehicleJourneyId, "vehicle_journey:0:123713792-1")
	assert.Equal(resp.VehicleOccupancies[0].StopId, "stop_point:0:SP:80:4121")
	assert.Equal(resp.VehicleOccupancies[0].Direction, 0)
	assert.Equal(resp.VehicleOccupancies[0].DateTime.Format("20060102T150405"), "20210118T072200")
	assert.Equal(resp.VehicleOccupancies[0].Occupancy, 1)
}
