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

func TestDataManagerLastUpdate(t *testing.T) {
	require := require.New(t)
	begin := time.Now()

	var manager DataManager
	manager.UpdateDepartures(nil)

	lastDepartureUpdate := manager.lastDepartureUpdate
	require.True(lastDepartureUpdate.After(begin))

	manager.UpdateDepartures(nil)
	require.True(manager.lastDepartureUpdate.After(lastDepartureUpdate))
}

func TestNewParking(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	parkingLine := []string{"DECC", "Décines Centre", "2018-09-17 19:29:00", "2018-09-17 19:30:02", "82", "105", "3", "4"}

	p, err := NewParking(parkingLine, location)
	require.Nil(err)
	require.NotNil(p)

	assert.Equal("DECC", p.ID)
	assert.Equal("Décines Centre", p.Label)
	assert.Equal(time.Date(2018, 9, 17, 19, 29, 0, 0, location), p.UpdatedTime)
	assert.Equal(82, p.AvailableStandardSpaces)
	assert.Equal(105, p.TotalStandardSpaces)
	assert.Equal(3, p.AvailableAccessibleSpaces)
	assert.Equal(4, p.TotalAccessibleSpaces)
}

func TestNewParkingWithMissingFields(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	parkingLine := []string{
		"DECC", "Décines Centre", "2018-09-17 19:29:00", "2018-09-17 19:30:02", "82", "105"}

	p, err := NewParking(parkingLine, location)
	assert.Error(err)
	assert.Nil(p)
}

func TestNewParkingWithMalformedFields(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	malformedParkingLines := [][]string{
		{"DECC", "Décines Centre", "this_should_be_a_date", "2018-09-17 19:30:02", "82", "105", "3", "4"},
		{"DECC", "Décines Centre", "2018-09-17 19:30:02", "2018-09-17 19:30:02", "this_is_an_int", "105", "3", "4"},
		{"DECC", "Décines Centre", "2018-09-17 19:30:02", "2018-09-17 19:30:02", "82", "another_int", "3", "4"},
		{"DECC", "Décines Centre", "2018-09-17 19:30:02", "2018-09-17 19:30:02", "82", "105", "int", "4"},
		{"DECC", "Décines Centre", "2018-09-17 19:30:02", "2018-09-17 19:30:02", "82", "105", "3", ""},
	}

	for _, malformedLine := range malformedParkingLines {
		p, err := NewParking(malformedLine, location)
		assert.Error(err)
		assert.Nil(p)
	}
}
