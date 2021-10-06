package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/manager"
	"github.com/CanalTP/forseti/internal/parkings"
	"github.com/CanalTP/forseti/internal/utils"
	vehicleoccupanciesv2 "github.com/CanalTP/forseti/internal/vehicleoccupancies_v2"
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

func TestStatusApiExist(t *testing.T) {
	require := require.New(t)
	var manager manager.DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	require.Equal(200, w.Code)
}

func TestStatusApiHasLastUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	departuresContext := &departures.DeparturesContext{}
	var manager manager.DataManager
	manager.SetDeparturesContext(departuresContext)

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	departures.AddDeparturesEntryPoint(router, departuresContext)
	router.GET("/status", StatusHandler(&manager))

	err = departures.RefreshDepartures(departuresContext, *firstURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Equal(response.Status, "ok")
	assert.True(response.LastDepartureUpdate.After(startTime))
	assert.True(response.LastDepartureUpdate.Before(time.Now()))
}

func TestStatusApiHasLastParkingUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	parkingURI, err := url.Parse(fmt.Sprintf("file://%s/parkings.txt", fixtureDir))
	require.Nil(err)

	parkingsContext := &parkings.ParkingsContext{}
	var manager manager.DataManager
	manager.SetParkingsContext(parkingsContext)

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	parkings.AddParkingsEntryPoint(router, parkingsContext)
	router.GET("/status", StatusHandler(&manager))

	err = parkings.RefreshParkings(parkingsContext, *parkingURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.True(response.LastParkingUpdate.After(startTime))
	assert.True(response.LastParkingUpdate.Before(time.Now()))
}

func TestParameterTypes(t *testing.T) {
	// valid types : {"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER"}
	// As toto is not a valid type it will not be added in types
	assert := assert.New(t)
	var p freefloatings.FreeFloatingRequestParameter = freefloatings.FreeFloatingRequestParameter{}
	types := make([]string, 0)
	types = append(types, "STATION")
	types = append(types, "toto")
	types = append(types, "MOTORSCOOTER")
	types = append(types, "OTHER")

	freefloatings.UpdateParameterTypes(&p, types)
	assert.Len(p.Types, 3)
}

func TestVehicleOccupanciesAPIWithDataFromFile(t *testing.T) {
	startTime := time.Now()
	require := require.New(t)
	assert := assert.New(t)
	var manager manager.DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vOditiContext, err := vehicleoccupanciesv2.VehicleOccupancyFactory("oditi")
	require.Nil(err)
	vehicleOccupanciesOditiContext, ok := vOditiContext.(*vehicleoccupanciesv2.VehicleOccupanciesOditiContext)
	require.True(ok)

	vehicleOccupanciesContext := vehicleOccupanciesOditiContext.GetVehicleOccupanciesContext()
	vehicleoccupanciesv2.SpFileName = "mapping_stops_netex.csv"
	vehicleoccupanciesv2.CourseFileName = "extraction_courses_netex.csv"

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := vehicleoccupanciesv2.LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesOditiContext.InitStopPoint(stopPoints)
	assert.Equal(len(vehicleOccupanciesOditiContext.GetStopPoints()), 32)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := vehicleoccupanciesv2.LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesOditiContext.InitCourse(courses)
	assert.Equal(len(vehicleOccupanciesOditiContext.GetCourses()), 2)
	coursesFor40 := (vehicleOccupanciesOditiContext.GetCourses())["40"]
	assert.Equal(len(coursesFor40), 99)
	coursesFor45 := (vehicleOccupanciesOditiContext.GetCourses())["45"]
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
	predictions := vehicleoccupanciesv2.LoadPredictionsData(predicts, loc)
	assert.Equal(len(predictions), 41)

	// Load vehicles journey
	uri, err = url.Parse(fmt.Sprintf("file://%s/vehicleJourneysNetex.json", fixtureDir))
	require.Nil(err)
	reader, err = utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err = ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaVJ := &vehicleoccupanciesv2.NavitiaVehicleJourney{}
	err = json.Unmarshal([]byte(jsonData), navitiaVJ)
	require.Nil(err)
	vehicles := vehicleoccupanciesv2.CreateVehicleJourney(navitiaVJ, time.Now())
	assert.Equal(len(vehicles), 25)
	vehicleOccupanciesOditiContext.InitVehicleJourneys(vehicles)

	// Create vehicles Occupancies
	occupanciesWithCharge := vehicleoccupanciesv2.CreateOccupanciesFromPredictions(vehicleOccupanciesOditiContext,
		predictions)
	vehicleOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehicleOccupanciesContext.GetVehiclesOccupancies()), 34)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	vehicleoccupanciesv2.AddVehicleOccupanciesEntryPoint(engine, vehicleOccupanciesOditiContext)
	engine = SetupRouter(&manager, engine)

	// Verify /status
	manager.SetVehicleOccupanciesOditiContext(vehicleOccupanciesOditiContext)
	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var status StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &status)

	require.Nil(err)
	assert.False(status.VehicleOccupancies.LastUpdate.After(startTime))
	assert.True(status.VehicleOccupancies.LastUpdate.Before(time.Now()))
	assert.False(status.VehicleOccupancies.RefreshActive)
	assert.False(status.FreeFloatings.RefreshActive)

	// Activate the periodic refresh of data
	c.Request = httptest.NewRequest("GET", "/status?vehicle_occupancies=true", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &status)

	require.Nil(err)
	assert.False(status.VehicleOccupancies.RefreshActive)
	assert.False(status.FreeFloatings.RefreshActive)
}

func TestStatusInFreeFloatingWithDataFromFile(t *testing.T) {
	startTime := time.Now()
	require := require.New(t)
	assert := assert.New(t)

	// Load freefloatings from a json file
	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicles.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	data := &data.Data{}
	err = json.Unmarshal([]byte(jsonData), data)
	require.Nil(err)

	freeFloatings, err := freefloatings.LoadFreeFloatingsData(data)
	require.Nil(err)

	freeFloatingsContext := &freefloatings.FreeFloatingsContext{}

	freeFloatingsContext.UpdateFreeFloating(freeFloatings)
	c, router := gin.CreateTestContext(httptest.NewRecorder())
	freefloatings.AddFreeFloatingsEntryPoint(router, freeFloatingsContext)

	manager := &manager.DataManager{}
	manager.SetFreeFloatingsContext(freeFloatingsContext)
	router.GET("/status", StatusHandler(manager))
	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	assert.True(response.FreeFloatings.LastUpdate.After(startTime))
	assert.True(response.FreeFloatings.LastUpdate.Before(time.Now()))
	assert.False(response.FreeFloatings.RefreshActive)
	assert.False(response.VehicleOccupancies.RefreshActive)

	// Activate the periodic refresh of data
	c.Request = httptest.NewRequest("GET", "/status?free_floatings=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)

	require.Nil(err)
	assert.True(response.FreeFloatings.RefreshActive)
	assert.False(response.VehicleOccupancies.RefreshActive)
}
