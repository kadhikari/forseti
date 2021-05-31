package vehiclelocations

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
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

func Test_GetVehicleLocations(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	vj := VehicleJourney{VehicleID: "vehicle_journey:STS:651969-1",
		CodesSource: "651969",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			NewStopPointVj("stop_point:STS:SP:1560", "1561")},
		CreateDate: date}

	vGtfsRt := VehicleGtfsRt{VehicleID: "52103", StopId: "1280", Label: "52103", Time: 1621900800,
		Speed: 11, Bearing: 274, Route: "1", Trip: "651970", Latitude: 45.398613, Longitude: -71.90111, Occupancy: 0}

	// Call Api without vehicleLocations in the data
	param := VehicleLocationRequestParameter{VehicleJourneyId: "vehicle_journey:STS:651969-1", Date: date}
	_, err = gtfsRtContext.GetVehicleLocations(&param)
	require.NotNil(err)
	assert.EqualError(err, "no vehicle_locations in the data")

	// Create VehicleOccupancies from existing data
	vl := createVehicleLocationFromDataSource(vj, vGtfsRt, location)
	gtfsRtContext.vehicleLocations.AddVehicleLocation(vl)
	require.NotNil(gtfsRtContext.vehicleLocations.vehicleLocations)
	assert.Equal(len(gtfsRtContext.vehicleLocations.vehicleLocations), 1)

	// Call Api with vehicle_journey_id
	vehicleLocations, err := gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 1)

	// Call Api with date
	date, err = time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)
	param = VehicleLocationRequestParameter{VehicleJourneyId: "", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 1)

	// Call Api with date before after date of vehicleLocation
	date, err = time.ParseInLocation("2006-01-02", "2021-05-26", location)
	require.Nil(err)
	param = VehicleLocationRequestParameter{VehicleJourneyId: "", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 0)

	// Call Api with no existing vehicle_journey_id
	pVehicleLocations.vehicleLocations = map[int]*VehicleLocation{
		651969: {651969, "vehicle_journey:STS:651970-1", date, 45.398613, -71.90111, 0, 0}}
	param = VehicleLocationRequestParameter{VehicleJourneyId: "vehicle_journey:STS:651969-1", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 0)

}

func Test_CleanListVehicleLocations(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	pVehicleLocations.vehicleLocations = vehicleLocationsMap
	require.NotNil(pVehicleLocations.vehicleLocations)
	gtfsRtContext.CleanListVehicleLocations()
	assert.Equal(len(pVehicleLocations.vehicleLocations), 0)
}

func Test_CleanListVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	gtfsRtContext.vehiclesJourney = mapVJ
	require.NotNil(gtfsRtContext.vehiclesJourney)

	gtfsRtContext.CleanListVehicleJourney()
	require.Nil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 0)
}

func Test_CleanListOldVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	gtfsRtContext.vehiclesJourney = mapVJ
	require.NotNil(gtfsRtContext.vehiclesJourney)

	gtfsRtContext.CleanListOldVehicleJourney(2)
	require.NotNil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 2)
}

func Test_AddVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)
	pVehicleLocations := gtfsRtContext.GetAllVehicleLocations()
	require.NotNil(pVehicleLocations)

	// Add vehiclejourney with first entry
	assert.Nil(gtfsRtContext.vehiclesJourney)
	gtfsRtContext.AddVehicleJourney(&VehicleJourney{VehicleID: "vehicle_journey:STS:651999-1",
		CodesSource: "651999",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			NewStopPointVj("stop_point:STS:SP:1560", "1561")},
		CreateDate: time.Now()})
	require.NotNil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 1)

	// Add vehiclejourney with existing data
	gtfsRtContext.AddVehicleJourney(&VehicleJourney{VehicleID: "vehicle_journey:STS:651999-1",
		CodesSource: "651999",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			NewStopPointVj("stop_point:STS:SP:1560", "1561")},
		CreateDate: time.Now()})
	require.NotNil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 1)
}

func Test_InitContext(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	connector, err := connectorFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*GtfsRtContext)
	require.True(ok)

	uriFile, err := url.Parse("file://path_to_files/")
	require.Nil(err)
	urlExternal, err := url.Parse("http://gtfs-rt/test.pb")
	require.Nil(err)
	urlNavitia, err := url.Parse("https://api.navitia.io/v1/coverage/test-fr")
	require.Nil(err)
	gtfsRtContext.InitContext(*uriFile, *urlExternal, "tokenExternal_123456789", *urlNavitia, "tokenNavitia_987654321", 300, 200, location, false)
	assert.Equal(gtfsRtContext.connector.GetFilesUri(), *uriFile)
	assert.Equal(gtfsRtContext.connector.GetUrl(), *urlExternal)
	assert.Equal(gtfsRtContext.connector.GetToken(), "tokenExternal_123456789")
	assert.Equal(gtfsRtContext.navitia.GetUrl(), *urlNavitia)
	assert.Equal(gtfsRtContext.navitia.GetToken(), "tokenNavitia_987654321")
	assert.Equal(gtfsRtContext.connector.GetRefreshTime(), time.Duration(300))
	assert.Equal(gtfsRtContext.connector.GetConnectionTimeout(), time.Duration(200))
	assert.NotNil(gtfsRtContext.vehicleLocations)
	assert.Equal(gtfsRtContext.location, location)
	assert.NotNil(gtfsRtContext.vehicleLocations.loadOccupancyData, false)
}

var vehicleLocationsMap = map[int]*VehicleLocation{
	653397: {
		Id:               653397,
		VehicleJourneyId: "vehicle_journey:STS:653397-1",
		DateTime:         time.Now(),
		Latitude:         45.413333892822266,
		Longitude:        -71.87944793701172,
		Bearing:          254,
		Speed:            10},
	653746: {
		Id:               653746,
		VehicleJourneyId: "vehicle_journey:STS:653746-1",
		DateTime:         time.Now().Truncate(96 * time.Hour),
		Latitude:         46.993333892822266,
		Longitude:        -73.87944793701172,
		Bearing:          120,
		Speed:            23},
}

var mapVJ = map[string]*VehicleJourney{
	"651969": {VehicleID: "vehicle_journey:STS:651969-1",
		CodesSource: "651969",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			NewStopPointVj("stop_point:STS:SP:1560", "1560")},
		CreateDate: time.Now().Add(-2 * time.Hour)},
	"652005": {VehicleID: "vehicle_journey:STS:652005-1",
		CodesSource: "652005",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:299", "299"),
			NewStopPointVj("stop_point:STS:SP:600", "600")},
		CreateDate: time.Now().Add(-1 * time.Hour)},
	"652373": {VehicleID: "vehicle_journey:STS:652373-1",
		CodesSource: "652373",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:814", "814"),
			NewStopPointVj("stop_point:STS:SP:900", "900")},
		CreateDate: time.Now()},
}
