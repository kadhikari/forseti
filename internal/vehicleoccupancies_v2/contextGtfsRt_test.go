package vehicleoccupanciesv2

import (
	"net/url"
	"os"
	"testing"
	"time"

	"reflect"

	"github.com/CanalTP/forseti/google_transit"
	"github.com/CanalTP/forseti/internal/connectors"
	gtfsrtvehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
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

func Test_InitContext(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	connector, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := connector.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)

	uriFile, err := url.Parse("file://path_to_files/")
	require.Nil(err)
	urlExternal, err := url.Parse("http://gtfs-rt/test.pb")
	require.Nil(err)
	urlNavitia, err := url.Parse("")
	require.Nil(err)
	gtfsRtContext.InitContext(*uriFile, *urlExternal, "tokenExternal_123456789", *urlNavitia, "", 300, 5000,
		5000, 200, location, false)
	assert.Equal(gtfsRtContext.connector.GetFilesUri(), *uriFile)
	assert.Equal(gtfsRtContext.connector.GetUrl(), *urlExternal)
	assert.Equal(gtfsRtContext.connector.GetToken(), "tokenExternal_123456789")
	assert.Equal(gtfsRtContext.connector.GetRefreshTime(), time.Duration(300))
	assert.Equal(gtfsRtContext.connector.GetConnectionTimeout(), time.Duration(200))
	assert.NotNil(gtfsRtContext.voContext)
	assert.NotNil(gtfsRtContext.voContext.loadOccupancyData, false)
}

func Test_CleanListVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	voContext.VehicleOccupancies = vehicleOccupanciesMap
	require.NotNil(voContext.VehicleOccupancies)
	gtfsRtContext.CleanListVehicleOccupancies(0 * time.Minute) // clean now
	assert.Equal(len(voContext.VehicleOccupancies), 0)
}

func Test_CleanListVehicleOccupancyForOldVo(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	// Create VehicleOccupancies from existing data
	vGtfsRt := gtfsrtvehiclepositions.VehicleGtfsRt{VehicleID: "52103", StopId: "263", Label: "52103",
		Time: 1620076139, Speed: 11, Bearing: 274, Route: "1", Trip: "652517", Latitude: 45.398613,
		Longitude: -71.90111, Occupancy: 0}
	vo := createOccupanciesFromDataSource(vGtfsRt, location)
	vo.DateTime = time.Now()
	gtfsRtContext.AddVehicleOccupancy(vo)

	vGtfsRt = gtfsrtvehiclepositions.VehicleGtfsRt{VehicleID: "52105", StopId: "47", Label: "52105",
		Time: 1620076116, Speed: 12, Bearing: 268, Route: "12", Trip: "652604", Latitude: 45.40917,
		Longitude: -71.930275, Occupancy: 00}
	vo = createOccupanciesFromDataSource(vGtfsRt, location)
	vo.DateTime = time.Now()
	gtfsRtContext.AddVehicleOccupancy(vo)

	vGtfsRt = gtfsrtvehiclepositions.VehicleGtfsRt{VehicleID: "53101", StopId: "1326", Label: "53101",
		Time: 1620076034, Speed: 1, Bearing: 262, Route: "17", Trip: "653036", Latitude: 45.40333,
		Longitude: -71.95417, Occupancy: 00}
	vo = createOccupanciesFromDataSource(vGtfsRt, location)
	gtfsRtContext.AddVehicleOccupancy(vo)
	assert.Equal(len(voContext.VehicleOccupancies), 3)

	require.NotNil(voContext.VehicleOccupancies)
	gtfsRtContext.CleanListVehicleOccupancies(5 * time.Minute)
	assert.Equal(len(voContext.VehicleOccupancies), 2)
}

func Test_AddVehicleOccupancy(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	gtfsRtContext.voContext = &VehicleOccupanciesContext{}
	require.NotNil(gtfsRtContext.voContext)

	gtfsRtContext.AddVehicleOccupancy(&VehicleOccupancy{
		VehicleJourneyCode: "vehicle_journey:0:124695149-1",
		StopPointCode:      "stop_point:0:SP:80:4029",
		Direction:          0,
		DateTime:           time.Now(),
		Occupancy:          google_transit.VehiclePosition_OccupancyStatus_name[1]})
	require.NotNil(gtfsRtContext.voContext.VehicleOccupancies)
	assert.Equal(len(gtfsRtContext.voContext.VehicleOccupancies), 1)
}

