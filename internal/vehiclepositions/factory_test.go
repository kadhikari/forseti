package vehiclepositions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VehiclePositionFactory(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Create vehicle locations type of GTFS-RT
	vehiclepositions, err := ConnectorFactory("gtfsrt")
	require.Nil(err)
	_, ok := vehiclepositions.(*GtfsRtContext)
	require.True(ok)

	// Create vehicle locations type of unknown
	vehiclepositions, err = ConnectorFactory("unknown")
	require.NotNil(err)
	assert.EqualError(err, "Wrong connector type passed")
	_, ok = vehiclepositions.(*GtfsRtContext)
	require.False(ok)
}
