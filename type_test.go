package forseti

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"
	"time"
	"strconv"
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
	types := make([]FreeFloatingType, 0)
	types = append(types, StationType)
	types = append(types, ScooterType)
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

func TestNewStopPoint(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	spLine := []string{"CPC", "Copernic", "0:SP:80:4029", "0"}
	sp, err := NewStopPoint(spLine)
	require.Nil(err)
	require.NotNil(sp)

	assert.Equal(sp.Id, "stop_point:0:SP:80:4029")
	assert.Equal(sp.Name, "Copernic")
	assert.Equal(sp.Direction, 0)
}

func TestNewStoppointWithMissingField(t *testing.T) {
	require := require.New(t)
	spLine := []string{"CPC", "Copernic", "0:SP:80:4029"}
	sp, err := NewStopPoint(spLine)
	require.Error(err)
	require.Nil(sp)
}

func TestNewCourse(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	courseLine := []string{"40","2774327","1","05:47:18","06:06:16.5","13","2020-09-21","2021-01-18","2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	require.NotNil(course)

	assert.Equal(course.LineCode, "40")
	assert.Equal(course.Course, "2774327")
	assert.Equal(course.DayOfWeek, 1)
	assert.Equal(course.FirstDate, time.Date(2020, 9, 21, 0, 0, 0, 0, location))
	firstTime, err := time.ParseInLocation("15:04:05", "05:47:18", location)
	require.Nil(err)
	assert.Equal(course.FirstTime, firstTime)
}

func TestNewCourseWithMissingField(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	courseLine := []string{"40","2774327","1","05:47:18","06:06:16.5","13","2020-09-21","2021-01-18"}
	course, err := NewCourse(courseLine, location)
	require.Error(err)
	require.Nil(course)
}

func TestNewRouteSchedule(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)	
	require.Nil(err)
	require.NotNil(rs)

	assert.Equal(rs.Id, 1)
	assert.Equal(rs.LineCode, "40")
	assert.Equal(rs.VehicleJourneyId, "vj_id_one")
	assert.Equal(rs.StopId, "stop_point:0:SP:80:4029")
	assert.Equal(rs.Direction, 0)
	assert.Equal(rs.Departure, true)
	dateTime, err := time.ParseInLocation("20060102T150405", "20210222T054500", location)
	require.Nil(err)
	assert.Equal(rs.DateTime, dateTime)
}

func TestDataManagerForVehicleOccupancies(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	var manager DataManager

	// Load StopPoints
	stopPoints := make(map[string]StopPoint)
	sp, err := NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4029","0"})
	require.Nil(err)
	stopPoints[sp.Name + strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4028","1"})
	require.Nil(err)
	stopPoints[sp.Name + strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4142","0"})
	require.Nil(err)
	stopPoints[sp.Name + strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141","1"})
	require.Nil(err)
	_, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141","2"})
	assert.NotNil(err)
	stopPoints[sp.Name + strconv.Itoa(sp.Direction)] = *sp
	manager.InitStopPoint(stopPoints)
	assert.Equal(len(*manager.stopPoints), 4)

	// Load Courses
	courses := make(map[string][]Course)
	courseLine := []string{"40","2774327","1","05:45:00","06:06:00","13","2020-09-21","2021-01-18","2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	courseLine = []string{"40","2774327","2","05:46:00","06:07:00","13","2020-09-21","2021-01-18","2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	manager.InitCourse(courses)
	assert.Equal(len(*manager.courses), 1)

	// Load RouteSchedules
	routeSchedules := make ([]RouteSchedule, 0)
	rs, err := NewRouteSchedule("40", "stop_point:0:SP:80:4029", "vj_id_one", "20210222T054500", 0, 1, true, location)
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	rs, err = NewRouteSchedule("40", "stop_point:0:SP:80:4142", "vj_id_one", "20210222T055000", 0, 2, false, location)	
	require.Nil(err)
	routeSchedules = append(routeSchedules, *rs)
	manager.InitRouteSchedule(routeSchedules)
	assert.Equal(len(*manager.routeSchedules), 2)

	// Load Predictions
	predictions := make([]Prediction, 0)
	pn := PredictionNode{Line: "40", Sens: 0, Date: "2021-02-22T00:00:00", Course: "2774327", Order: 0, StopName: "Copernic", Charge: 55}
	p := NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = PredictionNode{Line: "40", Sens: 0, Date: "2021-02-22T00:00:00", Course: "2774327", Order: 1, StopName: "Pasteur", Charge: 75}
	p = NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	assert.Equal(len(predictions), 2)

	// Create VehicleOccupancies from existing data
	occupanciesWithCharge := CreateOccupanciesFromPredictions(&manager, predictions)
	assert.Equal(len(occupanciesWithCharge), 2)
	manager.UpdateVehicleOccupancies(occupanciesWithCharge)
	assert.Equal(len(*manager.vehicleOccupancies), 2)
	date, err := time.ParseInLocation("2006-01-02", "2021-02-22", location)
	require.Nil(err)
	param := VehicleOccupancyRequestParameter{StopId: "", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err := manager.GetVehicleOccupancies(&param)
	assert.Equal(len(vehicleOccupancies), 2)

	// Call Api with StopId in the paraameter
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4029", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = manager.GetVehicleOccupancies(&param)
	assert.Equal(len(vehicleOccupancies), 1)

	// Verify attributes
	dateTime, err := time.ParseInLocation("20060102T150405", "20210222T054500", location)
	require.Nil(err)
	assert.Equal(vehicleOccupancies[0].LineCode, "40")
	assert.Equal(vehicleOccupancies[0].VehicleJourneyId, "vj_id_one")
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4029") //Copernic : departure StopPoint
	assert.Equal(vehicleOccupancies[0].Direction, 0)
	assert.Equal(vehicleOccupancies[0].DateTime, dateTime)
	assert.Equal(vehicleOccupancies[0].Occupancy, 55)

	// Call Api with another StopId in the paraameter
	param = VehicleOccupancyRequestParameter{StopId: "stop_point:0:SP:80:4142", VehicleJourneyId: "", Date: date}
	vehicleOccupancies, err = manager.GetVehicleOccupancies(&param)
	assert.Equal(len(vehicleOccupancies), 1)

	dateTime, err = time.ParseInLocation("20060102T150405", "20210222T055000", location)
	require.Nil(err)
	assert.Equal(vehicleOccupancies[0].LineCode, "40")
	assert.Equal(vehicleOccupancies[0].VehicleJourneyId, "vj_id_one")
	assert.Equal(vehicleOccupancies[0].StopId, "stop_point:0:SP:80:4142") //Pasteur
	assert.Equal(vehicleOccupancies[0].Direction, 0)
	assert.Equal(vehicleOccupancies[0].DateTime, dateTime)
	assert.Equal(vehicleOccupancies[0].Occupancy, 75)
}
