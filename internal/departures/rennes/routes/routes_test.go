package routes

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/departures"
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

func TestLoadRoutes(t *testing.T) {

	const EXPECTED_NUM_OF_STOP_POINTS int = 383

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedRoutes, err := LoadRoutes(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedRoutes, EXPECTED_NUM_OF_STOP_POINTS)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_ROUTE_ID string = "284722688"
		const EXPECTED_LINE_INTERNAL_ID string = "248"
		const EXPECTED_DIRECTION departures.DirectionType = departures.DirectionTypeBackward
		const EXPECTED_DESTNATION_ID string = "284721153"

		assert.Contains(loadedRoutes, EXPECTED_ROUTE_ID)
		assert.Equal(
			loadedRoutes[EXPECTED_ROUTE_ID],
			Route{
				Id:             EXPECTED_ROUTE_ID,
				LineInternalId: EXPECTED_LINE_INTERNAL_ID,
				Direction:      EXPECTED_DIRECTION,
				DestinationId:  EXPECTED_DESTNATION_ID,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_ROUTE_ID string = "268502016"
		const EXPECTED_LINE_INTERNAL_ID string = "1"
		const EXPECTED_DIRECTION departures.DirectionType = departures.DirectionTypeForward
		const EXPECTED_DESTNATION_ID string = "268500995"

		assert.Contains(loadedRoutes, EXPECTED_ROUTE_ID)
		assert.Equal(
			loadedRoutes[EXPECTED_ROUTE_ID],
			Route{
				Id:             EXPECTED_ROUTE_ID,
				LineInternalId: EXPECTED_LINE_INTERNAL_ID,
				Direction:      EXPECTED_DIRECTION,
				DestinationId:  EXPECTED_DESTNATION_ID,
			},
		)
	}

}
