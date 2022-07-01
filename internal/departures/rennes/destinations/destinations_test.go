package destinations

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

func TestLoadDestinations(t *testing.T) {

	const EXPECTED_NUM_OF_DESTINATIONS int = 611

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedDestinations, err := LoadDestinations(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedDestinations, EXPECTED_NUM_OF_DESTINATIONS)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_ID string = "284721153"
		const EXPECTED_NAME string = "Kennedy"

		assert.Contains(loadedDestinations, EXPECTED_ID)
		assert.Equal(
			loadedDestinations[EXPECTED_ID],
			Destination{
				Id:   EXPECTED_ID,
				Name: EXPECTED_NAME,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_ID string = "268500992"
		const EXPECTED_NAME string = "Dépôt"

		assert.Contains(loadedDestinations, EXPECTED_ID)
		assert.Equal(
			loadedDestinations[EXPECTED_ID],
			Destination{
				Id:   EXPECTED_ID,
				Name: EXPECTED_NAME,
			},
		)
	}
}
