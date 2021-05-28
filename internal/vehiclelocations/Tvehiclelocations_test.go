package vehiclelocations

import (
	"testing"
	"time"

	nav "github.com/CanalTP/forseti/navitia"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewVehicleLocation(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)

	vl, err := NewVehicleLocation(651969, "vehicle_journey:STS:651969-1", date, 45.398613, -71.90111, 0, 0)
	assert.Nil(err)
	require.NotNil(vl)
}

func Test_UpdateVehicleLocation(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-05-26", location)
	require.Nil(err)

	vehicleLocations := VehicleLocations{}

	vj := nav.VehicleJourney{VehicleID: "vehicle_journey:STS:651970-1",
		CodesSource: "651970",
		StopPoints: &[]nav.StopPointVj{nav.NewStopPointVj("stop_point:STS:SP:1280", "1280"),
			nav.NewStopPointVj("stop_point:STS:SP:1560", "1561")},
		CreateDate: date}
	vGtfsRt := VehicleGtfsRt{VehicleID: "52103", StopId: "1280", Label: "52103", Time: 1621900800,
		Speed: 0, Bearing: 0, Route: "1", Trip: "651970", Latitude: 45.9999, Longitude: -71.90111, Occupancy: 0}
	changeGtfsRt := VehicleGtfsRt{VehicleID: "52103", StopId: "1280", Label: "52103", Time: 1621900800,
		Speed: 11, Bearing: 254, Route: "1", Trip: "651970", Latitude: 46.0000, Longitude: -72.0000, Occupancy: 0}

	// Update vehicleLocation with no map cehiclelocations
	vehicleLocations.UpdateVehicleLocation(changeGtfsRt, location)
	require.Nil(vehicleLocations.vehicleLocations)

	// Create vehicleLocation from existing data
	vl := createVehicleLocationFromDataSource(vj, vGtfsRt, location)
	t.Log("DATE: ", vl.DateTime)
	vehicleLocations.AddVehicleLocation(vl)
	require.NotNil(vehicleLocations.vehicleLocations)
	assert.Equal(len(vehicleLocations.vehicleLocations), 1)

	// Update vehicleLocation with existing data
	vehicleLocations.UpdateVehicleLocation(changeGtfsRt, location)
	assert.Equal(vehicleLocations.vehicleLocations[651970].Latitude, float32(46.0000))
	assert.Equal(vehicleLocations.vehicleLocations[651970].Longitude, float32(-72.0000))
	assert.Equal(vehicleLocations.vehicleLocations[651970].Bearing, float32(254))
	assert.Equal(vehicleLocations.vehicleLocations[651970].Speed, float32(11))
}
