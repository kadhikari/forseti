package sytralrt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"

	"encoding/xml"

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

func TestDataManagerCanGetParkingById(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"toto": {"DECC", "Décines Centre", updateTime, 1, 2, 3, 4},
	})

	p, err := manager.GetParkingById("toto")
	assert.Nil(err)
	assert.Equal("DECC", p.ID)
	assert.Equal("Décines Centre", p.Label)
	assert.Equal(time.Date(2018, 9, 17, 19, 29, 0, 0, loc), p.UpdatedTime)
	assert.Equal(1, p.AvailableStandardSpaces)
	assert.Equal(2, p.AvailableAccessibleSpaces)
	assert.Equal(3, p.TotalStandardSpaces)
	assert.Equal(4, p.TotalAccessibleSpaces)
}

func TestDataManagerShouldErrorOnEmptyData(t *testing.T) {
	assert := assert.New(t)

	var manager DataManager
	p, err := manager.GetParkingById("toto")
	assert.Error(err)
	assert.Empty(p)
}

func TestDataManagerShouldErrorOnEmptyData2(t *testing.T) {
	assert := assert.New(t)

	var manager DataManager
	p, errs := manager.GetParkingsByIds([]string{"toto", "titi"})
	assert.NotEmpty(errs)
	assert.Empty(p)
	for _, e := range errs {
		assert.Error(e)
	}
}
func TestDataManagerCanGetParkingByIds(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	p, errs := manager.GetParkingsByIds([]string{"riri", "loulou"})
	assert.Nil(errs)
	require.Len(p, 2)
	assert.Equal("Riri", p[0].ID)
	assert.Equal("Loulou", p[1].ID)
}

func TestDataManagerCanPartiallyGetParkingByIds(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	p, errs := manager.GetParkingsByIds([]string{"fifi", "donald"})
	require.Len(p, 1)
	assert.Equal("Fifi", p[0].ID)

	require.Len(errs, 1)
	assert.Error(errs[0])
}

func TestDataManagerGetParking(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var manager DataManager
	manager.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	parkings, err := manager.GetParkings()
	require.Nil(err)
	require.Len(parkings, 3)

	sort.Sort(ByParkingId(parkings))
	assert.Equal("Fifi", parkings[0].ID)
	assert.Equal("Loulou", parkings[1].ID)
	assert.Equal("Riri", parkings[2].ID)
}

func TestData(t *testing.T) {
	assert := assert.New(t)

	xmlFile, err := os.Open("/home/kadhikari/partage/temp/NET_ACCESS.XML")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer xmlFile.Close()
	XMLdata, err := ioutil.ReadAll(xmlFile)
	assert.Nil(err)

	decoder := xml.NewDecoder(bytes.NewReader(XMLdata))
	decoder.CharsetReader = getCharsetReader

	var root Root
	err = decoder.Decode(&root)
	assert.Nil(err)

	assert.Equal(len(root.Data.Lines), 8)
}

func TestNewEquipmentDetail(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	readLine := []string{"821", "direction Gare de Vaise, accès Gare Routière ou Parc Relais", "ASCENSEUR", "Problème technique",
		"Accès impossible direction Gare de Vaise.", "2018-09-14", "2018-09-14", "13:00:00"}

	e, err := NewEquipmentDetail(readLine, location)
	require.Nil(err)
	require.NotNil(e)

	assert.Equal("821", e.ID)
	assert.Equal("direction Gare de Vaise, accès Gare Routière ou Parc Relais", e.Name)
	assert.Equal("elevator", e.EmbeddedType)
	assert.Equal("Problème technique", e.CurrentAvailability.Cause.Label)
	assert.Equal("Accès impossible direction Gare de Vaise.", e.CurrentAvailability.Effect.Label)
	assert.Equal(time.Date(2018, 9, 14, 0, 0, 0, 0, location), e.CurrentAvailability.Periods.Begin)
	assert.Equal(time.Date(2018, 9, 14, 13, 0, 0, 0, location), e.CurrentAvailability.Periods.End)
}

func TestDataManagerGetEquipments(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	var manager DataManager
	equipmentMaps := make(map[string]EquipmentDetail)

	equipmentsData := [][]string{
		{"toto", "toto paris", "ASCENSEUR", "Problème technique", "Accès", "2018-09-17", "2018-09-18", "23:30:00"},
		{"tata", "tata paris", "ESCALIER", "Problème technique", "Accès", "2018-09-17", "2018-09-18", "23:30:00"},
		{"titi", "titi paris", "ESCALIER", "Problème technique", "Accès", "2018-09-17", "2018-09-18", "23:30:00"},
	}

	for _, edLine := range equipmentsData {
		ed, err := NewEquipmentDetail(edLine, loc)
		require.Nil(err)
		equipmentMaps[ed.ID] = *ed
	}
	manager.UpdateEquipments(equipmentMaps)
	equipments, err := manager.GetEquipments()
	require.Nil(err)
	require.Len(equipments, 3)

	sort.Sort(ByEquipmentId(equipments))
	assert.Equal("tata", equipments[0].ID)
	assert.Equal("titi", equipments[1].ID)
	assert.Equal("toto", equipments[2].ID)
}

func TestEquipmentsWithBadEmbeddedType(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	equipmentsWithBadEType := [][]string{
		{"toto", "toto paris", "ASC", "Problème technique", "Accès", "2018-09-17", "2018-09-18", "23:30:00"},
		{"tata", "tata paris", "ASC", "Problème technique", "Accès", "2018-09-17", "2018-09-18", "23:30:00"},
	}

	for _, badETypeLine := range equipmentsWithBadEType {
		ed, err := NewEquipmentDetail(badETypeLine, loc)
		assert.Error(err)
		assert.Nil(ed)
	}
}
