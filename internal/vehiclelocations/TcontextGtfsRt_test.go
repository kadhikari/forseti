package vehiclelocations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetVehicleLocations(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	date, err := time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)

	gtfsRtContext := &VehicleLocationsGtfsRtContext{}
	gtfsRtContext.globalContext = &VehicleLocationsContext{}

	// Call Api without vehicleLocations in the data
	param := VehicleLocationRequestParameter{VehicleJourneyId: "vehicle_journey:STS:651969-1", Date: date}
	_, err = gtfsRtContext.GetVehicleLocations(&param)
	require.NotNil(err)
	assert.EqualError(err, "no vehicle_occupancies in the data")

	gtfsRtContext.globalContext.VehicleLocations = map[int]*VehicleLocation{
		651969: {651969, "vehicle_journey:STS:651969-1", "263", date, 45.398613, -71.90111, 0, 0}}

	// Call Api with vehicle_journey_id
	vehicleLocations, err := gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 1)

	// Call Api with date
	date, err = time.ParseInLocation("2006-01-02", "2021-05-25", location)
	require.Nil(err)
	param = VehicleLocationRequestParameter{VehicleJourneyId: "", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 1)

	// Call Api with date before after date of vehicleLocation
	date, err = time.ParseInLocation("2006-01-02", "2021-05-26", location)
	require.Nil(err)
	param = VehicleLocationRequestParameter{VehicleJourneyId: "", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 0)

	// Call Api with no existing vehicle_journey_id
	gtfsRtContext.globalContext.VehicleLocations = map[int]*VehicleLocation{
		651969: {651969, "vehicle_journey:STS:651970-1", "263", date, 45.398613, -71.90111, 0, 0}}
	param = VehicleLocationRequestParameter{VehicleJourneyId: "vehicle_journey:STS:651969-1", Date: date}
	vehicleLocations, err = gtfsRtContext.GetVehicleLocations(&param)
	require.Nil(err)
	assert.Equal(len(vehicleLocations), 0)

}
