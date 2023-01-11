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

	"github.com/hove-io/forseti/internal/data"
	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/departures/sytralrt"
	"github.com/hove-io/forseti/internal/freefloatings"
	"github.com/hove-io/forseti/internal/freefloatings/fluctuo"
	"github.com/hove-io/forseti/internal/manager"
	"github.com/hove-io/forseti/internal/parkings"
	"github.com/hove-io/forseti/internal/utils"
	vehicleoccupancies "github.com/hove-io/forseti/internal/vehicleoccupancies"
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
	departuresContext.SetConnectorType("sytralrt")
	departuresContext.SetStatus("ok")

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	departures.AddDeparturesEntryPoint(router, departuresContext)
	router.GET("/status", StatusHandler(&manager))

	sytralContext := &sytralrt.SytralRTContext{}
	sytralContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
	err = sytralrt.RefreshDepartures(sytralContext, departuresContext)
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

	// Verify newly added object departures in status
	assert.True(response.Departures.RefreshActive)
	assert.True(response.Departures.LastUpdate.After(startTime))
	assert.True(response.Departures.LastUpdate.Before(time.Now()))
	assert.Equal(response.Departures.Source, "sytralrt")
	assert.Equal(response.Departures.SourceStatus, "ok")
	assert.True(response.Departures.LastStatusUpdate.After(startTime))
	assert.True(response.Departures.LastStatusUpdate.Before(time.Now()))
	assert.Equal(response.Departures.Message, "")
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

	vOditiContext, err := vehicleoccupancies.VehicleOccupancyFactory("oditi")
	require.Nil(err)
	vehicleOccupanciesOditiContext, ok := vOditiContext.(*vehicleoccupancies.VehicleOccupanciesOditiContext)
	require.True(ok)

	vehicleOccupanciesContext := vehicleOccupanciesOditiContext.GetVehicleOccupanciesContext()
	vehicleoccupancies.SpFileName = "mapping_stops_netex.csv"
	vehicleoccupancies.CourseFileName = "extraction_courses_netex.csv"

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := vehicleoccupancies.LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesOditiContext.InitStopPoint(stopPoints)
	assert.Equal(len(vehicleOccupanciesOditiContext.GetStopPoints()), 32)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := vehicleoccupancies.LoadCourses(*uri, defaultTimeout)
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
	predictions := vehicleoccupancies.LoadPredictionsData(predicts, loc)
	assert.Equal(len(predictions), 41)

	// Load vehicles journey
	uri, err = url.Parse(fmt.Sprintf("file://%s/vehicleJourneysNetex.json", fixtureDir))
	require.Nil(err)
	reader, err = utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err = ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaVJ := &vehicleoccupancies.NavitiaVehicleJourney{}
	err = json.Unmarshal([]byte(jsonData), navitiaVJ)
	require.Nil(err)
	vehicles := vehicleoccupancies.CreateVehicleJourney(navitiaVJ, time.Now())
	assert.Equal(len(vehicles), 25)
	vehicleOccupanciesOditiContext.InitVehicleJourneys(vehicles)

	// Create vehicles Occupancies
	occupanciesWithCharge := vehicleoccupancies.CreateOccupanciesFromPredictions(vehicleOccupanciesOditiContext,
		predictions)
	vehicleOccupanciesContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehicleOccupanciesContext.GetVehiclesOccupancies()), 34)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	vehicleoccupancies.AddVehicleOccupanciesEntryPoint(engine, vehicleOccupanciesOditiContext)
	engine = SetupRouter(&manager, engine)

	// Verify /status
	manager.SetVehicleOccupanciesContext(vehicleOccupanciesOditiContext)
	vehicleOccupanciesOditiContext.SetStatus("ok")
	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var status StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &status)

	require.Nil(err)
	assert.True(status.VehicleOccupancies.LastUpdate.After(startTime))
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
	assert.True(status.VehicleOccupancies.RefreshActive)
	assert.False(status.FreeFloatings.RefreshActive)

	// Verify additional attributes in /status
	assert.Equal(status.VehicleOccupancies.Source, "oditi")
	assert.Equal(status.VehicleOccupancies.SourceStatus, "ok")
	assert.True(status.VehicleOccupancies.LastStatusUpdate.After(startTime))
	assert.True(status.VehicleOccupancies.LastStatusUpdate.Before(time.Now()))
	// Empty message
	assert.Equal(status.VehicleOccupancies.Message, "")
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

	freeFloatings, err := fluctuo.LoadFreeFloatingsData(data)
	require.Nil(err)

	freeFloatingsContext := &freefloatings.FreeFloatingsContext{}

	freeFloatingsContext.UpdateFreeFloating(freeFloatings)
	freeFloatingsContext.SetStatus("ok")
	freeFloatingsContext.SetPackageName("fluctuo")
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

	// Verify additional attributes in /status
	assert.Equal(response.FreeFloatings.Source, "fluctuo")
	assert.Equal(response.FreeFloatings.SourceStatus, "ok")
	assert.True(response.FreeFloatings.LastStatusUpdate.After(startTime))
	assert.True(response.FreeFloatings.LastStatusUpdate.Before(time.Now()))
	// Empty message
	assert.Equal(response.FreeFloatings.Message, "")
}
