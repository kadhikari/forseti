package stoptimes

import (
	"fmt"
	"net/url"
	"os"
	"strings"
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
	const EXPECTED_NUM_OF_STOP_TIMES int = 109_559

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	AMERICA_NEWYORK_LOCATION, _ := time.LoadLocation("America/New_York")

	var tests = []struct {
		localDailyServiceStartTime time.Time
		localTimeFirstLine         time.Time
		localTimeLastLine          time.Time
		utcTimeFirstLine           time.Time
		urcTimeLastLine            time.Time
	}{
		{ // UTC
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, time.UTC),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 17, 20, 0, 0, time.UTC),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 17, 20, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 17, 20, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 16, 20, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 3, 46, 1, 0, time.UTC),
		},
		{ // America/New_York
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 17, 20, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 22, 20, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 9, 46, 1, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localDailyServiceStartTime := test.localDailyServiceStartTime

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/2022-09-08/base_scheduled", fixtureDir))
		assert.Nil(err)

		loadedStopTimes, err := LoadStopTimes(*uri, defaultTimeout, &localDailyServiceStartTime)
		require.Nil(err)
		assert.Len(loadedStopTimes, EXPECTED_NUM_OF_STOP_TIMES)

		// Check the values read from the first line of the CSV
		{
			const EXPECTED_ID string = "268548156"
			const EXPECTED_ROUTE_STOP_POINT_ID string = "274435844"
			const EXPECTED_COURSE_ID CourseId = CourseId("268441507")

			require.Contains(loadedStopTimes, EXPECTED_ID)
			require.Equal(
				loadedStopTimes[EXPECTED_ID],
				StopTime{
					Id:               EXPECTED_ID,
					Time:             test.localTimeFirstLine,
					RouteStopPointId: EXPECTED_ROUTE_STOP_POINT_ID,
					CourseId:         EXPECTED_COURSE_ID,
				},
			)
			{
				utcLoadedTime := loadedStopTimes[EXPECTED_ID].Time.In(time.UTC)
				require.Equal(utcLoadedTime, test.utcTimeFirstLine)
			}

		}

	}
}

func TestCombineDateAndTimeInLocation(t *testing.T) {

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")

	var tests = []struct {
		name            string
		datePart        time.Time
		timePart        time.Time
		location        *time.Location
		errorIsExpected bool
		expectedResult  time.Time
	}{
		{
			name:            "test#1",
			datePart:        time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC),
			timePart:        time.Date(0, 1, 1, 12, 58, 59, 999_999_999, time.UTC),
			location:        time.UTC,
			errorIsExpected: false,
			expectedResult:  time.Date(2022, time.February, 1, 12, 58, 59, 999_999_999, time.UTC),
		},
		{
			name:            "test#2",
			datePart:        time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC),
			timePart:        time.Date(0, 1, 1, 12, 58, 59, 999_999_999, time.UTC),
			location:        EUROPE_PARIS_LOCATION,
			errorIsExpected: false,
			expectedResult:  time.Date(2022, time.February, 1, 12, 58, 59, 999_999_999, EUROPE_PARIS_LOCATION),
		},
	}

	_ = tests

	assert := assert.New(t)
	for _, test := range tests {
		errMessage := fmt.Sprintf("the unit test '%s' has failed", test.name)
		getResult, gotError := CombineDateAndTimeInLocation(
			&test.datePart,
			&test.timePart,
			test.location)
		assert.Equal(gotError != nil, test.errorIsExpected, errMessage)
		if gotError != nil {
			assert.Equal(getResult, test.expectedResult, errMessage)
		}
	}
}

func TestComputeActualStopTime(t *testing.T) {
	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	SERVICE_START_TIME_AT_MIDNIGHT := time.Date(2022, time.June, 15, 0, 0, 0, 0, EUROPE_PARIS_LOCATION)
	SERVICE_START_TIME_AT_MIDDAY := time.Date(2022, time.June, 15, 12, 0, 0, 0, EUROPE_PARIS_LOCATION)

	var tests = []struct {
		name                   string
		stopTime               time.Time
		dailyServiceStartTime  time.Time
		expectedActualStopTime time.Time
	}{
		{
			name:                   "0 nanosecond after service start time (midday)",
			stopTime:               time.Date(0, time.January, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 15, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{
			name:                   "1 nanosecond after service start time (midday)",
			stopTime:               time.Date(0, time.January, 1, 12, 0, 0, 1, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 15, 12, 0, 0, 1, EUROPE_PARIS_LOCATION),
		},
		{
			name:                   "1 nanosecond before service start time (midday)",
			stopTime:               time.Date(0, time.January, 1, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 16, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
		},
		{
			name:                   "0 nanosecond after service start time (midnight)",
			stopTime:               time.Date(0, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
			expectedActualStopTime: time.Date(2022, time.June, 15, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{
			name:                   "1 nanosecond after service start time (midnight)",
			stopTime:               time.Date(0, time.January, 2, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
			expectedActualStopTime: time.Date(2022, time.June, 15, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
		},
		{
			name:                   "1 nanosecond before service start time (midnight)",
			stopTime:               time.Date(0, time.January, 2, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
			expectedActualStopTime: time.Date(2022, time.June, 15, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
		},
	}
	assert := assert.New(t)
	for _, test := range tests {
		errMessage := fmt.Sprintf("the unit test '%s' failed", test.name)
		getActualStopTime := ComputeActualStopTime(test.stopTime, &test.dailyServiceStartTime)
		assert.Equal(test.expectedActualStopTime, getActualStopTime, errMessage)
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
		expectedResult  *StopTime
	}{
		{ // With all required fields
			record:          "268435464;11:51:01;274137857;268435460",
			location:        location,
			errorIsExpected: false,
			expectedResult: &StopTime{
				Id: "268435464",
				Time: time.Date(
					0, time.January, 1,
					11, 51, 1, 0,
					location,
				),
				RouteStopPointId: "274137857",
				CourseId:         CourseId("268435460"),
			},
		},
		{ // With too many fields
			record:          "268435464;11:51:01;274137857;268435460;xxx",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
		{ // With too many fields
			record:          "268435464;11:51:01;274137857;268435460;",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
		{ // With a missing field
			record:          "268435464;11:51:01;274137857",
			location:        location,
			errorIsExpected: true,
			expectedResult:  nil,
		},
	}

	for _, test := range tests {
		_ = test
		splittedRecord := strings.Split(test.record, ";")
		getResult, err := newStopTime(
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
