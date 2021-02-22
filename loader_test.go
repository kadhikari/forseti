package forseti

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"
	"io/ioutil"
	"encoding/json"
	"time"
	"github.com/ory/dockertest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sftpPort string
var fixtureDir string

const oneline = "1;87A;Mions Bourdelle;11 min;E;2018-09-17 20:28:00;35998;87A-022AM:5:2:12\r\n"

var defaultTimeout time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	flag.Parse() //required to get Short() from testing
	if testing.Short() {
		log.Warn("skipping test Docker in short mode.")
		os.Exit(m.Run())
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	opts := dockertest.RunOptions{
		Repository: "atmoz/sftp",
		Tag:        "alpine",
		Cmd:        []string{"forseti:pass"},
		Mounts:     []string{fmt.Sprintf("%s:/home/forseti", fixtureDir)},
	}
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	//lets wait a bit for the docker to start :(
	time.Sleep(3 * time.Second)
	sftpPort = resource.GetPort("22/tcp")
	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestGetSFTPFileTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://forseti:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri, time.Microsecond)
	require.NotNil(t, err)
}
func TestGetSFTPFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://forseti:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	reader, err := getFileWithSftp(*uri, defaultTimeout)
	require.Nil(t, err)
	b := make([]byte, 100)
	len, err := reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

	reader, err = getFile(*uri, defaultTimeout)
	assert.Nil(t, err)
	len, err = reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))
}

func TestGetSFTPFS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("file://%s/oneline.txt", fixtureDir))
	require.Nil(t, err)
	reader, err := getFileWithFS(*uri)
	require.Nil(t, err)
	b := make([]byte, 100)
	len, err := reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

	reader, err = getFile(*uri, defaultTimeout)
	assert.Nil(t, err)
	len, err = reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

}

func TestGetSFTPFileError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://forseti:wrongpass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)

	_, err = getFile(*uri, defaultTimeout)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://monuser:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)
	_, err = getFile(*uri, defaultTimeout)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://forseti:pass@localhost:%s/not.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)
	_, err = getFile(*uri, defaultTimeout)
	require.Error(t, err)
}

func TestLoadDepartureData(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("file://%s/oneline.txt", fixtureDir))
	require.Nil(t, err)
	reader, err := getFileWithFS(*uri)
	require.Nil(t, err)

	consumer := makeDepartureLineConsumer()
	departures := consumer.data
	err = LoadData(reader, consumer)
	require.Nil(t, err)
	assert.Len(t, departures, 1)

	require.Contains(t, departures, "1")
	require.Len(t, departures["1"], 1)

	d := departures["1"][0]
	assert.Equal(t, "1", d.Stop)
	assert.Equal(t, "87A", d.Line)
	assert.Equal(t, "E", d.Type)
	assert.Equal(t, "35998", d.Direction)
	assert.Equal(t, "Mions Bourdelle", d.DirectionName)
	assert.Equal(t, "2018-09-17 20:28:00 +0200 CEST", d.Datetime.String())
}

func TestLoadFull(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("file://%s/extract_edylic.txt", fixtureDir))
	require.Nil(t, err)
	reader, err := getFileWithFS(*uri)
	require.Nil(t, err)

	consumer := makeDepartureLineConsumer()
	departures := consumer.data
	err = LoadData(reader, consumer)
	require.Nil(t, err)
	assert.Len(t, departures, 347)

	require.Contains(t, departures, "3")
	require.Len(t, departures["3"], 4)

	//lets check that the sort is OK
	assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures["3"][0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures["3"][1].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures["3"][2].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures["3"][3].Datetime.String())
}

func TestLoadParkingData(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/parkings.txt", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	consumer := makeParkingLineConsumer()
	err = LoadDataWithOptions(reader, consumer, LoadDataOptions{
		delimiter:     ';',
		nbFields:      0,
		skipFirstLine: true,
	})
	require.Nil(err)

	parkings := consumer.parkings
	assert.Len(parkings, 19)
	require.Contains(parkings, "DECC")

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	p := parkings["DECC"]
	assert.Equal("DECC", p.ID)
	assert.Equal("Décines Centre", p.Label)
	assert.Equal(time.Date(2018, 9, 17, 19, 29, 0, 0, location), p.UpdatedTime)
	assert.Equal(82, p.AvailableStandardSpaces)
	assert.Equal(105, p.TotalStandardSpaces)
	assert.Equal(0, p.AvailableAccessibleSpaces)
	assert.Equal(3, p.TotalAccessibleSpaces)
}

