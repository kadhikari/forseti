package vehiclepositions

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hove-io/forseti/google_transit"
	"github.com/stretchr/testify/require"
)

var fixtureDir string

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestExtractCsvStringFromJson(t *testing.T) {
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf(
		"file://%s/data_rennes/2022-09-08/real_time/%s",
		fixtureDir,
		vehiclePositionsFileName,
	))

	require.Nil(err)

	inputBytes, err := ioutil.ReadFile(uri.Path)
	require.Nil(err)
	_ = inputBytes

	csvString, err := extractCsvStringFromJson(inputBytes)
	require.Nil(err)
	require.NotEmpty(csvString)
	// // Check the first line (header) of the extracted CSV
	require.True(strings.HasPrefix(csvString, "idVehicule;no;Etat;idCourse;X;Y;Cap;latitude;longitude\n"))
	// Check the last line of the extracted CSV
	require.True(strings.HasSuffix(csvString, "268435472;16;INC;-1;0;0;0;-5.983863;-1.363087\n"))
}

func TestParseVehiclePositionsFromFileBytes(t *testing.T) {

	const EXPECTED_NUM_OF_VEHICLE_POSITIONS int = 474
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf(
		"file://%s/data_rennes/2022-09-08/real_time/%s",
		fixtureDir,
		vehiclePositionsFileName,
	))
	require.Nil(err)

	fileBytes, err := ioutil.ReadFile(uri.Path)
	require.Nil(err)

	gotVehiclePositions, err := parseVehiclePositionsFromFileBytes(
		fileBytes,
	)
	require.Nil(err)

	require.Len(gotVehiclePositions, EXPECTED_NUM_OF_VEHICLE_POSITIONS)

	// Check the first line
	{
		var EXPECTED_VEHICLE_POSITION = VehiclePosition{
			ExternalVehiculeId: ExternalVehiculeId("2078"),
			State:              HS,
			Heading:            236,
			Latitude:           48.147359,
			Longitude:          -1.705388,
			OccupancyStatus:    google_transit.VehiclePosition_NO_DATA_AVAILABLE,
		}

		require.Equal(
			EXPECTED_VEHICLE_POSITION,
			gotVehiclePositions[0],
		)
	}

	// Check the last line
	{
		var EXPECTED_VEHICLE_POSITION = VehiclePosition{
			ExternalVehiculeId: ExternalVehiculeId("16"),
			State:              INC,
			Heading:            0,
			Latitude:           -5.983863,
			Longitude:          -1.363087,
			OccupancyStatus:    google_transit.VehiclePosition_NO_DATA_AVAILABLE,
		}

		require.Equal(
			EXPECTED_VEHICLE_POSITION,
			gotVehiclePositions[EXPECTED_NUM_OF_VEHICLE_POSITIONS-1],
		)
	}

}
