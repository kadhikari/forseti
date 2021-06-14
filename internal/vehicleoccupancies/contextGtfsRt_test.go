package vehicleoccupancies

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CheckLastLoadChanged(t *testing.T) {
	assert := assert.New(t)

	gtfsRtContext := &VehicleOccupanciesGtfsRtContext{}
	gtfsRtContext.lastLoadNavitia = "20210511T090912.579677"
	changed := gtfsRtContext.CheckLastLoadChanged("20210511T090912.579677")
	assert.Equal(changed, false)

	changed = gtfsRtContext.CheckLastLoadChanged("20210515T000000.000000")
	assert.Equal(changed, true)
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
	gtfsRtContext.CleanListVehicleOccupancies()
	assert.Equal(len(voContext.VehicleOccupancies), 0)
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
		Id:               200,
		LineCode:         "40",
		VehicleJourneyId: "vehicle_journey:0:124695149-1",
		StopId:           "stop_point:0:SP:80:4029",
		Direction:        0,
		DateTime:         time.Now(),
		Occupancy:        1})
	require.NotNil(gtfsRtContext.voContext.VehicleOccupancies)
	assert.Equal(len(gtfsRtContext.voContext.VehicleOccupancies), 1)
}

func Test_CleanListVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	gtfsRtContext.vehiclesJourney = mapVJ
	require.NotNil(gtfsRtContext.vehiclesJourney)

	gtfsRtContext.CleanListVehicleJourney()
	require.Nil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 0)
}

func Test_CleanListOldVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	gtfsRtContext.vehiclesJourney = mapVJ
	require.NotNil(gtfsRtContext.vehiclesJourney)

	gtfsRtContext.CleanListOldVehicleJourney(1)
	require.NotNil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 1)
}

func Test_AddVehicleJourney(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	gtfsRtContext.AddVehicleJourney(&VehicleJourney{VehicleID: "vehicle_journey:STS:651969-1",
		CodesSource: "651969",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			NewStopPointVj("stop_point:STS:SP:1560", "1560")},
		CreateDate: time.Now()})
	require.NotNil(gtfsRtContext.vehiclesJourney)
	assert.Equal(len(gtfsRtContext.vehiclesJourney), 1)
}

func Test_GetVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	dateVj, _ := time.Parse("2006-01-02", "2021-05-12")
	require.Nil(err)

	vehicleOccupanciesContext, err := VehicleOccupancyFactory(string(connectors.Connector_GRFS_RT))
	require.Nil(err)
	gtfsRtContext, ok := vehicleOccupanciesContext.(*VehicleOccupanciesGtfsRtContext)
	require.True(ok)
	voContext := gtfsRtContext.GetVehicleOccupanciesContext()
	require.NotNil(voContext)

	vj := VehicleJourney{VehicleID: "vehicle_journey:STS:651969-1",
		CodesSource: "652517",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:263", "263"),
			NewStopPointVj("stop_point:STS:SP:1560", "1560")},
		CreateDate: dateVj}

	vGtfsRt := VehicleGtfsRt{"52103", "263", "52103", 1620777600, 11, 274, "1", "652517", 45.398613, -71.90111, 0}

	// Create VehicleOccupancies from existing data
	vo := createOccupanciesFromDataSource(vj, vGtfsRt, location)
	gtfsRtContext.AddVehicleOccupancy(vo)
	require.NotNil(voContext.VehicleOccupancies)
	assert.Equal(len(voContext.VehicleOccupancies), 1)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{StopId: "", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err := gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Call Api with StopId in the paraameter
	param = VehicleOccupancyRequestParameter{
		StopId:           "stop_point:STS:SP:263",
		VehicleJourneyId: "",
		Date:             date}
	vehicleOccupancies, err = gtfsRtContext.GetVehicleOccupancies(&param)
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 1)

	// Call Api when vehicleOcupancies list is nil
	param = VehicleOccupancyRequestParameter{
		StopId:           "stop_point:STS:SP:263",
		VehicleJourneyId: "",
		Date:             date}
	voContext.VehicleOccupancies = nil
	_, err = gtfsRtContext.GetVehicleOccupancies(&param)
	assert.Error(err, "no vehicle_occupancies in the data")
}