func checkFirst(t *testing.T, departures []Departure) {
	require.Len(t, departures, 4)
	assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures[1].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[2].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[3].Datetime.String())
}

func checkSecond(t *testing.T, departures []Departure) {
	require.Len(t, departures, 3)
	assert.Equal(t, "2018-09-17 20:42:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[1].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[2].Datetime.String())
}

func TestRefreshData(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(t, err)
	secondURI, err := url.Parse(fmt.Sprintf("file://%s/second.txt", fixtureDir))
	require.Nil(t, err)

	var manager DataManager
	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(t, err)
	departures, err := manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *secondURI, defaultTimeout)
	assert.Nil(t, err)
	departures, err = manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	checkSecond(t, departures)
}

func TestMultipleStopsID(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(t, err)

	var manager DataManager
	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(t, err)
	departures, err := manager.GetDeparturesByStops([]string{"3", "4"})
	require.Nil(t, err)
	require.Len(t, departures, 8)
	assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:32:37 +0200 CEST", departures[1].Datetime.String())
	assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures[2].Datetime.String())
	assert.Equal(t, "2018-09-17 20:39:37 +0200 CEST", departures[3].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[4].Datetime.String())
	assert.Equal(t, "2018-09-17 20:55:55 +0200 CEST", departures[5].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[6].Datetime.String())
	assert.Equal(t, "2018-09-17 21:02:55 +0200 CEST", departures[7].Datetime.String())

	departures, err = manager.GetDeparturesByStops([]string{"3", "832813923", "4"})
	require.Nil(t, err)
	require.Len(t, departures, 8)
}

func TestByDirectionType(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(t, err)

	var manager DataManager
	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(t, err)
	departures, err := manager.GetDeparturesByStopsAndDirectionType([]string{"3", "4"}, DirectionTypeForward)
	require.Nil(t, err)
	require.Len(t, departures, 4)
	assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:39:37 +0200 CEST", departures[1].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[2].Datetime.String())
	assert.Equal(t, "2018-09-17 21:02:55 +0200 CEST", departures[3].Datetime.String())

	departures, err = manager.GetDeparturesByStopsAndDirectionType([]string{"3"}, DirectionTypeBackward)
	require.Nil(t, err)
	require.Len(t, departures, 2)
	assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[1].Datetime.String())

	departures, err = manager.GetDeparturesByStopsAndDirectionType([]string{"5"}, DirectionTypeForward)
	require.Nil(t, err)
	require.Len(t, departures, 4)

	departures, err = manager.GetDeparturesByStopsAndDirectionType([]string{"5"}, DirectionTypeBackward)
	require.Nil(t, err)
	require.Len(t, departures, 0)
}

func TestRefreshDataError(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(t, err)

	misssingFieldURI, err := url.Parse(fmt.Sprintf("file://%s/missingfield.txt", fixtureDir))
	require.Nil(t, err)

	invalidDateURI, err := url.Parse(fmt.Sprintf("file://%s/invaliddate.txt", fixtureDir))
	require.Nil(t, err)

	secondURI, err := url.Parse(fmt.Sprintf("file://%s/second.txt", fixtureDir))
	require.Nil(t, err)

	var manager DataManager

	reader, err := getFile(*misssingFieldURI, defaultTimeout)
	require.Nil(t, err)

	err = LoadData(reader, makeDepartureLineConsumer())
	require.Error(t, err)

	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	require.Nil(t, err)
	departures, err := manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *misssingFieldURI, defaultTimeout)
	require.Error(t, err)
	//data hasn't been updated
	departures, err = manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *secondURI, defaultTimeout)
	require.Nil(t, err)
	departures, err = manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	checkSecond(t, departures)

	reader, err = getFile(*invalidDateURI, defaultTimeout)
	require.Nil(t, err)
	err = LoadData(reader, makeDepartureLineConsumer())
	require.Error(t, err)

	err = RefreshDepartures(&manager, *invalidDateURI, defaultTimeout)
	require.Error(t, err)
	departures, err = manager.GetDeparturesByStops([]string{"3"})
	require.Nil(t, err)
	//we still get old departures
	checkSecond(t, departures)
}

