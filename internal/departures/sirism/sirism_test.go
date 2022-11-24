package sirism

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetNextResetDeparturesTimerFrom(t *testing.T) {
	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")

	var tests = []struct {
		dailyServiceSwitchTime        time.Time
		location                      *time.Location
		currentTime                   time.Time
		expectedNextServiceSwitchTime time.Time
	}{
		{ // Before the daily service switch time (= 00:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2021, time.December, 31, 23, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{ // At the daily service switch time (= 00:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2022, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 2, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{ // After the daily service switch time (= 00:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2022, time.January, 1, 0, 0, 0, 1, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 2, 0, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{ // Before daily the service switch time (= 03:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2022, time.January, 1, 2, 59, 59, 999_999_999, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 1, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{ // At the daily service switch time (= 03:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2022, time.January, 1, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 2, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{ // After the daily service switch time (= 03:00:00)
			dailyServiceSwitchTime:        time.Date(1, time.January, 1, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
			location:                      EUROPE_PARIS_LOCATION,
			currentTime:                   time.Date(2022, time.January, 1, 3, 0, 0, 1, EUROPE_PARIS_LOCATION),
			expectedNextServiceSwitchTime: time.Date(2022, time.January, 2, 3, 0, 0, 0, EUROPE_PARIS_LOCATION),
		},
	}

	assert := assert.New(t)

	for _, test := range tests {
		siriSmContext := &SiriSmContext{
			connector:               nil,
			lastUpdate:              nil,
			departures:              nil,
			notificationsStream:     nil,
			notificationsStreamName: "",
			roleARN:                 "",
			dailyServiceSwitchTime:  &test.dailyServiceSwitchTime,
			location:                test.location,
		}

		nextServiceSwitchTime, _ := siriSmContext.getNextResetDeparturesTimerFrom(&test.currentTime)
		assert.Equal(test.expectedNextServiceSwitchTime, nextServiceSwitchTime)
	}

}