func Test_parseVehiclesResponse(t *testing.T) {
	type args struct {
		b []byte
	}

	require := require.New(t)
	f, err := os.Open(fmt.Sprintf("%s/vehiclePositions.pb", fixtureDir))
	require.Nil(err)
	reader, err := ioutil.ReadAll(f)
	require.Nil(err)

	tests := []struct {
		name    string
		args    args
		want    *GtfsRt
		wantErr bool
	}{
		{
			name: "ParseVehiclesResponseGtfs-rt_OK",
			args: args{
				b: reader,
			},
			want: &GtfsRt{
				Timestamp: "1620076153",
				Vehicles:  dataGtfsRt,
			},
			wantErr: false,
		},
		{
			name: "ParseVehiclesResponseGtfs-rt_KO",
			args: args{
				b: []byte{},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVehiclesResponse(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVehiclesResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVehiclesResponse() = %v, want %v", got, tt.want)
			}
		})
	}
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
		CreateDate: time.Now().Add(-2 * time.Hour)},
	"652373": {VehicleID: "vehicle_journey:STS:652373-1",
		CodesSource: "652373",
		StopPoints: &[]StopPointVj{NewStopPointVj("stop_point:STS:SP:814", "814"),
			NewStopPointVj("stop_point:STS:SP:900", "900")},
		CreateDate: time.Now()},
}

var vehicleOccupanciesMap = map[int]*VehicleOccupancy{
	156: {
		Id:               156,
		LineCode:         "40",
		VehicleJourneyId: "vehicle_journey:0:124695149-1",
		StopId:           "stop_point:0:SP:80:4029",
		Direction:        0,
		DateTime:         time.Now(),
		Occupancy:        1},
	700: {
		Id:               700,
		LineCode:         "45",
		VehicleJourneyId: "vehicle_journey:0:124695000-1",
		StopId:           "stop_point:0:SP:80:4043",
		Direction:        0,
		DateTime:         time.Now(),
		Occupancy:        2},
}

