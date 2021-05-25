package vehiclelocations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VehicleLocationFactory(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Create vehicle locations type of GTFS-RT
	vehiclelocations, err := VehicleLocationFactory("gtfs")
	require.Nil(err)
	_, ok := vehiclelocations.(*VehicleLocationsGtfsRtContext)
	require.True(ok)

	// Create vehicle locations type of unknown
	vehiclelocations, err = VehicleLocationFactory("unknown")
	require.NotNil(err)
	assert.EqualError(err, "Wrong vehiclelocation type passed")
	_, ok = vehiclelocations.(*VehicleLocationsGtfsRtContext)
	require.False(ok)
}
