package vehiclepositions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavitiaVehiclePositionsJsonDecoder(t *testing.T) {

	var tests = []struct {
		responseFileName                string
		expectedDecodedVehiclePositions NavitiaVehiclePositions
	}{
		{
			responseFileName: "response_vehicle_positions_OK.json",
			expectedDecodedVehiclePositions: NavitiaVehiclePositions{
				VehiclePositions: []NavitiaVehiclePosition{
					{
						Line: NavitiaLine{
							Id: "line:SAR:0100",
						},
						VehicleJourneyPositions: []NavitiaVehicleJourneyPosition{
							{
								VehicleJourney: NavitiaVehicleJourney{
									Id: "vehicle_journey:SAR:2222x051500_dst_2",
									Codes: []NavitiaCode{
										{
											Type:  "source",
											Value: "2222x051500",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			responseFileName: "response_vehicle_positions_KO.json",
			expectedDecodedVehiclePositions: NavitiaVehiclePositions{
				VehiclePositions: []NavitiaVehiclePosition{},
			},
		},
	}

	assert := assert.New(t)
	require := require.New(t)

	for _, test := range tests {

		responseFileName := test.responseFileName
		expectedDecodedVehiclePositions := test.expectedDecodedVehiclePositions

		uri, err := url.Parse(
			fmt.Sprintf(
				"file://%s/data_rennes/vehicle_positions/navitia/%s",
				fixtureDir,
				responseFileName,
			),
		)

		require.Nil(err)

		inputXmlFile, err := os.Open(uri.Path)
		require.Nil(err)

		decoder := json.NewDecoder(inputXmlFile)
		getDecodedVehiclePositions := NavitiaVehiclePositions{}
		err = decoder.Decode(&getDecodedVehiclePositions)
		require.Nil(err)
		_ = assert
		_ = expectedDecodedVehiclePositions
		assert.Equal(
			expectedDecodedVehiclePositions,
			getDecodedVehiclePositions,
		)
	}
}
