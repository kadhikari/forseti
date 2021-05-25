package vehiclelocations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_NewVehicleLocation(t *testing.T) {
	require := require.New(t)
	//assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)

	gtfsRtContext := &VehicleLocationsGtfsRtContext{}
	gtfsRtContext.globalContext = &VehicleLocationsContext{}

	vl := NewVehicleLocation(651969, "vehicle_journey:STS:651969-1", "263", date, 45.398613, -71.90111, 0, 0)
	require.NotNil(vl)
}
