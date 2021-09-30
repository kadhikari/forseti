package vehiclepositions

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
	gtfsRt_vehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtureDir string

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func Test_GetVehiclePositions(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)

	connector, err := ConnectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehiclePositions := gtfsRtContext.GetAllVehiclePositions()
	require.NotNil(pVehiclePositions)

	vGtfsRt := gtfsRt_vehiclepositions.VehicleGtfsRt{VehicleID: "52103", StopId: "1280", Label: "52103", Time: 1621900800,
		Speed: 11, Bearing: 274, Route: "1", Trip: "651970", Latitude: 45.398613, Longitude: -71.90111, Occupancy: 0}

	// Call Api without vehiclePositions in the data
	param := VehiclePositionRequestParameter{VehicleJourneyCodes: []string{"651970"}, Date: date}
	_, err = gtfsRtContext.GetVehiclePositions(&param)
	require.NotNil(err)
	assert.EqualError(err, "no vehicle_locations in the data")

	// Create vehiclePositions from existing data
	vp := createVehiclePositionFromDataSource(1, vGtfsRt, location)
	gtfsRtContext.vehiclePositions.AddVehiclePosition(vp)
	require.NotNil(gtfsRtContext.vehiclePositions.vehiclePositions)
	assert.Equal(len(gtfsRtContext.vehiclePositions.vehiclePositions), 1)

	// Call Api with vehicle_journey_code
	vehiclePositions, err := gtfsRtContext.GetVehiclePositions(&param)
	require.Nil(err)
	assert.Equal(len(vehiclePositions), 1)

	// Call Api with date
	date, err = time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)
	param = VehiclePositionRequestParameter{VehicleJourneyCodes: []string{""}, Date: date}
	vehiclePositions, err = gtfsRtContext.GetVehiclePositions(&param)
	require.Nil(err)
	assert.Equal(len(vehiclePositions), 0)

	// Call Api with date before after date of vehiclePosition
	date, err = time.ParseInLocation("2006-01-02", "2021-05-26", location)
	require.Nil(err)
	param = VehiclePositionRequestParameter{VehicleJourneyCodes: []string{""}, Date: date}
	vehiclePositions, err = gtfsRtContext.GetVehiclePositions(&param)
	require.Nil(err)
	assert.Equal(len(vehiclePositions), 0)

	// Call Api with no existing vehicle_journey_code
	gtfsRtContext.CleanListVehiclePositions(1 * time.Minute)
	pVehiclePositions.vehiclePositions = map[int]*VehiclePosition{
		0: {0, "651970", date, 45.398613, -71.90111, 0, 0}}
	param = VehiclePositionRequestParameter{VehicleJourneyCodes: []string{"651969"}, Date: date}
	vehiclePositions, err = gtfsRtContext.GetVehiclePositions(&param)
	require.Nil(err)
	assert.Equal(len(vehiclePositions), 0)
}

func Test_CleanListVehiclePositions(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := ConnectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehiclePositions := gtfsRtContext.GetAllVehiclePositions()
	require.NotNil(pVehiclePositions)

	pVehiclePositions.vehiclePositions = vehiclePositionsMap
	require.NotNil(pVehiclePositions.vehiclePositions)
	gtfsRtContext.CleanListVehiclePositions(0 * time.Minute) // clean now
	assert.Equal(len(pVehiclePositions.vehiclePositions), 0)
}

func Test_InitContext(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	connector, err := ConnectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)

	uriFile, err := url.Parse("file://path_to_files/")
	require.Nil(err)
	urlExternal, err := url.Parse("http://gtfs-rt/test.pb")
	require.Nil(err)
	gtfsRtContext.InitContext(*uriFile, *urlExternal, "tokenExternal_123456789", 300, 5000, 200, location, false)
	assert.Equal(gtfsRtContext.connector.GetFilesUri(), *uriFile)
	assert.Equal(gtfsRtContext.connector.GetUrl(), *urlExternal)
	assert.Equal(gtfsRtContext.connector.GetToken(), "tokenExternal_123456789")
	assert.Equal(gtfsRtContext.connector.GetRefreshTime(), time.Duration(300))
	assert.Equal(gtfsRtContext.connector.GetConnectionTimeout(), time.Duration(200))
	assert.NotNil(gtfsRtContext.vehiclePositions)
	assert.Equal(gtfsRtContext.location, location)
	assert.NotNil(gtfsRtContext.vehiclePositions.loadOccupancyData, false)
}

var vehiclePositionsMap = map[int]*VehiclePosition{
	1: {
		Id:                 1,
		VehicleJourneyCode: "653397",
		DateTime:           time.Now(),
		Latitude:           45.413333892822266,
		Longitude:          -71.87944793701172,
		Bearing:            254,
		Speed:              10},
	2: {
		Id:                 2,
		VehicleJourneyCode: "653746",
		DateTime:           time.Now().AddDate(0, 0, -4),
		Latitude:           46.993333892822266,
		Longitude:          -73.87944793701172,
		Bearing:            120,
		Speed:              23},
}
