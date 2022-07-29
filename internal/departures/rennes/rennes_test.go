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

var fixtureDir string

const defaultTimeout time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestLoadTheoreticalDeparturesFromDailyDataFiles(t *testing.T) {

	const EXPECTED_NUM_OF_THEORETICAL_DEPARTURES int = 109_817

	var LOCATION *time.Location
	LOCATION, _ = time.LoadLocation("Europe/Paris")
	var DAILY_SERVICE_START_TIME time.Time = time.Date(2012, 2, 29, 0, 0, 0, 0, LOCATION)

	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referential", fixtureDir))
	require.Nil(err)

	theoreticalDepartures, err := loadTheoreticalDeparturesFromDailyDataFiles(
		*uri,
		defaultTimeout,
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
			RouteStopPointId: "284722693",
			StopPointId:      "1412",
			LineId:           "0801",
			Direction:        departures.DirectionTypeBackward,
			DestinationId:    "284721153",
			DestinationName:  "Kennedy",
			Time: time.Date(
				DAILY_SERVICE_START_TIME.Year(), DAILY_SERVICE_START_TIME.Month(), DAILY_SERVICE_START_TIME.Day(),
				5, 13, 0, 0,
				DAILY_SERVICE_START_TIME.Location(),
			),
		}
		require.Contains(theoreticalDepartures, EXPECTED_STOP_TIME_ID)
		require.Equal(theoreticalDepartures[EXPECTED_STOP_TIME_ID], EXPECTED_DEPARTURE)
	}

}
