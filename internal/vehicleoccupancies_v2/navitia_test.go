package vehicleoccupanciesv2

import (
	"reflect"
	"testing"
	"time"
)

func TestNewVehicleJourney(t *testing.T) {
	type args struct {
		vehicleJourneyCode string
		direction          int
		stopPoints         *[]StopPointVj
		createDate         time.Time
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
				vehicleJourneyCode: "KEOLIS:ServiceJourney:12888-C00048-3948471:LOC",
				direction:          1,
				stopPoints: &[]StopPointVj{{"stop_point:IDFM:28649", "7002", date},
					{"stop_point:STS:SP:169", "169", date}},
				createDate: createD,
			},
			want: &VehicleJourney{
				VehicleJourneyCode: "KEOLIS:ServiceJourney:12888-C00048-3948471:LOC",
				Direction:          1,
				StopPoints: &[]StopPointVj{{"stop_point:IDFM:28649", "7002", date},
					{"stop_point:STS:SP:169", "169", date}},
				CreateDate: createD,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewVehicleJourney(tt.args.vehicleJourneyCode, tt.args.direction,
				*tt.args.stopPoints, createD); !reflect.
				DeepEqual(got, tt.want) {
				t.Errorf("NewVehicleJourney() = %v, want %v", got, tt.want)
			}
		})
	}
}