func TestLoadEquipmentsData(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/NET_ACCESS.XML", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	eds, err := LoadXmlData(reader)
	require.Nil(err)

	assert.Len(eds, 3)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	var ed EquipmentDetail
	for _, e := range eds {
		if e.ID == "821" {
			ed = e
			break
		}
	}
	assert.Equal("821", ed.ID)
	assert.Equal("elevator", ed.EmbeddedType)
	assert.Equal("direction Gare de Vaise, accès Gare Routière ou Parc Relais", ed.Name)
	assert.Equal("Problème technique", ed.CurrentAvailability.Cause.Label)
	assert.Equal("available", ed.CurrentAvailability.Status)
	assert.Equal("Accès impossible direction Gare de Vaise.", ed.CurrentAvailability.Effect.Label)
	assert.Equal(time.Date(2018, 9, 14, 0, 0, 0, 0, location), ed.CurrentAvailability.Periods[0].Begin)
	assert.Equal(time.Date(2018, 9, 14, 13, 0, 0, 0, location), ed.CurrentAvailability.Periods[0].End)
	assert.Equal(time.Date(2018, 9, 15, 12, 1, 31, 0, location), ed.CurrentAvailability.UpdatedAt)
}

func TestLoadFreeFloatingsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicles.json", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	data := &Data{}
	err = json.Unmarshal([]byte(jsonData), data)
	require.Nil(err)

	freeFloatings, err := LoadFreeFloatingData(data)

	require.Nil(err)
	assert.NotEqual(len(freeFloatings), 0)
	assert.Len(freeFloatings, 3)
	assert.NotEmpty(freeFloatings[0].Id)
	assert.Equal("NSCBH3", freeFloatings[0].PublicId)
	assert.Equal("Pony", freeFloatings[0].ProviderName)
	assert.Equal("cG9ueTpCSUtFOjEwMDQ0MQ==", freeFloatings[0].Id)
	assert.Equal("BIKE", freeFloatings[0].Type)
	assert.Equal("ASSIST", freeFloatings[0].Propulsion)
	assert.Equal(55, freeFloatings[0].Battery)
	assert.Equal("http://test1", freeFloatings[0].Deeplink)
	assert.Equal(48.847232, freeFloatings[0].Coord.Lat)
	assert.Equal(2.377601, freeFloatings[0].Coord.Lon)
	assert.Equal([]string{"ELECTRIC"}, freeFloatings[0].Attributes)
}

func TestLoadStopPointsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// FileName for StopPoints should be mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Equal(len(stopPoints), 25)
	stopPoint := stopPoints["Copernic0"]
	assert.Equal(stopPoint.Name, "Copernic")
	assert.Equal(stopPoint.Sens, 0)
	assert.Equal(stopPoint.Id, "stop_point:0:SP:80:4029")
}

func TestLoadCoursesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// FileName for StopPoints should be extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	// It's map size (line_code=40)
	assert.Equal(len(courses), 1)
	course40 := courses["40"]
	assert.Equal(len(course40), 310)
	assert.Equal(course40[0].LineCode, "40")
	assert.Equal(course40[0].Course, "2774327")
	assert.Equal(course40[0].Dow, 1)
	time, err := time.ParseInLocation("15:04:05", "05:47:18", location)
	require.Nil(err)
	assert.Equal(course40[0].FirstTime, time)
}

func TestLoadRouteSchedulesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	uri, err := url.Parse(fmt.Sprintf("file://%s/route_schedules.json", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaRoutes := &NavitiaRoutes{}
	err = json.Unmarshal([]byte(jsonData), navitiaRoutes)
	require.Nil(err)
	sens := 0
	startIndex := 1
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, location)
	assert.Equal(len(routeSchedules), 141)
	assert.Equal(routeSchedules[0].Id, 1)
	assert.Equal(routeSchedules[0].LineCode, "40")
	assert.Equal(routeSchedules[0].VehicleJourneyId, "vehicle_journey:0:123713787-1")
	assert.Equal(routeSchedules[0].StopId, "stop_point:0:SP:80:4131")
	assert.Equal(routeSchedules[0].Sens, 0)
	assert.Equal(routeSchedules[0].Departure, true)
	assert.Equal(routeSchedules[0].DateTime, time.Date(2021, 1, 18, 06, 0, 0, 0, location))
}

func TestLoadPredictionsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	uri, err := url.Parse(fmt.Sprintf("file://%s/predictions.json", fixtureDir))
	require.Nil(err)
	reader, err := getFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, location)
	assert.Equal(len(predictions), 65)
	assert.Equal(predictions[0].LineCode, "40")
	assert.Equal(predictions[0].Order, 0)
	assert.Equal(predictions[0].Sens, 0)
	assert.Equal(predictions[0].Date, time.Date(2021, 1, 9, 0, 0, 0, 0, location))
	assert.Equal(predictions[0].Course, "2774327")
	assert.Equal(predictions[0].StopName, "Pont de Sevres")
	assert.Equal(predictions[0].Charge, 12)
}