var dataGtfsRt = []VehicleGtfsRt{
	{"52103", "263", "52103", 1620076139, 11, 274, "1", "652517", 45.398613, -71.90111, 0},
	{"52105", "47", "52105", 1620076116, 12, 268, "12", "652604", 45.40917, -71.930275, 0},
	{"53101", "1326", "53101", 1620076034, 1, 262, "17", "653036", 45.40333, -71.95417, 0},
	{"53103", "185", "53103", 1620076121, 11, 0, "16", "652712", 45.421112, -71.91722, 0},
	{"53105", "171", "53105", 1620076127, 0, 60, "11", "652883", 45.3825, -71.9275, 0},
	{"54103", "800", "54103", 1620076143, 14, 41, "7", "652543", 45.42389, -71.8725, 0},
	{"55103", "1292", "55103", 1620076107, 13, 126, "12", "652530", 45.41028, -71.93445, 0},
	{"55104", "2454", "55104", 1620076084, 0, 94, "17", "652731", 45.34861, -71.99167, 0},
	{"57101", "7005", "57101", 1620076006, 0, 136, "6", "652817", 45.378334, -71.92917, 0},
	{"57103", "7000", "57103", 1620076012, 0, 10, "9", "652787", 45.378613, -71.92917, 0},
	{"58102", "2019", "58102", 1620076132, 0, 304, "2", "652740", 45.405834, -71.886665, 1},
	{"59102", "330", "59102", 1620076143, 14, 60, "3", "652984", 45.39722, -71.94334, 1},
	{"59103", "234", "59103", 1620076030, 9, 126, "8", "652772", 45.45, -71.865555, 1},
	{"60102", "242", "60102", 1620076094, 0, 184, "8", "652830", 45.422222, -71.87417, 1},
	{"60107", "581", "60107", 1620076136, 13, 48, "9", "652670", 45.388054, -71.90833, 0},
	{"60108", "7002", "60108", 1620075990, 0, 18, "8", "652659", 45.378613, -71.92944, 1},
	{"61103", "993", "61103", 1620076006, 18.9, 174, "11", "653112", 45.40278, -71.96111, 1},
	{"61104", "7004", "61104", 1620076092, 0, 16, "16", "653028", 45.378613, -71.929726, 0},
	{"62101", "6003", "62101", 1620075988, 0, 264, "7", "652905", 45.411667, -71.884445, 0},
	{"64104", "270", "64104", 1620076130, 8, 260, "3", "653044", 45.398335, -71.92389, 1},
	{"65101", "588", "65101", 1620076103, 0, 4, "8", "652940", 45.394444, -71.89611, 1},
	{"65105", "415", "65105", 1620076103, 0, 62, "3", "652959", 45.411945, -71.87139, 1},
	{"65106", "9991", "65106", 1620076130, 0, 278, "Sortie", "652701", 45.391666, -71.928055, 1},
	{"65107", "699", "65107", 1620076125, 0, 90, "14", "652890", 45.38389, -71.88695, 1},
	{"65109", "2038", "65109", 1620076111, 13, 272, "4", "653126", 45.40278, -71.825836, 1},
	{"65111", "214", "65111", 1620076114, 0, 0, "8", "653283", 45.40361, -71.87334, 1},
	{"66101", "370", "66101", 1620076130, 12, 268, "4", "652647", 45.40111, -71.86889, 1},
	{"66103", "2125", "66103", 1620076130, 0, 0, "11", "653058", 45.413612, -71.94444, 1},
	{"67101", "7002", "67101", 1620075993, 1.1, 300, "14", "652913", 45.378887, -71.92944, 1},
	{"67102", "524", "67102", 1620076141, 0, 184, "7", "652845", 45.386112, -71.90139, 1},
	{"67104", "436", "67104", 1620076135, 9, 178, "5", "653230", 45.4275, -71.88722, 1},
	{"67106", "2082", "67106", 1620076101, 0, 0, "7", "652780", 45.378887, -71.895, 1},
	{"68101", "1721", "68101", 1620076133, 0, 0, "18", "653247", 45.384445, -71.975555, 1},
	{"68105", "297", "68105", 1620076117, 0, 0, "3", "653023", 45.4025, -71.96333, 0},
	{"69101", "8002", "69101", 1620076107, 0, 0, "12", "653164", 45.40361, -71.949165, 1},
	{"69104", "2295", "69104", 1620076076, 8, 178, "9", "653012", 45.41028, -71.851944, 1},
	{"69105", "1134", "69105", 1620076055, 12, 178, "5", "653265", 45.418888, -71.882225, 0},
	{"70101", "378", "70101", 1620076112, 0, 262, "17", "653197", 45.40361, -71.91333, 1},
	{"70102", "2273", "70102", 1620076050, 3, 226, "3", "652564", 45.421944, -71.87222, 1},
	{"70104", "1137", "70104", 1620076133, 0, 176, "14", "652998", 45.39111, -71.89056, 1},
	{"70105", "2111", "70105", 1620076142, 0, 246, "9", "652858", 45.410557, -71.84055, 1},
	{"53104", "6000", "53104", 1620076057, 0, 90, "Entree", "653145", 45.41179, -71.88419, 0},
	{"56104", "2162", "56104", 1620076004, 8, 178, "19", "652934", 45.384125, -71.944145, 0},
	{"58104", "8000", "58104", 1620075972, 0, 0, "Entree", "652969", 45.402363, -71.95269, 1},
	{"58101", "234", "58101", 1620075782, 5, 216, "8", "653213", 45.44625, -71.869576, 1},
	{"52101", "6000", "52101", 1620075746, 9, 132, "Entree", "652467", 45.411736, -71.884796, 0},
	{"67103", "234", "67103", 1620075508, 0, 218, "7", "652918", 45.44625, -71.869576, 3},
	{"70103", "7005", "70103", 1620076110, 13, 200, "6", "652255", 45.385, -71.924446, 1},
	{"62102", "6001", "62102", 1620075961, 0, 173, "12", "652221", 45.408054, -71.88333, 1},
	{"64102", "5100", "64102", 1620075922, 5, 52, "8", "652205", 45.40167, -71.889725, 1},
	{"66105", "7004", "66105", 1620075859, 0, 16, "18", "653078", 45.378056, -71.92917, 1},
	{"69103", "5200", "69103", 1620075320, 6, 142, "7", "652360", 45.40333, -71.88639, 1},
}
