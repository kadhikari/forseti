package estimatedstoptimes

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(
		fmt.Sprintf(
			"file://%s/data_rennes/2022-11-25/real_time/%s",
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
	require.True(strings.HasPrefix(csvString, "ref_hor;hor;ref_apr;ref_crs;fiable\n"))
	// Check the last line of the extracted CSV
	require.True(strings.HasSuffix(csvString, "268543806;12:40:00;274432778;268441522;T\n"))
}

func TestParseEstimatedStopTimesFromFile(t *testing.T) {
	const EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES int = 6_584

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
			localTimeFirstLine:         time.Date(2012, time.February, 29, 11, 51, 1, 0, time.UTC),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 40, 0, 0, time.UTC),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 11, 51, 1, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 12, 40, 0, 0, time.UTC),
		},
		{ // Europe/Paris
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 11, 51, 1, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 40, 0, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 10, 51, 1, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 11, 40, 0, 0, time.UTC),
		},
		{ // America/New_York
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 11, 51, 1, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 12, 40, 0, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 16, 51, 1, 0, time.UTC),
			utcTimeLastLine:            time.Date(2012, time.February, 29, 17, 40, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localDailyServiceStartTime := test.localDailyServiceStartTime

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(
			fmt.Sprintf(
				"file://%s/data_rennes/2022-11-25/real_time/%s",
				fixtureDir,
				estimatedStopTimesFileName,
			),
		)
		require.Nil(err)

		fileBytes, err := ioutil.ReadFile(uri.Path)
		require.Nil(err)

		loadedEstimatedStopTimes, err := parseEstimatedStopTimesFromFile(
			fileBytes,
			&localDailyServiceStartTime,
		)
		assert.Nil(err)
		require.Len(loadedEstimatedStopTimes, EXPECTED_NUM_OF_ESTIMATED_STOP_TIMES)

		// Check the values read from the first line of the CSV
		{
			const EXPECTED_ID string = "268435464"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274137857"
			const EXPECTED_COURSE_ID CourseId = CourseId("268435460")
			const EXPECTED_FIABILITY Fiability = Fiability_T

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				EstimatedStopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeFirstLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
					CourseId:         EXPECTED_COURSE_ID,
					Fiability:        EXPECTED_FIABILITY,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeFirstLine)
			}
		}

		// Check the values read from the last line of the CSV
		{
			const EXPECTED_ID string = "268543806"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274432778"
			const EXPECTED_COURSE_ID CourseId = CourseId("268441522")
			const EXPECTED_FIABILITY Fiability = Fiability_T

			assert.Contains(loadedEstimatedStopTimes, EXPECTED_ID)
			assert.Equal(
				loadedEstimatedStopTimes[EXPECTED_ID],
				EstimatedStopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeLastLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
					CourseId:         EXPECTED_COURSE_ID,
					Fiability:        EXPECTED_FIABILITY,
				},
			)
			{
				utcLoadedTime := loadedEstimatedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				assert.Equal(utcLoadedTime, test.utcTimeLastLine)
			}
		}
	}
}

func TestEstimatedStopTimeFiabiliyFromSring(t *testing.T) {
	require := require.New(t)

	var tests = []struct {
		fiabilityStr      string
		expectedFiability Fiability
		errorIsExpected   bool
	}{
		{
			fiabilityStr:      "T",
			expectedFiability: Fiability_T,
			errorIsExpected:   false,
		},
		{
			fiabilityStr:      "",
			expectedFiability: Fiability_T,
			errorIsExpected:   false,
		},
		{
			fiabilityStr:      "F",
			expectedFiability: Fiability_F,
			errorIsExpected:   false,
		},
		{
			fiabilityStr:      "UNKNOWN",
			expectedFiability: Fiability_T,
			errorIsExpected:   true,
		},
	}

	for _, test := range tests {
		getFiability, err := parseFiabilityFromString(test.fiabilityStr)
		_ = getFiability
		if test.errorIsExpected {
			require.NotNil(err)
		} else {
			require.Nil(err)
			require.Equal(
				test.expectedFiability,
				getFiability,
			)
		}
	}
}

func TestNewEstimatedStopTime(t *testing.T) {
	require := require.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	var tests = []struct {
		record          string
		location        *time.Location
		errorIsExpected bool
		expectedResult  *EstimatedStopTime
	}{
		{ // With a field `fiable` = `T`
			record:          "268435464;11:51:01;274137857;268435460;T",
			location:        location,
			errorIsExpected: false,
			expectedResult: &EstimatedStopTime{
				Id: "268435464",
				Time: time.Date(
					0, time.January, 1,
					11, 51, 1, 0,
					location,
				),
				RouteStopPointId: "274137857",
				CourseId:         CourseId("268435460"),
				Fiability:        Fiability_T,
			},
		},
		{ // With a field `fiable` = `F`
			record:          "268435769;11:30:01;268502544;268435503;F",
			location:        location,
			errorIsExpected: false,
			expectedResult: &EstimatedStopTime{
				Id: "268435769",
				Time: time.Date(
					0, time.January, 1,
					11, 30, 1, 0,
					location,
				),
				RouteStopPointId: "268502544",
				CourseId:         CourseId("268435503"),
				Fiability:        Fiability_F,
			},
		},
		{ // With an empty field `fiable`
			record:          "2268440017;11:33:06;268536328;268435670;",
			location:        location,
			errorIsExpected: false,
			expectedResult: &EstimatedStopTime{
				Id: "2268440017",
				Time: time.Date(
					0, time.January, 1,
					11, 33, 6, 0,
					location,
				),
				RouteStopPointId: "268536328",
				CourseId:         CourseId("268435670"),
				Fiability:        Fiability_T,
			},
		},
		{ // With a missing field `fiable`
			record:          "2268440017;11:33:06;268536328;268435670",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
		{ // With too many fields
			record:          "268435769;11:30:01;268502544;268435503;F;",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
		{ // With too many fields
			record:          "268435769;11:30:01;268502544;268435503;F;xxx",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
	}

	for _, test := range tests {
		_ = test
		splittedRecord := strings.Split(test.record, ";")
		getResult, err := newEstimatedStopTime(
			splittedRecord,
			test.location,
		)
		if test.errorIsExpected {
			require.NotNil(err)
			require.Nil(getResult)
		} else {
			require.Nil(err)
			require.Equal(
				*(test.expectedResult),
				*getResult,
			)
		}
	}
}
