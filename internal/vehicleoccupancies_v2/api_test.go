package vehicleoccupanciesv2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/CanalTP/forseti/google_transit"
	gtfsrtvehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
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
		occupanciesWithCharge := createOccupanciesFromDataSource(i, positions.Vehicles[i], loc)
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
