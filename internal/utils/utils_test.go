package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToInt(t *testing.T) {

	assert := assert.New(t)

    result := StringToInt("10", 0)
    assert.Equal(result, 10)

    result = StringToInt("aas", 0)
    assert.Equal(result, 0)
}

func TestAddDateAndTime(t *testing.T) {

    assert := assert.New(t)
	require := require.New(t)

    str := "2014-11-12T00:00:00.000Z"
    date, err := time.Parse(time.RFC3339, str)
	require.Nil(err)

    offsetStr := "0001-01-01T00:00:00.000Z"
    offsetDate, err := time.Parse(time.RFC3339, offsetStr)
	require.Nil(err)

    result := AddDateAndTime(date, offsetDate)
    assert.Equal(result.String(), "2015-11-12 00:00:00 +0000 UTC")
}

func TestCalculateOccupancy(t *testing.T) {

	assert := assert.New(t)

    result := CalculateOccupancy(0)
    assert.Equal(result, 0)
    result = CalculateOccupancy(50)
    assert.Equal(result, 50)
}
