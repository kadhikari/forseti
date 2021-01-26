package forseti

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
	assert.Equal(DirectionTypeUnknown, d.DirectionType)

	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)
}

func TestNewDepartureWithDirectionType(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	d, err := NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3", "vjid", "said", "ALL"},
		location)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	assert.Equal(DirectionTypeForward, d.DirectionType)

	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)

	d, err = NewDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3", "vjid", "said", "RET"},
		location)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	assert.Equal(DirectionTypeBackward, d.DirectionType)

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
	fileName := fmt.Sprintf("/%s/NET_ACCESS.XML", fixtureDir)
	xmlFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		require.Fail(t, err.Error())
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
	updatedAt := time.Now()
	es := EquipementSource{ID: "821", Name: "direction Gare de Vaise, accès Gare Routière ou Parc Relais",
		Type: "ASCENSEUR", Cause: "Problème technique", Effect: "Accès impossible direction Gare de Vaise.",
		Start: "2018-09-14", End: "2018-09-14", Hour: "13:00:00"}
	e, err := NewEquipmentDetail(es, updatedAt, location)

	require.Nil(err)
	require.NotNil(e)

	assert.Equal("821", e.ID)
	assert.Equal("direction Gare de Vaise, accès Gare Routière ou Parc Relais", e.Name)
	assert.Equal("elevator", e.EmbeddedType)
	assert.Equal("Problème technique", e.CurrentAvailability.Cause.Label)
	assert.Equal("Accès impossible direction Gare de Vaise.", e.CurrentAvailability.Effect.Label)
	assert.Equal(time.Date(2018, 9, 14, 0, 0, 0, 0, location), e.CurrentAvailability.Periods[0].Begin)
	assert.Equal(time.Date(2018, 9, 14, 13, 0, 0, 0, location), e.CurrentAvailability.Periods[0].End)
	assert.Equal(updatedAt, e.CurrentAvailability.UpdatedAt)
}

func TestDataManagerGetEquipments(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	var manager DataManager
	equipments := make([]EquipmentDetail, 0)
	ess := make([]EquipementSource, 0)
	es := EquipementSource{ID: "toto", Name: "toto paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	es = EquipementSource{ID: "tata", Name: "tata paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	es = EquipementSource{ID: "titi", Name: "toto paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	updatedAt := time.Now()

	for _, es := range ess {
		ed, err := NewEquipmentDetail(es, updatedAt, loc)
		require.Nil(err)
		equipments = append(equipments, *ed)
	}
	manager.UpdateEquipments(equipments)
	equipDetails, err := manager.GetEquipments()
	require.Nil(err)
	require.Len(equipments, 3)

	assert.Equal("toto", equipDetails[0].ID)
	assert.Equal("tata", equipDetails[1].ID)
	assert.Equal("titi", equipDetails[2].ID)
}

func TestEquipmentsWithBadEmbeddedType(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	equipmentsWithBadEType := make([]EquipementSource, 0)
	es := EquipementSource{ID: "toto", Name: "toto paris", Type: "ASC", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	equipmentsWithBadEType = append(equipmentsWithBadEType, es)
	es = EquipementSource{ID: "tata", Name: "tata paris", Type: "ASC", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	equipmentsWithBadEType = append(equipmentsWithBadEType, es)
	updatedAt := time.Now()

	for _, badEType := range equipmentsWithBadEType {
		ed, err := NewEquipmentDetail(badEType, updatedAt, loc)
		assert.Error(err)
		assert.Nil(ed)
	}
}

func TestParseDirectionType(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(DirectionTypeForward, ParseDirectionType("ALL"))
	assert.Equal(DirectionTypeBackward, ParseDirectionType("RET"))
	assert.Equal(DirectionTypeUnknown, ParseDirectionType(""))
	assert.Equal(DirectionTypeUnknown, ParseDirectionType("foo"))
}

func TestParseDirectionTypeFromNavitia(t *testing.T) {
	assert := assert.New(t)

	r, err := ParseDirectionTypeFromNavitia("forward")
	assert.Nil(err)
	assert.Equal(DirectionTypeForward, r)

	r, err = ParseDirectionTypeFromNavitia("backward")
	assert.Nil(err)
	assert.Equal(DirectionTypeBackward, r)

	r, err = ParseDirectionTypeFromNavitia("")
	assert.Nil(err)
	assert.Equal(DirectionTypeBoth, r)

	_, err = ParseDirectionTypeFromNavitia("foo")
	assert.NotNil(err)
	_, err = ParseDirectionTypeFromNavitia("ALL")
	assert.NotNil(err)
}

func TestKeepDirection(t *testing.T) {
	assert := assert.New(t)
	assert.True(keepDirection(DirectionTypeForward, DirectionTypeForward))
	assert.False(keepDirection(DirectionTypeForward, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeForward, DirectionTypeBoth))

	assert.False(keepDirection(DirectionTypeBackward, DirectionTypeForward))
	assert.True(keepDirection(DirectionTypeBackward, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeBackward, DirectionTypeBoth))

	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeForward))
	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeBoth))
}

