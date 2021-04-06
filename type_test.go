package forseti

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/equipments"
)

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

	var root data.Root
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
	es := data.EquipementSource{ID: "821", Name: "direction Gare de Vaise, accès Gare Routière ou Parc Relais",
		Type: "ASCENSEUR", Cause: "Problème technique", Effect: "Accès impossible direction Gare de Vaise.",
		Start: "2018-09-14", End: "2018-09-14", Hour: "13:00:00"}
	e, err := equipments.NewEquipmentDetail(es, updatedAt, location)

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

	courseLine := []string{"40", "2774327", "1", "05:47:18", "06:06:16.5",
		"13", "2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
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

	courseLine := []string{"40", "2774327", "1", "05:47:18", "06:06:16.5", "13", "2020-09-21", "2021-01-18"}
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
	sp, err := NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4029", "0"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"CPC", "Copernic", "0:SP:80:4028", "1"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4142", "0"})
	require.Nil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	sp, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141", "1"})
	require.Nil(err)
	_, err = NewStopPoint([]string{"PTR", "Pasteur", "0:SP:80:4141", "2"})
	assert.NotNil(err)
	stopPoints[sp.Name+strconv.Itoa(sp.Direction)] = *sp
	manager.InitStopPoint(stopPoints)
	assert.Equal(len(*manager.stopPoints), 4)

	// Load Courses
	courses := make(map[string][]Course)
	courseLine := []string{"40", "2774327", "1", "05:45:00", "06:06:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err := NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	courseLine = []string{"40", "2774327", "2", "05:46:00", "06:07:00", "13",
		"2020-09-21", "2021-01-18", "2021-01-22 13:27:13.066197+00"}
	course, err = NewCourse(courseLine, location)
	require.Nil(err)
	courses[course.LineCode] = append(courses[course.LineCode], *course)
	manager.InitCourse(courses)
	assert.Equal(len(*manager.courses), 1)

	// Load RouteSchedules
	routeSchedules := make([]RouteSchedule, 0)
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
	pn := data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    0,
		StopName: "Copernic",
		Charge:   55}
	p := NewPrediction(pn, location)
	require.NotNil(p)
	predictions = append(predictions, *p)
	pn = data.PredictionNode{
		Line:     "40",
		Sens:     0,
		Date:     "2021-02-22T00:00:00",
		Course:   "2774327",
		Order:    1,
		StopName: "Pasteur",
		Charge:   75}
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
	require.Nil(err)
	assert.Equal(len(vehicleOccupancies), 2)

	// Call Api with StopId in the paraameter
	param = VehicleOccupancyRequestParameter{
		StopId:           "stop_point:0:SP:80:4029",
		VehicleJourneyId: "",
		Date:             date}
	vehicleOccupancies, err = manager.GetVehicleOccupancies(&param)
	require.Nil(err)
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
	require.Nil(err)
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
