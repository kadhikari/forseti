package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/data"

	//"github.com/hove-io/forseti/internal/manager"
	"github.com/hove-io/forseti/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultTimeout time.Duration = time.Second * 10

func TestWithOutStopPointFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory("oditi")
	require.Nil(err)
	oditiContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesOditiContext)
	require.True(ok)
	voContext := oditiContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	// No load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s_/", fixtureDir))
	require.Nil(err)
	_, err = LoadStopPoints(*uri, defaultTimeout)
	require.Error(err)
	assert.Nil(oditiContext.stopPoints, nil)
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
	SpFileName = "mapping_stops0.csv"

	vehicleOccupanciesContext, err := VehicleOccupancyFactory("oditi")
	require.Nil(err)
	oditiContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesOditiContext)
	require.True(ok)
	voContext := oditiContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	sp, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	oditiContext.InitStopPoint(sp)

	assert.Equal(len(oditiContext.GetStopPoints()), 0)
}

func TestWithOutCoursesFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory("oditi")
	require.Nil(err)
	oditiContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesOditiContext)
	require.True(ok)

	// No load Courses from file .../extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s_/", fixtureDir))
	require.Nil(err)
	_, err = LoadCourses(*uri, defaultTimeout)
	require.Error(err)

	assert.Nil(oditiContext.courses, nil)
}

func TestCoursesFileWithOutField(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	CourseFileName = "extraction_courses0.csv"

	vehicleOccupanciesContext, err := VehicleOccupancyFactory("oditi")
	require.Nil(err)
	oditiContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesOditiContext)
	require.True(ok)
	voContext := oditiContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	course, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	oditiContext.InitCourse(course)

	assert.Equal(len(oditiContext.GetCourses()), 0)
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

func TestDataManagerForVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory("oditi")
	require.Nil(err)
	oditiContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesOditiContext)
	require.True(ok)
	voContext := oditiContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	// Load StopPoints from file .../mapping_stops.csv
	SpFileName = "mapping_stops_netex.csv"
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	oditiContext.InitStopPoint(stopPoints)
	assert.Equal(len(oditiContext.GetStopPoints()), 32)

	// Load courses from file .../extraction_courses.csv
	CourseFileName = "extraction_courses_netex.csv"
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	oditiContext.InitCourse(courses)
	assert.Equal(len(oditiContext.GetCourses()), 2)
	coursesFor40 := (oditiContext.GetCourses())["40"]
	assert.Equal(len(coursesFor40), 99)
	coursesFor45 := (oditiContext.GetCourses())["45"]
	assert.Equal(len(coursesFor45), 100)

	// Load prediction from a file
	uri, err = url.Parse(fmt.Sprintf("file://%s/predictionsNetex.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &data.PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, location)
	assert.Equal(len(predictions), 41)

	// Load vehicles journey
	uri, err = url.Parse(fmt.Sprintf("file://%s/vehicleJourneysNetex.json", fixtureDir))
	require.Nil(err)
	reader, err = utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err = ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaVJ := &NavitiaVehicleJourney{}
	err = json.Unmarshal([]byte(jsonData), navitiaVJ)
	require.Nil(err)
	vehicles := CreateVehicleJourney(navitiaVJ, time.Now())
	assert.Equal(len(vehicles), 25)
	oditiContext.InitVehicleJourneys(vehicles)

	// Create VehicleOccupancies from existing data
	occupanciesWithCharge := CreateOccupanciesFromPredictions(oditiContext, predictions)
	assert.Equal(len(occupanciesWithCharge), 34)
	voContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(voContext.VehicleOccupancies), 34)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{StopPointCodes: []string{""}, VehicleJourneyCodes: []string{""}, Date: date}
	vehicleOccupancies, err := voContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 0)

	// Call Api with StopCode in the paraameter
	param = VehicleOccupancyRequestParameter{
		StopPointCodes: []string{"stop_point:IDFM:28649"},
		Date:           date}
	vehicleOccupancies, err = oditiContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 5)
}

func TestLoadStopPointsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	SpFileName = "mapping_stops_netex.csv"

	// FileName for StopPoints should be mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Equal(len(stopPoints), 32)
	stopPoint := stopPoints["Copernic0"]
	assert.Equal(stopPoint.Name, "Copernic")
	assert.Equal(stopPoint.Direction, 0)
	assert.Equal(stopPoint.Id, "stop_point:IDFM:28650")
}

func TestLoadCoursesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	CourseFileName = "extraction_courses_netex.csv"

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// FileName for StopPoints should be extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	assert.Equal(len(courses), 2)
	course40 := courses["40"]
	assert.Equal(len(course40), 99)
	course45 := courses["45"]
	assert.Equal(len(course45), 100)
	assert.Equal(course40[0].LineCode, "40")
	assert.Equal(course40[0].Course, "3948459")
	assert.Equal(course40[0].DayOfWeek, 1)
	time, err := time.ParseInLocation("15:04:05", "06:00:00", location)
	require.Nil(err)
	assert.Equal(course40[0].FirstTime, time)
}

func TestLoadPredictionsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	uri, err := url.Parse(fmt.Sprintf("file://%s/predictionsNetex.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &data.PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, location)
	assert.Equal(len(predictions), 41)
	assert.Equal(predictions[0].LineCode, "40")
	assert.Equal(predictions[0].Order, 2)
	assert.Equal(predictions[0].Direction, 1)
	assert.Equal(predictions[0].Date, time.Date(2021, 10, 05, 0, 0, 0, 0, location))
	assert.Equal(predictions[0].Course, "3948525")
	assert.Equal(predictions[0].StopName, "Residence Europe")
	assert.Equal(predictions[0].Occupancy, 5)
}
