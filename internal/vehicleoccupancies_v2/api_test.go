package vehicleoccupanciesv2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/CanalTP/forseti/google_transit"
	"github.com/CanalTP/forseti/internal/data"
	gtfsrtvehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VehicleOccupanciesAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	//var manager manager.DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	//vehiculeOccupanciesContext := &VehicleOccupanciesContext{}
	var vehicleContext, errr = VehicleOccupancyFactory("gtfsrt")
	require.Nil(errr)
	vehicleOccupanciesContext := vehicleContext.(*VehicleOccupanciesGtfsRtContext)
	require.NotNil(vehicleOccupanciesContext)
	vehicleOccupanciesContext.voContext = &VehicleOccupanciesContext{}
	require.NotNil(vehicleOccupanciesContext.voContext)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())

	// Load positions from a file
	f, err := os.Open(fmt.Sprintf("%s/vehiclePositions.pb", fixtureDir))
	require.Nil(err)
	reader, err := ioutil.ReadAll(f)
	require.Nil(err)

	positions, err := gtfsrtvehiclepositions.ParseVehiclesResponse(reader)
	require.Nil(err)
	assert.Equal(len(positions.Vehicles), 52)

	for i := 0; i < len(positions.Vehicles); i++ {
		occupanciesWithCharge := createOccupanciesFromDataSource(positions.Vehicles[i], loc)
		vehicleOccupanciesContext.voContext.AddVehicleOccupancy(occupanciesWithCharge)
	}
	assert.Equal(len(vehicleOccupanciesContext.voContext.GetVehiclesOccupancies()), len(positions.Vehicles))

	AddVehicleOccupanciesEntryPoint(engine, vehicleOccupanciesContext)

	// Request without any parameter (Date with default value = Now().Format("20060102"))
	response := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Len(response.VehicleOccupancies, 0)
	assert.Empty(response.Error)

	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Len(response.VehicleOccupancies, 52)
	assert.Empty(response.Error)

	resp := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118&vehicle_journey_code[]=652517",
		nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 1)
	assert.Empty(resp.Error)

	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vehicle_journey_code[]=652517&vehicle_journey_code[]=999999", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 1)
	assert.Empty(resp.Error)

	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vehicle_journey_code[]=652517&vehicle_journey_code[]=652731", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 2)
	assert.Empty(resp.Error)

	// Test if parameter stop_point_code corresponding to vehicle_journey_code
	resp = VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vehicle_journey_code[]=653197&stop_point_code[]=378", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 1)
	assert.Empty(resp.Error)

	require.Nil(err)
	assert.Equal(resp.VehicleOccupancies[0].VehicleJourneyCode, "653197")
	assert.Equal(resp.VehicleOccupancies[0].StopPointCode, "378")
	assert.Equal(resp.VehicleOccupancies[0].DateTime.Format("20060102T150405"), "20210503T210832")
	assert.Equal(resp.VehicleOccupancies[0].Occupancy, google_transit.VehiclePosition_OccupancyStatus_name[1])

	// Test if parameter stop_point_code not corresponding to vehicle_journey_code
	resp = VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vehicle_journey_code[]=653197&stop_point_code[]=47", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 0)
	assert.Empty(resp.Error)
}

func Test_VehicleOccupanciesAPIWithDataFromFileOditi(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	//var manager manager.DataManager
	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	//vehiculeOccupanciesContext := &VehicleOccupanciesContext{}
	var vehicleContext, errr = VehicleOccupancyFactory("oditi")
	require.Nil(errr)
	vehicleOccupanciesContext := vehicleContext.(*VehicleOccupanciesOditiContext)
	require.NotNil(vehicleOccupanciesContext)
	vehicleOccupanciesContext.voContext = &VehicleOccupanciesContext{}
	require.NotNil(vehicleOccupanciesContext.voContext)

	SpFileName = "mapping_stops_netex.csv"
	CourseFileName = "extraction_courses_netex.csv"

	// Load StopPoints from file .../mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesContext.InitStopPoint(stopPoints)
	assert.Equal(len(vehicleOccupanciesContext.GetStopPoints()), 32)

	// Load courses from file .../extraction_courses.csv
	uri, err = url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	vehicleOccupanciesContext.InitCourse(courses)
	assert.Equal(len(vehicleOccupanciesContext.GetCourses()), 2)
	coursesFor40 := (vehicleOccupanciesContext.GetCourses())["40"]
	assert.Equal(len(coursesFor40), 99)
	coursesFor45 := (vehicleOccupanciesContext.GetCourses())["45"]
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
	predictions := LoadPredictionsData(predicts, loc)
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
	vehicleOccupanciesContext.InitVehicleJourneys(vehicles)

	// Create vehicles Occupancies
	occupanciesWithCharge := CreateOccupanciesFromPredictions(vehicleOccupanciesContext, predictions)
	vehicleOccupanciesContext.voContext.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(vehicleOccupanciesContext.voContext.GetVehiclesOccupancies()), 34)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	AddVehicleOccupanciesEntryPoint(engine, vehicleOccupanciesContext)
	//engine = SetupRouter(&manager, engine)

	// Request without any parameter (Date with default value = Now().Format("20060102"))
	response := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 34)
	assert.Empty(response.Error)

	c.Request = httptest.NewRequest("GET", "/vehicle_occupancies?date=20210118", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleOccupancies)
	assert.Len(response.VehicleOccupancies, 34)
	assert.Empty(response.Error)

	resp := VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest(
		"GET",
		"/vehicle_occupancies?date=20210118&vehicle_journey_code[]=KEOLIS:ServiceJourney:12888-C00048-3948471:LOC",
		nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 7)
	assert.Empty(resp.Error)

	resp = VehicleOccupanciesResponse{}
	c.Request = httptest.NewRequest("GET",
		"/vehicle_occupancies?date=20210118&vvehicle_journey_code[]=KEOLIS:ServiceJourney:12888-C00048-3948471:LOC&"+
			"stop_point_code[]=stop_point:IDFM:28649",
		nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.Nil(err)
	assert.Len(resp.VehicleOccupancies, 5)
	assert.Empty(resp.Error)
}
