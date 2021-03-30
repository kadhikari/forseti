package forseti

import (
	"encoding/json"
	"flag"
	"fmt"

	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
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
	_, err = utils.GetFileWithSftp(*uri, time.Microsecond)
	require.NotNil(t, err)
}
func TestGetSFTPFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://forseti:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	reader, err := utils.GetFileWithSftp(*uri, defaultTimeout)
	require.Nil(t, err)
	b := make([]byte, 100)
	len, err := reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

	reader, err = utils.GetFile(*uri, defaultTimeout)
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
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(t, err)
	b := make([]byte, 100)
	len, err := reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

	reader, err = utils.GetFile(*uri, defaultTimeout)
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
	_, err = utils.GetFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)

	_, err = utils.GetFile(*uri, defaultTimeout)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://monuser:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = utils.GetFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)
	_, err = utils.GetFile(*uri, defaultTimeout)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://forseti:pass@localhost:%s/not.txt", sftpPort))
	require.Nil(t, err)
	_, err = utils.GetFileWithSftp(*uri, defaultTimeout)
	require.Error(t, err)
	_, err = utils.GetFile(*uri, defaultTimeout)
	require.Error(t, err)
}

func TestLoadParkingData(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/parkings.txt", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	consumer := makeParkingLineConsumer()
	err = utils.LoadDataWithOptions(reader, consumer, utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,
		SkipFirstLine: true,
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

func TestLoadStopPointsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// FileName for StopPoints should be mapping_stops.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	stopPoints, err := LoadStopPoints(*uri, defaultTimeout)
	require.Nil(err)
	assert.Equal(len(stopPoints), 25)
	stopPoint := stopPoints["Copernic0"]
	assert.Equal(stopPoint.Name, "Copernic")
	assert.Equal(stopPoint.Direction, 0)
	assert.Equal(stopPoint.Id, "stop_point:0:SP:80:4029")
}

func TestLoadCoursesFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	// FileName for StopPoints should be extraction_courses.csv
	uri, err := url.Parse(fmt.Sprintf("file://%s/", fixtureDir))
	require.Nil(err)
	courses, err := LoadCourses(*uri, defaultTimeout)
	require.Nil(err)
	// It's map size (line_code=40)
	assert.Equal(len(courses), 1)
	course40 := courses["40"]
	assert.Equal(len(course40), 310)
	assert.Equal(course40[0].LineCode, "40")
	assert.Equal(course40[0].Course, "2774327")
	assert.Equal(course40[0].DayOfWeek, 1)
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
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	navitiaRoutes := &data.NavitiaRoutes{}
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
	assert.Equal(routeSchedules[0].Direction, 0)
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
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	predicts := &data.PredictionData{}
	err = json.Unmarshal([]byte(jsonData), predicts)
	require.Nil(err)
	predictions := LoadPredictionsData(predicts, location)
	assert.Equal(len(predictions), 65)
	assert.Equal(predictions[0].LineCode, "40")
	assert.Equal(predictions[0].Order, 0)
	assert.Equal(predictions[0].Direction, 0)
	assert.Equal(predictions[0].Date, time.Date(2021, 1, 18, 0, 0, 0, 0, location))
	assert.Equal(predictions[0].Course, "2774327")
	assert.Equal(predictions[0].StopName, "Pont de Sevres")
	assert.Equal(predictions[0].Occupancy, 12)
}
