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

	var LOCATION *time.Location
	LOCATION, _ = time.LoadLocation("Europe/Paris")
	var PROCESSING_DATE time.Time = time.Date(2012, 2, 29, 0, 0, 0, 0, LOCATION)

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	_ = assert
	departuresList, err := LoadScheduledDeparturesFromDailyDataFiles(*uri, defaultTimeout, &PROCESSING_DATE)
	require.Nil(err)
	require.NotNil(departuresList)
	require.Len(departuresList, EXPECTED_NUM_OF_DEPARTURES)

	// Check the departure defined by the first line
	{
		departure := Departure{
			RouteStopPointId: "284722693",
			StopPointId:      "1412",
			LineId:           "0801",
			Direction:        departures.DirectionTypeBackward,
			DestinationId:    "284721153",
			DestinationName:  "Kennedy",
			Time: time.Date(
				PROCESSING_DATE.Year(), PROCESSING_DATE.Month(), PROCESSING_DATE.Day(),
				5, 13, 0, 0,
				PROCESSING_DATE.Location(),
			),
		}
		require.Contains(departuresList, departure)
	}

}
