package lastupdate

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

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

func TestParseLastUpdateInLocation(t *testing.T) {

	EUROPE_PARIS_LOCATION, _ := time.LoadLocation("Europe/Paris")
	AMERICA_NEWYORK_LOCATION, _ := time.LoadLocation("America/New_York")
	// AMERICA_NEWYORK_LOCATION, _ := time.LoadLocation("America/New_York")
	var tests = []struct {
		location           *time.Location
		timeStr            string
		expectedLastUpdate time.Time
	}{
		{
			location:           EUROPE_PARIS_LOCATION,
			timeStr:            "2022-04-04 15:38:00",
			expectedLastUpdate: time.Date(2022, time.April, 4, 15, 38, 0, 0, EUROPE_PARIS_LOCATION),
		},
		{
			location:           AMERICA_NEWYORK_LOCATION,
			timeStr:            "2022-04-04 15:38:00",
			expectedLastUpdate: time.Date(2022, time.April, 4, 15, 38, 0, 0, AMERICA_NEWYORK_LOCATION),
		},
	}
	require := require.New(t)

	for _, test := range tests {
		gotLastUpdate, err := parseLastUpdateInLocation(test.timeStr, test.location)

		require.Nil(err)
		require.Equal(gotLastUpdate, test.expectedLastUpdate)
	}
}

func TestExtractCsvStringFromJson(t *testing.T) {
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/2022-09-08/real_time/%s", fixtureDir, lastUpdateFileName))
	require.Nil(err)

	inputBytes, err := ioutil.ReadFile(uri.Path)
	require.Nil(err)

	csvString, err := extractCsvStringFromJson(inputBytes)
	require.Nil(err)
	require.NotEmpty(csvString)
	// Check the string of the extracted CSV
	require.Equal(csvString, "2022-09-08 11:28:58")
}
