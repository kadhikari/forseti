package vehiclelocations

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VehicleLocationsAPI(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := ConnectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	AddVehicleLocationsEntryPoint(engine, gtfsRtContext)

	// Request without locations data
	//pVehicleLocations.vehicleLocations = nil
	response := VehicleLocationsResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_locations", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.Nil(response.VehicleLocations)
	assert.Len(response.VehicleLocations, 0)
	assert.Equal(response.Error, "No data loaded")

	// Add locations data
	pVehicleLocations.vehicleLocations = vehicleLocationsMap
	assert.Equal(len(pVehicleLocations.vehicleLocations), 2)

	// Request without any parameter
	response = VehicleLocationsResponse{}
	c.Request = httptest.NewRequest("GET", "/vehicle_locations", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleLocations)
	assert.Len(response.VehicleLocations, 1)
	assert.Empty(response.Error)

	c.Request = httptest.NewRequest("GET", "/vehicle_locations?date=20210127", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleLocations)
	assert.Len(response.VehicleLocations, 2)
	assert.Empty(response.Error)

	response = VehicleLocationsResponse{}
	c.Request = httptest.NewRequest(
		"GET", "/vehicle_locations?date=20210118&vehiclejourney_id=vehicle_journey:STS:653397-1", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.VehicleLocations)
	assert.Len(response.VehicleLocations, 1)
	assert.Empty(response.Error)

	assert.Equal(response.VehicleLocations[0].VehicleJourneyId, "vehicle_journey:STS:653397-1")
	assert.Equal(response.VehicleLocations[0].Latitude, float32(45.413333892822266))
	assert.Equal(response.VehicleLocations[0].Longitude, float32(-71.87944793701172))
	assert.Equal(response.VehicleLocations[0].Bearing, float32(254))
	assert.Equal(response.VehicleLocations[0].Speed, float32(10))
}
