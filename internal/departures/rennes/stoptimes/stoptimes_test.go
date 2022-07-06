package stoptimes

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

func TestLoadStopTimes(t *testing.T) {
	const EXPECTED_NUM_OF_STOP_TIMES int = 109_817

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	AMERICA_NEWYORK_LOCATION, _ := time.LoadLocation("America/New_York")

	var tests = []struct {
		localProcessingDate time.Time
		localTimeFirstLine  time.Time
		localTimeLastLine   time.Time
		utcTimeFirstLine    time.Time
		urcTimeLastLine     time.Time
	}{
		{ // UTC
			localProcessingDate: time.Date(2012, time.February, 29, 0, 0, 0, 0, time.UTC),
			localTimeFirstLine:  time.Date(2012, time.February, 29, 12, 56, 0, 0, time.UTC),
			localTimeLastLine:   time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 12, 56, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localProcessingDate: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:  time.Date(2012, time.February, 29, 12, 56, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:   time.Date(2012, time.February, 29, 4, 46, 1, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 11, 56, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 3, 46, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localProcessingDate: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:  time.Date(2012, time.February, 29, 12, 56, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:   time.Date(2012, time.February, 29, 4, 46, 1, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 17, 56, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 9, 46, 1, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localProcessingDate := test.localProcessingDate

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
		require.Nil(err)

		loadedStopTimes, err := LoadStopTimes(*uri, defaultTimeout, &localProcessingDate)
		require.Nil(err)
		assert.Len(loadedStopTimes, EXPECTED_NUM_OF_STOP_TIMES)

		// Check the values read from the first line of the CSV
		{
			const EXPECTED_ID string = "268548470"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274605064"

			assert.Contains(loadedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedStopTimes[EXPECTED_ID],
				StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeFirstLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
				},
			)
			{
				utcLoadedTime := loadedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeFirstLine)
			}

		}

		// Check the values read from the last line of the CSV
		{
			const EXPECTED_ID string = "268435458"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274137857"

			assert.Contains(loadedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedStopTimes[EXPECTED_ID],
				StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeLastLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
				},
			)
			{
				utcLoadedTime := loadedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.urcTimeLastLine)
			}
		}
	}
}
