package sytralrt

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sftpPort string
var fixtureDir string

const oneline = "1;87A;Mions Bourdelle;11 min;E;2018-09-17 20:28:00;35998;87A-022AM:5:2:12\r\n"

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
		Cmd:        []string{"sytral:pass"},
		Mounts:     []string{fmt.Sprintf("%s:/home/sytral", fixtureDir)},
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

func TestGetSFTPFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://sytral:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	reader, err := getFileWithSftp(*uri)
	require.Nil(t, err)
	b := make([]byte, 100)
	len, err := reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

	reader, err = getFile(*uri)
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

	reader, err = getFile(*uri)
	assert.Nil(t, err)
	len, err = reader.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, oneline, string(b[0:len]))

}

func TestGetSFTPFileError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	uri, err := url.Parse(fmt.Sprintf("sftp://sytral:wrongpass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri)
	require.Error(t, err)

	_, err = getFile(*uri)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://monuser:pass@localhost:%s/oneline.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri)
	require.Error(t, err)
	_, err = getFile(*uri)
	require.Error(t, err)

	uri, err = url.Parse(fmt.Sprintf("sftp://sytral:pass@localhost:%s/not.txt", sftpPort))
	require.Nil(t, err)
	_, err = getFileWithSftp(*uri)
	require.Error(t, err)
	_, err = getFile(*uri)
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
	err = RefreshDepartures(&manager, *firstURI)
	assert.Nil(t, err)
	departures, err := manager.GetDeparturesByStop("3")
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *secondURI)
	assert.Nil(t, err)
	departures, err = manager.GetDeparturesByStop("3")
	require.Nil(t, err)
	checkSecond(t, departures)
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

	reader, err := getFile(*misssingFieldURI)
	require.Nil(t, err)

	err = LoadData(reader, makeDepartureLineConsumer())
	require.Error(t, err)

	err = RefreshDepartures(&manager, *firstURI)
	require.Nil(t, err)
	departures, err := manager.GetDeparturesByStop("3")
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *misssingFieldURI)
	require.Error(t, err)
	//data hasn't been updated
	departures, err = manager.GetDeparturesByStop("3")
	require.Nil(t, err)
	checkFirst(t, departures)

	err = RefreshDepartures(&manager, *secondURI)
	require.Nil(t, err)
	departures, err = manager.GetDeparturesByStop("3")
	require.Nil(t, err)
	checkSecond(t, departures)

	reader, err = getFile(*invalidDateURI)
	require.Nil(t, err)
	err = LoadData(reader, makeDepartureLineConsumer())
	require.Error(t, err)

	err = RefreshDepartures(&manager, *invalidDateURI)
	require.Error(t, err)
	departures, err = manager.GetDeparturesByStop("3")
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

	consumer := makeEquipmentLineConsumer()
	err = LoadXmlData(reader, consumer)
	require.Nil(err)

	eds := consumer.equipments
	assert.Len(eds, 3)
	require.Contains(eds, "821")

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	ed := eds["821"]
	assert.Equal("821", ed.ID)
	assert.Equal("elevator", ed.EmbeddedType)
	assert.Equal("direction Gare de Vaise, accès Gare Routière ou Parc Relais", ed.Name)
	assert.Equal("Problème technique", ed.CurrentAvailability.Cause.Label)
	assert.Equal("Accès impossible direction Gare de Vaise.", ed.CurrentAvailability.Effect.Label)
	assert.Equal(time.Date(2018, 9, 14, 0, 0, 0, 0, location), ed.CurrentAvailability.Periods.Begin)
	assert.Equal(time.Date(2018, 9, 14, 13, 0, 0, 0, location), ed.CurrentAvailability.Periods.End)
}
