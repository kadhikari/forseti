package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
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
	vehiculeOccupanciesContext.InitStopPoint(stopPoints)
	assert.Equal(len(*vehiculeOccupanciesContext.stopPoints), 4)

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
	vehiculeOccupanciesContext.InitCourse(courses)
	assert.Equal(len(*vehiculeOccupanciesContext.courses), 1)

	// Load RouteSchedules
	routeSchedules := make([]RouteSchedule, 0)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4142", "vj_id_one", "20210222T055000", 0, 2, false, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	vehiculeOccupanciesContext.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*vehiculeOccupanciesContext.routeSchedules), 2)

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
	occupanciesWithCharge := CreateOccupanciesFromPredictions(vehiculeOccupanciesContext, predictions)
	assert.Equal(len(occupanciesWithCharge), 2)
	vehiculeOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(*vehiculeOccupanciesContext.vehicleOccupancies), 2)
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
	assert.Equal(vehicleOccupancies[0].Occupancy, 55)

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
	assert.Equal(vehicleOccupancies[0].Occupancy, 75)
}

func TestLoadStopPointsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

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