func Test_UpdateVehicleOccupancy(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	vGtfsRt := gtfsrtvehiclepositions.VehicleGtfsRt{
		VehicleID: "52103",
		StopId:    "263",
		Label:     "52103",
		Time:      1620777600,
		Speed:     11,
		Bearing:   274,
		Route:     "1",
		Trip:      "652517",
		Latitude:  45.398613,
		Longitude: -71.90111,
		Occupancy: 1}

	// Create VehicleOccupancies from existing data
	vo := createOccupanciesFromDataSource(vGtfsRt, location)
	gtfsRtContext.AddVehicleOccupancy(vo)
	require.NotNil(voContext.VehicleOccupancies)
	assert.Equal(len(voContext.VehicleOccupancies), 1)
	gtfsRtContext.UpdateOccupancy(vo, vGtfsRt, location)
	keys := reflect.ValueOf(voContext.VehicleOccupancies).MapKeys()
	first_key := keys[0].String()
	assert.Equal(voContext.VehicleOccupancies[first_key].Occupancy, google_transit.VehiclePosition_OccupancyStatus_name[1])
}

func Test_GetVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	// Create VehicleOccupancies from existing data
	for i := 0; i < 5; i++ {
		vo := createOccupanciesFromDataSource(dataGtfsRt[i], location)
		gtfsRtContext.AddVehicleOccupancy(vo)
	}
	require.NotNil(voContext.VehicleOccupancies)
	assert.Equal(len(voContext.VehicleOccupancies), 5)

	date, err := time.ParseInLocation("2006-01-02", "2021-05-01", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{Date: date}
	vehicleOccupancies, err := gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 5)

	// Call Api with single StopCode parameter and without vehicleJourneyCode
	param = VehicleOccupancyRequestParameter{StopPointCodes: []string{"263"}, Date: date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Call Api when multiple StopCode and without vehicleJourneyCode
	param = VehicleOccupancyRequestParameter{StopPointCodes: []string{"1326", "185"}, Date: date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 2)

	// Call Api when single vehicleJourneyCode and without StopCode
	param = VehicleOccupancyRequestParameter{VehicleJourneyCodes: []string{"652712"}, Date: date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Call Api when multiple vehicleJourneyCode and without StopCode
	param = VehicleOccupancyRequestParameter{VehicleJourneyCodes: []string{"652604", "652712"}, Date: date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 2)

	// Call Api when multiple vehicleJourneyCode and StopCode
	param = VehicleOccupancyRequestParameter{StopPointCodes: []string{"1326", "185"},
		VehicleJourneyCodes: []string{"652604", "652712"}, Date: date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Call Api when StopCode vehicleOcupancies list is nil
	param = VehicleOccupancyRequestParameter{StopPointCodes: []string{"1326"}, Date: date}
	voContext.VehicleOccupancies = nil
	_, err = gtfsRtContext.GetVehicleOccupancies(&param)
	assert.Error(err, "no vehicle_occupancies in the data")
}

var vehicleOccupanciesMap = map[string]*VehicleOccupancy{
	getKeyVehicleOccupancy(): {
		VehicleJourneyCode: "vehicle_journey_code:123456",
		StopPointCode:      "stop_point_code:987",
		Direction:          0,
		DateTime:           time.Now(),
		Occupancy:          google_transit.VehiclePosition_OccupancyStatus_name[1]},
	getKeyVehicleOccupancy(): {
		VehicleJourneyCode: "vehicle_journey_code:789012",
		StopPointCode:      "stop_point_code:456",
		Direction:          0,
		DateTime:           time.Now(),
		Occupancy:          google_transit.VehiclePosition_OccupancyStatus_name[2]},
	getKeyVehicleOccupancy(): {
		VehicleJourneyCode: "vehicle_journey_code:652517",
		StopPointCode:      "stop_point_code:263",
		Direction:          0,
		DateTime:           time.Now().Add(time.Duration(300)),
		Occupancy:          google_transit.VehiclePosition_OccupancyStatus_name[2]},
}

var dataGtfsRt = []gtfsrtvehiclepositions.VehicleGtfsRt{
	{VehicleID: "52103", StopId: "263", Label: "52103", Time: 1620076139, Speed: 11, Bearing: 274, Route: "1",
		Trip: "652517", Latitude: 45.398613, Longitude: -71.90111, Occupancy: 0},
	{VehicleID: "52105", StopId: "47", Label: "52105", Time: 1620076116, Speed: 12, Bearing: 268, Route: "12",
		Trip: "652604", Latitude: 45.40917, Longitude: -71.930275, Occupancy: 00},
	{VehicleID: "53101", StopId: "1326", Label: "53101", Time: 1620076034, Speed: 1, Bearing: 262, Route: "17",
		Trip: "653036", Latitude: 45.40333, Longitude: -71.95417, Occupancy: 00},
	{VehicleID: "53103", StopId: "185", Label: "53103", Time: 1620076121, Speed: 11, Bearing: 0, Route: "16",
		Trip: "652712", Latitude: 45.421112, Longitude: -71.91722, Occupancy: 00},
	{VehicleID: "53105", StopId: "171", Label: "53105", Time: 1620076127, Speed: 0, Bearing: 60, Route: "11",
		Trip: "652883", Latitude: 45.3825, Longitude: -71.9275, Occupancy: 00},
	{VehicleID: "54103", StopId: "800", Label: "54103", Time: 1620076143, Speed: 14, Bearing: 41, Route: "7",
		Trip: "652543", Latitude: 45.42389, Longitude: -71.8725, Occupancy: 00},
	{VehicleID: "55103", StopId: "1292", Label: "55103", Time: 1620076107, Speed: 13, Bearing: 126, Route: "12",
		Trip: "652530", Latitude: 45.41028, Longitude: -71.93445, Occupancy: 00},
	{VehicleID: "55104", StopId: "2454", Label: "55104", Time: 1620076084, Speed: 0, Bearing: 94, Route: "17",
		Trip: "652731", Latitude: 45.34861, Longitude: -71.99167, Occupancy: 00},
	{VehicleID: "57101", StopId: "7005", Label: "57101", Time: 1620076006, Speed: 0, Bearing: 136, Route: "6",
		Trip: "652817", Latitude: 45.378334, Longitude: -71.92917, Occupancy: 00},
	{VehicleID: "57103", StopId: "7000", Label: "57103", Time: 1620076012, Speed: 0, Bearing: 10, Route: "9",
		Trip: "652787", Latitude: 45.378613, Longitude: 71.92917, Occupancy: 00},
	{VehicleID: "58102", StopId: "2019", Label: "58102", Time: 1620076132, Speed: 0, Bearing: 304, Route: "2",
		Trip: "652740", Latitude: 45.405834, Longitude: -71.886665, Occupancy: 01},
	{VehicleID: "59102", StopId: "330", Label: "59102", Time: 1620076143, Speed: 14, Bearing: 60, Route: "3",
		Trip: "652984", Latitude: 45.39722, Longitude: -71.94334, Occupancy: 01},
	{VehicleID: "59103", StopId: "234", Label: "59103", Time: 1620076030, Speed: 9, Bearing: 126, Route: "8",
		Trip: "652772", Latitude: 45.45, Longitude: -71.865555, Occupancy: 01},
	{VehicleID: "60102", StopId: "242", Label: "60102", Time: 1620076094, Speed: 0, Bearing: 184, Route: "8",
		Trip: "652830", Latitude: 45.422222, Longitude: -71.87417, Occupancy: 01},
	{VehicleID: "60107", StopId: "581", Label: "60107", Time: 1620076136, Speed: 13, Bearing: 48, Route: "9",
		Trip: "652670", Latitude: 45.388054, Longitude: -71.90833, Occupancy: 00},
	{VehicleID: "60108", StopId: "7002", Label: "60108", Time: 1620075990, Speed: 0, Bearing: 18, Route: "8",
		Trip: "652659", Latitude: 45.378613, Longitude: -71.92944, Occupancy: 01},
	{VehicleID: "61103", StopId: "993", Label: "61103", Time: 1620076006, Speed: 18.9, Bearing: 174, Route: "11",
		Trip: "653112", Latitude: 45.40278, Longitude: -71.96111, Occupancy: 01},
	{VehicleID: "61104", StopId: "7004", Label: "61104", Time: 1620076092, Speed: 0, Bearing: 16, Route: "16",
		Trip: "653028", Latitude: 45.378613, Longitude: -71.929726, Occupancy: 00},
	{VehicleID: "62101", StopId: "6003", Label: "62101", Time: 1620075988, Speed: 0, Bearing: 264, Route: "7",
		Trip: "652905", Latitude: 45.411667, Longitude: -71.884445, Occupancy: 00},
	{VehicleID: "64104", StopId: "270", Label: "64104", Time: 1620076130, Speed: 8, Bearing: 260, Route: "3",
		Trip: "653044", Latitude: 45.398335, Longitude: -71.92389, Occupancy: 01},
	{VehicleID: "65101", StopId: "588", Label: "65101", Time: 1620076103, Speed: 0, Bearing: 4, Route: "8",
		Trip: "652940", Latitude: 45.394444, Longitude: -71.89611, Occupancy: 01},
	{VehicleID: "65105", StopId: "415", Label: "65105", Time: 1620076103, Speed: 0, Bearing: 62, Route: "3",
		Trip: "652959", Latitude: 45.411945, Longitude: -71.87139, Occupancy: 01},
	{VehicleID: "65106", StopId: "9991", Label: "65106", Time: 1620076130, Speed: 0, Bearing: 278, Route: "Sortie",
		Trip: "652701", Latitude: 45.391666, Longitude: -71.928055, Occupancy: 01},
	{VehicleID: "65107", StopId: "699", Label: "65107", Time: 1620076125, Speed: 0, Bearing: 90, Route: "14",
		Trip: "652890", Latitude: 45.38389, Longitude: -71.88695, Occupancy: 01},
	{VehicleID: "65109", StopId: "2038", Label: "65109", Time: 1620076111, Speed: 13, Bearing: 272, Route: "4",
		Trip: "653126", Latitude: 45.40278, Longitude: -71.825836, Occupancy: 01},
	{VehicleID: "65111", StopId: "214", Label: "65111", Time: 1620076114, Speed: 0, Bearing: 0, Route: "8",
		Trip: "653283", Latitude: 45.40361, Longitude: -71.87334, Occupancy: 01},
	{VehicleID: "66101", StopId: "370", Label: "66101", Time: 1620076130, Speed: 12, Bearing: 268, Route: "4",
		Trip: "652647", Latitude: 45.40111, Longitude: -71.86889, Occupancy: 01},
	{VehicleID: "66103", StopId: "2125", Label: "66103", Time: 1620076130, Speed: 0, Bearing: 0, Route: "11",
		Trip: "653058", Latitude: 45.413612, Longitude: -71.94444, Occupancy: 01},
	{VehicleID: "67101", StopId: "7002", Label: "67101", Time: 1620075993, Speed: 1.1, Bearing: 300, Route: "14",
		Trip: "652913", Latitude: 45.378887, Longitude: -71.92944, Occupancy: 01},
	{VehicleID: "67102", StopId: "524", Label: "67102", Time: 1620076141, Speed: 0, Bearing: 184, Route: "7",
		Trip: "652845", Latitude: 45.386112, Longitude: -71.90139, Occupancy: 1},
	{VehicleID: "67104", StopId: "436", Label: "67104", Time: 1620076135, Speed: 9, Bearing: 178, Route: "5",
		Trip: "653230", Latitude: 45.4275, Longitude: -71.88722, Occupancy: 1},
	{VehicleID: "67106", StopId: "2082", Label: "67106", Time: 1620076101, Speed: 0, Bearing: 0, Route: "7",
		Trip: "652780", Latitude: 45.378887, Longitude: -71.895, Occupancy: 1},
	{VehicleID: "68101", StopId: "1721", Label: "68101", Time: 1620076133, Speed: 0, Bearing: 0, Route: "18",
		Trip: "653247", Latitude: 45.384445, Longitude: -71.975555, Occupancy: 1},
	{VehicleID: "68105", StopId: "297", Label: "68105", Time: 1620076117, Speed: 0, Bearing: 0, Route: "3",
		Trip: "653023", Latitude: 45.4025, Longitude: -71.96333, Occupancy: 0},
	{VehicleID: "69101", StopId: "8002", Label: "69101", Time: 1620076107, Speed: 0, Bearing: 0, Route: "12",
		Trip: "653164", Latitude: 45.40361, Longitude: -71.949165, Occupancy: 1},
	{VehicleID: "69104", StopId: "2295", Label: "69104", Time: 1620076076, Speed: 8, Bearing: 178, Route: "9",
		Trip: "653012", Latitude: 45.41028, Longitude: -71.851944, Occupancy: 1},
	{VehicleID: "69105", StopId: "1134", Label: "69105", Time: 1620076055, Speed: 12, Bearing: 178, Route: "5",
		Trip: "653265", Latitude: 45.418888, Longitude: -71.882225, Occupancy: 0},
	{VehicleID: "70101", StopId: "378", Label: "70101", Time: 1620076112, Speed: 0, Bearing: 262, Route: "17",
		Trip: "653197", Latitude: 45.40361, Longitude: -71.91333, Occupancy: 1},
	{VehicleID: "70102", StopId: "2273", Label: "70102", Time: 1620076050, Speed: 3, Bearing: 226, Route: "3",
		Trip: "652564", Latitude: 45.421944, Longitude: -71.87222, Occupancy: 1},
	{VehicleID: "70104", StopId: "1137", Label: "70104", Time: 1620076133, Speed: 0, Bearing: 176, Route: "14",
		Trip: "652998", Latitude: 45.39111, Longitude: -71.89056, Occupancy: 1},
	{VehicleID: "70105", StopId: "2111", Label: "70105", Time: 1620076142, Speed: 0, Bearing: 246, Route: "9",
		Trip: "652858", Latitude: 45.410557, Longitude: -71.84055, Occupancy: 1},
	{VehicleID: "53104", StopId: "6000", Label: "53104", Time: 1620076057, Speed: 0, Bearing: 90, Route: "Entree",
		Trip: "653145", Latitude: 45.41179, Longitude: -71.88419, Occupancy: 0},
	{VehicleID: "56104", StopId: "2162", Label: "56104", Time: 1620076004, Speed: 8, Bearing: 178, Route: "19",
		Trip: "652934", Latitude: 45.384125, Longitude: -71.944145, Occupancy: 0},
	{VehicleID: "58104", StopId: "8000", Label: "58104", Time: 1620075972, Speed: 0, Bearing: 0, Route: "Entree",
		Trip: "652969", Latitude: 45.402363, Longitude: -71.95269, Occupancy: 1},
	{VehicleID: "58101", StopId: "234", Label: "58101", Time: 1620075782, Speed: 5, Bearing: 216, Route: "8",
		Trip: "653213", Latitude: 45.44625, Longitude: -71.869576, Occupancy: 1},
	{VehicleID: "52101", StopId: "6000", Label: "52101", Time: 1620075746, Speed: 9, Bearing: 132, Route: "Entree",
		Trip: "652467", Latitude: 45.411736, Longitude: -71.884796, Occupancy: 0},
	{VehicleID: "67103", StopId: "234", Label: "67103", Time: 1620075508, Speed: 0, Bearing: 218, Route: "7",
		Trip: "652918", Latitude: 45.44625, Longitude: -71.869576, Occupancy: 3},
	{VehicleID: "70103", StopId: "7005", Label: "70103", Time: 1620076110, Speed: 13, Bearing: 200, Route: "6",
		Trip: "652255", Latitude: 45.385, Longitude: -71.924446, Occupancy: 1},
	{VehicleID: "62102", StopId: "6001", Label: "62102", Time: 1620075961, Speed: 0, Bearing: 173, Route: "12",
		Trip: "652221", Latitude: 45.408054, Longitude: -71.88333, Occupancy: 1},
	{VehicleID: "64102", StopId: "5100", Label: "64102", Time: 1620075922, Speed: 5, Bearing: 52, Route: "8",
		Trip: "652205", Latitude: 45.40167, Longitude: -71.889725, Occupancy: 1},
	{VehicleID: "66105", StopId: "7004", Label: "66105", Time: 1620075859, Speed: 0, Bearing: 16, Route: "18",
		Trip: "653078", Latitude: 45.378056, Longitude: -71.92917, Occupancy: 1},
	{VehicleID: "69103", StopId: "5200", Label: "69103", Time: 1620075320, Speed: 6, Bearing: 142, Route: "7",
		Trip: "652360", Latitude: 45.40333, Longitude: -71.88639, Occupancy: 1},
}
