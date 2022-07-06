package departures

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDepartureTypeString(t *testing.T) {
	require := require.New(t)
	{
		require.Equal(fmt.Sprint(DepartureTypeTheoretical), "T")
	}
	{
		require.Equal(fmt.Sprint(DepartureTypeEstimated), "E")
	}
}
