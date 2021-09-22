package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/CanalTP/forseti/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestNewVehicleJourney(t *testing.T) {
	type args struct {
		vehicleId   string
		codesSource string
		line        string
		direction   string
		stopPoints  *[]StopPointVj
		createDate  time.Time
	}
	createD, _ := time.Parse("2006-01-02", "2021-05-01")
	location, _ := time.LoadLocation("Europe/Paris")
	date := time.Date(2020, 9, 21, 0, 0, 0, 0, location)
	tests := []struct {
		name string
		args args
		want *VehicleJourney
	}{
		{
			name: "NewVehicleJourney",
			args: args{
				vehicleId:   "stop_point:STS:SP:7002",
				codesSource: "7002",
				line:        "40",
				direction:   "inbound",
				stopPoints: &[]StopPointVj{{"stop_point:STS:SP:7002", "7002", date},
					{"stop_point:STS:SP:169", "169", date}},
				createDate: createD,
			},
			want: &VehicleJourney{
				VehicleID:   "stop_point:STS:SP:7002",
				CodesSource: "7002",
				Line:        "40",
				Direction:   "inbound",
				StopPoints: &[]StopPointVj{{"stop_point:STS:SP:7002", "7002", date},
					{"stop_point:STS:SP:169", "169", date}},
				CreateDate: createD,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVehicleJourney(tt.args.vehicleId, tt.args.codesSource, tt.args.line, tt.args.direction,
				*tt.args.stopPoints, createD); !reflect.
				DeepEqual(got, tt.want) {
				t.Errorf("NewVehicleJourney() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStopPointVj(t *testing.T) {
	type args struct {
		id   string
		code string
		date string
	}
	location, _ := time.LoadLocation("Europe/Paris")
	date := time.Date(2021, 9, 22, 0, 6, 0, 0, location)
	tests := []struct {
		name string
		args args
		want StopPointVj
	}{
		{
			name: "NewStopPointVj",
			args: args{id: "stop_point:STS:SP:7002", code: "7002", date: "06000"},
			want: StopPointVj{Id: "stop_point:STS:SP:7002", GtfsStopCode: "7002", DepartureTime: date},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStopPointVj(tt.args.id, tt.args.code, "060000"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStopPointVj() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateVehicleJourney(t *testing.T) {
	type args struct {
		navitiaVJ *NavitiaVehicleJourney
		line      string
	}
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicleJourneys.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)
	vj := &NavitiaVehicleJourney{}
	err = json.Unmarshal([]byte(jsonData), vj)
	require.Nil(err)

	createD, _ := time.Parse("2006-01-02", "2021-05-01")
	location, _ := time.LoadLocation("UTC")
	date := time.Date(2021, 05, 1, 0, 0, 0, 0, location)
	tests := []struct {
		name string
		args args
		want VehicleJourney
	}{
		{
			name: "CreateVehicleJourney",
			args: args{
				navitiaVJ: vj,
				line:      "40",
			},
			want: VehicleJourney{
				VehicleID:   "vehicle_journey:STS:652187-1",
				CodesSource: "652187",
				Line:        "40",
				Direction:   "",
				StopPoints: &[]StopPointVj{{"stop_point:STS:SP:7002", "7002", date},
					{"stop_point:STS:SP:169", "169", date}},
				CreateDate: createD,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateVehicleJourney(tt.args.navitiaVJ, tt.args.line, date); !reflect.DeepEqual(got[0], tt.want) {
				for _, vj := range got {
					t.Errorf("CreateVehicleJourney() = %v, want %v", vj, tt.want)
				}
			}
		})
	}
}
