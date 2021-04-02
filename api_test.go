package forseti

import (
	"encoding/json"
	"fmt"

	"io/ioutil"
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
	"github.com/CanalTP/forseti/internal/parkings"
	"github.com/CanalTP/forseti/internal/utils"
)

func TestStatusApiExist(t *testing.T) {
	require := require.New(t)
	var manager DataManager

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
	var manager DataManager
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
	var manager DataManager
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
	var manager DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	manager.InitStopPoint(stopPoints)
	assert.Equal(len(*manager.stopPoints), 25)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	manager.InitCourse(courses)
	assert.Equal(len(*manager.courses), 1)
	coursesFor40 := (*manager.courses)["40"]
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
	manager.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*manager.routeSchedules), 141)

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

	occupanciesWithCharge := CreateOccupanciesFromPredictions(&manager, predictions)
	manager.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(*manager.vehicleOccupancies), 35)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

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
	assert.Equal(resp.VehicleOccupancies[0].Occupancy, 11)

	// Verify /status
	c.Request = httptest.NewRequest("GET", "/status", nil)
	w = httptest.NewRecorder()
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

	manager := &DataManager{}
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
