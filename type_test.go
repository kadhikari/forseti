package sytralrt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeparture(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	d, err := NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3"}, location)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)
}

func TestNewDepartureMissingField(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	_, err = NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00"}, location)
	require.Error(err)
}

func TestNewDepartureInvalidDate(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	_, err = NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28", "3"}, location)
	require.Error(err)

	_, err = NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 25:28:00", "3"}, location)
	require.Error(err)
}
