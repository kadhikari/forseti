package estimatedstoptimes

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/departures/rennes/stoptimes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtureDir string

const defaultTimeout time.Duration = time.Second * 10

type CourseId = stoptimes.CourseId

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestExtractCsvStringFromJson(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(
		fmt.Sprintf(
			"file://%s/data_rennes/2022-09-08/real_time/%s",
			fixtureDir,
			estimatedStopTimesFileName,
		),
	)
	assert.Nil(err)

	inputBytes, err := ioutil.ReadFile(uri.Path)
	assert.Nil(err)

	csvString, err := extractCsvStringFromJson(inputBytes)
	assert.Nil(err)
	assert.NotEmpty(csvString)
	// Check the first line (header) of the extracted CSV
	require.True(strings.HasPrefix(csvString, "ref_hor;hor;ref_apr;ref_crs\n"))
	// Check the last line of the extracted CSV
	// require.True(strings.HasSuffix(csvString, "268547687;12:26:00;274432770;268441480\n"))
}

func TestParseEstimatedStopTimesFromFile(t *testing.T) {
	const EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES int = 6_783

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	AMERICA_NEWYORK_LOCATION, _ := time.LoadLocation("America/New_York")

	var tests = []struct {
		localDailyServiceStartTime time.Time
		localTimeFirstLine         time.Time
		localTimeLastLine          time.Time
		utcTimeFirstLine           time.Time
		utcTimeLastLine            time.Time
	}{
		{ // UTC
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, time.UTC),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 26, 0, 0, time.UTC),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 1, 1, 0, time.UTC),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 12, 26, 0, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 12, 1, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 26, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 1, 1, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 11, 26, 0, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 11, 1, 1, 0, time.UTC),
		},
		{ // America/New_York
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 26, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 1, 1, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 17, 26, 0, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 17, 1, 1, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localDailyServiceStartTime := test.localDailyServiceStartTime

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(
			fmt.Sprintf(
				"file://%s/data_rennes/2022-09-08/real_time/%s",
				fixtureDir,
				estimatedStopTimesFileName,
			),
		)
		require.Nil(err)

		fileBytes, err := ioutil.ReadFile(uri.Path)
		require.Nil(err)

		loadedEstimatedStopTimes, err := parseEstimatedStopTimesFromFile(
			fileBytes,
			defaultTimeout,
			&localDailyServiceStartTime,
		)
		assert.Nil(err)
		require.Len(loadedEstimatedStopTimes, EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES)

		// Check the values read from the first line of the CSV
		{
			const EXPECTED_ID string = "268547687"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274432770"
			const EXPECTED_COURSE_ID CourseId = CourseId("268441480")

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				stoptimes.StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeFirstLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
					CourseId:         EXPECTED_COURSE_ID,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeFirstLine)
			}
		}

		// Check the values read from the last line of the CSV
		{
			const EXPECTED_ID string = "268435464"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274137857"
			const EXPECTED_COURSE_ID CourseId = CourseId("268435460")

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				stoptimes.StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeLastLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
					CourseId:         EXPECTED_COURSE_ID,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeLastLine)
			}
		}
	}
}
