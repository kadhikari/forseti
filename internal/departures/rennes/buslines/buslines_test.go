package buslines

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

func TestLoadBusLines(t *testing.T) {

	const EXPECTED_NUM_OF_BUS_LINES int = 121

	assert := assert.New(t)
	require := require.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referentiel", fixtureDir))
	require.Nil(err)

	loadedBusLines, err := LoadBusLines(*uri, defaultTimeout)
	require.Nil(err)
	assert.Len(loadedBusLines, EXPECTED_NUM_OF_BUS_LINES)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_INTERNAL_ID string = "248"
		const EXPECTED_EXTERNAL_ID string = "0801"

		assert.Contains(loadedBusLines, EXPECTED_INTERNAL_ID)
		assert.Equal(
			loadedBusLines[EXPECTED_INTERNAL_ID],
			BusLine{
				InternalId: EXPECTED_INTERNAL_ID,
				ExternalId: EXPECTED_EXTERNAL_ID,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_INTERNAL_ID string = "1"
		const EXPECTED_EXTERNAL_ID string = "0001"

		assert.Contains(loadedBusLines, EXPECTED_INTERNAL_ID)
		assert.Equal(
			loadedBusLines[EXPECTED_INTERNAL_ID],
			BusLine{
				InternalId: EXPECTED_INTERNAL_ID,
				ExternalId: EXPECTED_EXTERNAL_ID,
			},
		)
	}

}