func TestNewFreeFloating(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	provider := ProviderNode {Name: "Pony"}
	ff := Vehicle{PublicId: "NSCBH3", Provider: provider, Id: "cG9ueTpCSUtFOjEwMDQ0MQ", Type: "BIKE",
	Latitude: 45.180335, Longitude:  5.7069425, Propulsion: "ASSIST", Battery: 85, Deeplink: "http://test"}
	f := NewFreeFloating(ff)
	require.NotNil(f)

	assert.Equal("NSCBH3", f.PublicId)
	assert.Equal("cG9ueTpCSUtFOjEwMDQ0MQ", f.Id)
	assert.Equal("Pony", f.ProviderName)
	assert.Equal("BIKE", f.Type)
	assert.Equal(45.180335, f.Coord.Lat)
	assert.Equal(5.7069425, f.Coord.Lon)
	assert.Equal("ASSIST", f.Propulsion)
	assert.Equal(85, f.Battery)
	assert.Equal("http://test", f.Deeplink)
}

func TestDataManagerGetFreeFloatings(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var manager DataManager
	p1 := ProviderNode {Name: "Pony"}
	p2 := ProviderNode {Name: "Tier"}

	vehicles := make([]Vehicle, 0)
	freeFloatings := make([]FreeFloating, 0)
	v := Vehicle{PublicId: "NSCBH3", Provider: p1, Id: "cG9ueTpCSUtFOjEwMDQ0MQ", Type: "BIKE",
		Latitude: 48.847232, Longitude:  2.377601, Propulsion: "ASSIST", Battery: 85, Deeplink: "http://test1"}
	vehicles = append(vehicles, v)
	v = Vehicle{PublicId: "718WSK", Provider: p1, Id: "cG9ueTpCSUtFOjEwMDQ0MQ==4b", Type: "SCOOTER",
		Latitude: 48.847299, Longitude:  2.37772, Propulsion: "ELECTRIC", Battery: 85, Deeplink: "http://test2"}
	vehicles = append(vehicles, v)
	v = Vehicle{PublicId: "0JT9J6", Provider: p2, Id: "cG9ueTpCSUtFOjEwMDQ0MQ==55c", Type: "SCOOTER",
		Latitude: 48.847326, Longitude:  2.377734, Propulsion: "ELECTRIC", Battery: 85, Deeplink: "http://test3"}
	vehicles = append(vehicles, v)

	for _, vehicle := range vehicles {
		freeFloatings = append(freeFloatings, *NewFreeFloating(vehicle))
	}
	manager.UpdateFreeFloating(freeFloatings)
	// init parameters:
	types := make([]string, 0)
	types = append(types, "STATION")
	types = append(types, "SCOOTER")
	coord := Coord{Lat: 48.846781, Lon: 2.37715}
	p:= FreeFloatingRequestParameter{distance: 500, coord: coord, count: 10, types: types}
	free_floatings, err := manager.GetFreeFloatings(&p)
	require.Nil(err)
	require.Len(free_floatings, 2)

	assert.Equal("718WSK", free_floatings[0].PublicId)
	assert.Equal("Pony", free_floatings[0].ProviderName)
	assert.Equal("SCOOTER", free_floatings[0].Type)
	assert.Equal("0JT9J6", free_floatings[1].PublicId)
	assert.Equal("Tier", free_floatings[1].ProviderName)
}
