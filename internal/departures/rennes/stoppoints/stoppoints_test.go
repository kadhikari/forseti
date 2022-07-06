package stoppoints

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

func TestLoadStopPoints(t *testing.T) {

	const EXPECTED_NUM_OF_STOP_POINTS int = 1_584

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedStopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedStopPoints, EXPECTED_NUM_OF_STOP_POINTS)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_INTERNAL_ID string = "10021"
		const EXPECTED_EXTERNAL_ID string = "10021"

		assert.Contains(loadedStopPoints, EXPECTED_EXTERNAL_ID)
		assert.Equal(
			loadedStopPoints[EXPECTED_EXTERNAL_ID],
			StopPoint{
				InternalId: EXPECTED_INTERNAL_ID,
				ExternalId: EXPECTED_EXTERNAL_ID,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_INTERNAL_ID string = "1001"
		const EXPECTED_EXTERNAL_ID string = "1001"

		assert.Contains(loadedStopPoints, EXPECTED_EXTERNAL_ID)
		assert.Equal(
			loadedStopPoints[EXPECTED_EXTERNAL_ID],
			StopPoint{
				InternalId: EXPECTED_INTERNAL_ID,
				ExternalId: EXPECTED_EXTERNAL_ID,
			},
		)
	}

}
