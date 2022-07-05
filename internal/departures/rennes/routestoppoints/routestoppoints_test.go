package routestoppoints

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtureDir string

const defaultTimeout time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestLoadRouteStopPoints(t *testing.T) {

	const EXPECTED_NUM_OF_ROUTE_STOP_POINTS int = 12_508

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedRoutesStopPoints, err := LoadRouteStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedRoutesStopPoints, EXPECTED_NUM_OF_ROUTE_STOP_POINTS)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_ID string = "284722693"
		const EXPECTED_STOP_POINT_ID string = "1412"
		const EXPECTED_ROUTES_ID string = "284722688"
		const EXPECTED_STOP_POINT_ORDER int = 4251

		assert.Contains(loadedRoutesStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedRoutesStopPoints[EXPECTED_ID],
			RouteStopPoint{
				Id:             EXPECTED_ID,
				StopPointId:    EXPECTED_STOP_POINT_ID,
				RouteId:        EXPECTED_ROUTES_ID,
				StopPointOrder: EXPECTED_STOP_POINT_ORDER,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_ID string = "268501249"
		const EXPECTED_STOP_POINT_ID string = "1004"
		const EXPECTED_ROUTES_ID string = "268501248"
		const EXPECTED_STOP_POINT_ORDER int = 0

		assert.Contains(loadedRoutesStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedRoutesStopPoints[EXPECTED_ID],
			RouteStopPoint{
				Id:             EXPECTED_ID,
				StopPointId:    EXPECTED_STOP_POINT_ID,
				RouteId:        EXPECTED_ROUTES_ID,
				StopPointOrder: EXPECTED_STOP_POINT_ORDER,
			},
		)
	}

}
