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

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestExtractCsvStringFromJson(t *testing.T) {
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/webservice/%s", fixtureDir, estimatedStopTimesFileName))
	require.Nil(err)

	inputBytes, err := ioutil.ReadFile(uri.Path)
	require.Nil(err)

	csvString, err := extractCsvStringFromJson(inputBytes)
	require.Nil(err)
	require.NotEmpty(csvString)
	// Check the first line (header) of the extracted CSV
	require.True(strings.HasPrefix(csvString, "ref_hor;hor;ref_apr;ref_crs\n"))
	// Check the last line of the extracted CSV
	require.True(strings.HasSuffix(csvString, "268435927;15:30:24;268534785;268435509\n"))
}

func TestParseEstimatedStopTimesFromFile(t *testing.T) {
	const EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES int = 8_777

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
			localTimeFirstLine:  time.Date(2012, time.February, 29, 16, 36, 0, 0, time.UTC),
			localTimeLastLine:   time.Date(2012, time.February, 29, 15, 30, 24, 0, time.UTC),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 16, 36, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 15, 30, 24, 0, time.UTC),
		},
		{ // Europe/Paris
			localProcessingDate: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:  time.Date(2012, time.February, 29, 16, 36, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:   time.Date(2012, time.February, 29, 15, 30, 24, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 15, 36, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 14, 30, 24, 0, time.UTC),
		},
		{ // America/New_York
			localProcessingDate: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:  time.Date(2012, time.February, 29, 16, 36, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:   time.Date(2012, time.February, 29, 15, 30, 24, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:    time.Date(2012, time.February, 29, 21, 36, 0, 0, time.UTC),
			urcTimeLastLine:     time.Date(2012, time.February, 29, 20, 30, 24, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localProcessingDate := test.localProcessingDate

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/webservice/%s", fixtureDir, estimatedStopTimesFileName))
		require.Nil(err)

		fileBytes, err := ioutil.ReadFile(uri.Path)
		require.Nil(err)

		loadedEstimatedStopTimes, err := parseEstimatedStopTimesFromFile(
			fileBytes,
			defaultTimeout,
			&localProcessingDate,
		)
		require.Nil(err)
		assert.Len(loadedEstimatedStopTimes, EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES)

		// Check the values read from the first line of the CSV
		{
			const EXPECTED_ID string = "268546728"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274605070"

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				stoptimes.StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeFirstLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeFirstLine)
			}
		}

		// Check the values read from the last line of the CSV
		{
			const EXPECTED_ID string = "268435927"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "268534785"

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				stoptimes.StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeLastLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.urcTimeLastLine)
			}
		}
	}
}
