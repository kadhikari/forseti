package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"testing"

	"github.com/CanalTP/forseti/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestNewVehicleJourney(t *testing.T) {
	type args struct {
		vehicleId   string
		codesSource string
		stopPoints  []StopPointVj
	}
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
				stopPoints: []StopPointVj{{"stop_point:STS:SP:7002", "7002"},
					{"stop_point:STS:SP:169", "169"}},
			},
			want: &VehicleJourney{
				VehicleID:   "stop_point:STS:SP:7002",
				CodesSource: "7002",
				StopPoints: &[]StopPointVj{{"stop_point:STS:SP:7002", "7002"},
					{"stop_point:STS:SP:169", "169"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVehicleJourney(tt.args.vehicleId, tt.args.codesSource, tt.args.stopPoints); !reflect.
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
	}
	tests := []struct {
		name string
		args args
		want StopPointVj
	}{
		{
			name: "NewStopPointVj",
			args: args{id: "stop_point:STS:SP:7002", code: "7002"},
			want: StopPointVj{Id: "stop_point:STS:SP:7002", GtfsStopCode: "7002"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStopPointVj(tt.args.id, tt.args.code); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStopPointVj() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateVehicleJourney(t *testing.T) {
	type args struct {
		navitiaVJ *NavitiaVehicleJourney
		id_gtfsRt string
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

	tests := []struct {
		name string
		args args
		want *VehicleJourney
	}{
		{
			name: "CreateVehicleJourney",
			args: args{
				navitiaVJ: vj,
				id_gtfsRt: "652187",
			},
			want: &VehicleJourney{
				VehicleID:   "vehicle_journey:STS:652187-1",
				CodesSource: "652187",
				StopPoints: &[]StopPointVj{{"stop_point:STS:SP:7002", "7002"},
					{"stop_point:STS:SP:169", "169"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateVehicleJourney(tt.args.navitiaVJ, tt.args.id_gtfsRt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateVehicleJourney() = %v, want %v", got.StopPoints, tt.want.StopPoints)
			}
		})
	}
}
