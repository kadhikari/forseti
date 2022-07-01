package rennes

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hove-io/forseti/internal/departures"
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

func TestLoadScheduledDeparturesFromDailyDataFiles(t *testing.T) {

	const EXPECTED_NUM_OF_DEPARTURES int = 109_817

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	_ = assert
	departuresList, err := LoadScheduledDeparturesFromDailyDataFiles(*uri, defaultTimeout)
	require.Nil(err)
	require.NotNil(departuresList)
	require.Len(departuresList, EXPECTED_NUM_OF_DEPARTURES)

	loc, _ := time.LoadLocation("Europe/Paris")

	// Check the departure defined by the first line
	{
		departure := Departure{
			StopPointId:     "1412",
			BusLineId:       "0801",
			Direction:       departures.DirectionTypeBackward,
			DestinationId:   "284721153",
			DestinationName: "Kennedy",
			Time: DepartureTime{
				Id:            "268506433",
				ScheduledTime: time.Date(0, 1, 1, 5, 13, 0, 0, loc),
			},
		}
		require.Contains(departuresList, departure)
	}

	// // Check the departure defined by the last line
	// {
	// 	departure := Departure{
	// 		StopPointId:     "2126",
	// 		BusLineId:       "0001",
	// 		Direction:       departures.DirectionTypeForward,
	// 		DestinationId:   "268500995",
	// 		DestinationName: "Chantepie",
	// 		Time: DepartureTime{
	// 			Id:           "",
	// 			SceduledTime: time.Date(0, 1, 1, 5, 13, 0, 0, loc),
	// 		},
	// 	}
	// 	require.Contains(departuresList, departure)
	// }

}
