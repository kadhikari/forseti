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
	vehiclelocations, err := ConnectorFactory("gtfsrt")
	require.Nil(err)
	_, ok := vehiclelocations.(*GtfsRtContext)
	require.True(ok)

	// Create vehicle locations type of unknown
	vehiclelocations, err = ConnectorFactory("unknown")
	require.NotNil(err)
	assert.EqualError(err, "Wrong connector type passed")
	_, ok = vehiclelocations.(*GtfsRtContext)
	require.False(ok)
}
