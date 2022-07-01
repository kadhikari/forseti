package scheduledtimes

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

func TestLoadScheduledStopPoints(t *testing.T) {
	const EXPECTED_NUM_OF_SCHEDULED_STOP_POINTS int = 109_817

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedScheduledStopPoints, err := LoadScheduledTimes(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedScheduledStopPoints, EXPECTED_NUM_OF_SCHEDULED_STOP_POINTS)

	loc, _ := time.LoadLocation("Europe/Paris")
	// Check the values read from the first line of the CSV
	{
		const EXPECTED_ID string = "268548470"
		var EXPECTED_TIME time.Time = time.Date(0, 1, 1, 12, 56, 0, 0, loc)
		const EXPECTED_DB_INTERNAL_LINK_ID string = "274605064"

		assert.Contains(loadedScheduledStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedScheduledStopPoints[EXPECTED_ID],
			ScheduledTime{
				Id:               EXPECTED_ID,
				Time:             EXPECTED_TIME,
				DbInternalLinkId: EXPECTED_DB_INTERNAL_LINK_ID,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_ID string = "268435458"
		var EXPECTED_TIME time.Time = time.Date(0, 1, 1, 4, 46, 1, 0, loc)
		const EXPECTED_DB_INTERNAL_LINK_ID string = "274137857"

		assert.Contains(loadedScheduledStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedScheduledStopPoints[EXPECTED_ID],
			ScheduledTime{
				Id:               EXPECTED_ID,
				Time:             EXPECTED_TIME,
				DbInternalLinkId: EXPECTED_DB_INTERNAL_LINK_ID,
			},
		)
	}

}
