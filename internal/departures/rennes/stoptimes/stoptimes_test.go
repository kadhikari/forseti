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
		localDailyServiceStartTime time.Time
		localTimeFirstLine         time.Time
		localTimeLastLine          time.Time
		utcTimeFirstLine           time.Time
		urcTimeLastLine            time.Time
	}{
		{ // UTC
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, time.UTC),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 56, 0, 0, time.UTC),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 12, 56, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 4, 46, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 56, 0, 0, EUROPE_PARIS_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, EUROPE_PARIS_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 11, 56, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 3, 46, 1, 0, time.UTC),
		},
		{ // Europe/Paris
			localDailyServiceStartTime: time.Date(2012, time.February, 29, 0, 0, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeFirstLine:         time.Date(2012, time.February, 29, 12, 56, 0, 0, AMERICA_NEWYORK_LOCATION),
			localTimeLastLine:          time.Date(2012, time.February, 29, 4, 46, 1, 0, AMERICA_NEWYORK_LOCATION),
			utcTimeFirstLine:           time.Date(2012, time.February, 29, 17, 56, 0, 0, time.UTC),
			urcTimeLastLine:            time.Date(2012, time.February, 29, 9, 46, 1, 0, time.UTC),
		},
	}

	for _, test := range tests {

		localDailyServiceStartTime := test.localDailyServiceStartTime

		assert := assert.New(t)
		require := require.New(t)
		uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referential", fixtureDir))
		require.Nil(err)

		loadedStopTimes, err := LoadStopTimes(*uri, defaultTimeout, &localDailyServiceStartTime)
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

func TestIsBelongedToPreviousDay(t *testing.T) {

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")

	var tests = []struct {
		name                   string
		t                      time.Time
		dailyServiceSwitchTime time.Time
		errorIsExpected        bool
		expectedResult         bool
	}{
		// At same time
		{
			name:                   "same time than 12:00:00 ",
			t:                      time.Date(2022, 2, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond after 12:00:00",
			t:                      time.Date(2022, 2, 1, 12, 0, 0, 1, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond before 12:00:00",
			t:                      time.Date(2022, 2, 1, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         true,
		},
		{
			name:                   "one second after 12:00:00",
			t:                      time.Date(2022, 2, 1, 12, 0, 1, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one second before 12:00:00",
			t:                      time.Date(2022, 2, 1, 11, 59, 59, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         true,
		},
		{
			name:                   "one minute after 12:00:00",
			t:                      time.Date(2022, 2, 1, 12, 1, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one minute before 12:00:00",
			t:                      time.Date(2022, 2, 1, 11, 59, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         true,
		},
		{
			name:                   "one hour after 12:00:00",
			t:                      time.Date(2022, 2, 1, 13, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one hour before 12:00:00",
			t:                      time.Date(2022, 2, 1, 11, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         true,
		},
		{
			name:                   "same time than 00:00:00 ",
			t:                      time.Date(2022, 2, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond after 00:00:00 ",
			t:                      time.Date(2022, 2, 1, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond before 00:00:00 ",
			t:                      time.Date(2022, 2, 1, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "same time than 23:59:59:999999999",
			t:                      time.Date(2022, 2, 1, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond after 23:59:59:999999999 ",
			t:                      time.Date(2022, 2, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         false,
		},
		{
			name:                   "one nanosecond before 23:59:59:999999999  ",
			t:                      time.Date(2022, 2, 1, 23, 59, 59, 999_999_998, EUROPE_PARIS_LOCATION),
			dailyServiceSwitchTime: time.Date(0, 1, 1, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			errorIsExpected:        false,
			expectedResult:         true,
		},
	}

	_ = tests

	assert := assert.New(t)
	for _, test := range tests {
		errMessage := fmt.Sprintf("the unit test '%s' failed", test.name)
		gotResult, gotError := IsBelongedToNextDay(&test.t, &test.dailyServiceSwitchTime)
		assert.Equal(gotError != nil, test.errorIsExpected, errMessage)
		assert.Equal(gotResult, test.expectedResult, errMessage)
	}
}

func TestComputeActualStopTime(t *testing.T) {
	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	SERVICE_START_TIME_AT_MIDNIGHT := time.Date(2022, time.June, 15, 0, 0, 0, 0, EUROPE_PARIS_LOCATION)
	SERVICE_START_TIME_AT_MIDDAY := time.Date(2022, time.June, 15, 12, 0, 0, 0, EUROPE_PARIS_LOCATION)

	var tests = []struct {
		name                   string
		stopTime               StopTime
		dailyServiceStartTime  time.Time
		expectedActualStopTime time.Time
	}{
		{
			name: "0 nanosecond after service start time (midday)",
			stopTime: StopTime{
				Id:               "UNUSED_ID",
				Time:             time.Date(0, time.January, 1, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
				RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
			},
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 15, 12, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{
			name: "1 nanosecond after service start time (midday)",
			stopTime: StopTime{
				Id:               "UNUSED_ID",
				Time:             time.Date(0, time.January, 1, 12, 0, 0, 1, EUROPE_PARIS_LOCATION),
				RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
			},
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 15, 12, 0, 0, 1, EUROPE_PARIS_LOCATION),
		},
		{
			name: "1 nanosecond before service start time (midday)",
			stopTime: StopTime{
				Id:               "UNUSED_ID",
				Time:             time.Date(0, time.January, 1, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
				RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
			},
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDDAY,
			expectedActualStopTime: time.Date(2022, time.June, 16, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
		},
		// {
		// 	name: "0 nanosecond after service start time (midnight)",
		// 	stopTime: StopTime{
		// 		Id:               "UNUSED_ID",
		// 		Time:             time.Date(0, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		// 		RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
		// 	},
		// 	dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
		// 	expectedActualStopTime: time.Date(2022, time.June, 15, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		// },
		{
			name: "1 nanosecond after service start time (midnight)",
			stopTime: StopTime{
				Id:               "UNUSED_ID",
				Time:             time.Date(0, time.January, 2, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
				RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
			},
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
			expectedActualStopTime: time.Date(2022, time.June, 15, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
		},
		{
			name: "1 nanosecond before service start time (midnight)",
			stopTime: StopTime{
				Id:               "UNUSED_ID",
				Time:             time.Date(0, time.January, 2, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
				RouteStopPointId: "UNUSED_ROUTE_STOP_POINT_ID",
			},
			dailyServiceStartTime:  SERVICE_START_TIME_AT_MIDNIGHT,
			expectedActualStopTime: time.Date(2022, time.June, 15, 11, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
		},
	}
	assert := assert.New(t)
	for _, test := range tests {
		errMessage := fmt.Sprintf("the unit test '%s' failed", test.name)
		gotActualStopTime := test.stopTime.computeActualStopTime(&test.dailyServiceStartTime)
		assert.Equal(test.expectedActualStopTime, gotActualStopTime, errMessage)
	}
}
