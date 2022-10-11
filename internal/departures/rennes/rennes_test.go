package rennes

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/stretchr/testify/require"
)

const SECONDS_PER_HOUR int = 3_600

var LOCATION *time.Location = time.FixedZone("", 2*SECONDS_PER_HOUR)

var FIXTURE_DIR_PATH string

const DEFAULT_TIMEOUT time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	FIXTURE_DIR_PATH = os.Getenv("FIXTUREDIR")
	if FIXTURE_DIR_PATH == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestLoadTheoreticalDeparturesFromDailyDataFiles(t *testing.T) {

	const EXPECTED_NUM_OF_THEORETICAL_DEPARTURES int = 109_559
	var DAILY_SERVICE_START_TIME time.Time = time.Date(2012, 2, 29, 0, 0, 0, 0, LOCATION)

	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/2022-09-08/base_scheduled", FIXTURE_DIR_PATH))
	require.Nil(err)

	theoreticalDepartures, err := loadTheoreticalDeparturesFromDailyDataFiles(
		*uri,
		DEFAULT_TIMEOUT,
		&DAILY_SERVICE_START_TIME,
	)
	require.Nil(err)
	require.NotNil(theoreticalDepartures)
	require.Len(theoreticalDepartures, EXPECTED_NUM_OF_THEORETICAL_DEPARTURES)

	// Check the departure defined by the first line
	{
		const EXPECTED_STOP_TIME_ID string = "268506433"
		EXPECTED_DEPARTURE := Departure{
			StopTimeId:       EXPECTED_STOP_TIME_ID,
			RouteStopPointId: "284688651",
			StopPointId:      "1658",
			LineId:           "0801",
			Direction:        departures.DirectionTypeForward,
			DestinationId:    "1065",
			DestinationName:  "La Poterie",
			Time: time.Date(
				DAILY_SERVICE_START_TIME.Year(), DAILY_SERVICE_START_TIME.Month(), DAILY_SERVICE_START_TIME.Day(),
				20, 24, 0, 0,
				DAILY_SERVICE_START_TIME.Location(),
			),
		}
		require.Contains(theoreticalDepartures, EXPECTED_STOP_TIME_ID)
		require.Equal(theoreticalDepartures[EXPECTED_STOP_TIME_ID], EXPECTED_DEPARTURE)
	}

}
