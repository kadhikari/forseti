package parkings

import (
	"encoding/json"

	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CanalTP/forseti/internal/utils"
)

var fixtureDir string

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestParkingsPRAPIwithParkingsID(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	AddParkingsEntryPoint(router, &parkingsContext)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R?ids[]=donald&ids[]=picsou&ids[]=fifi", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(http.StatusOK, w.Code)

	response := ParkingsResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	parkings := response.Parkings
	sort.Sort(ByParkingResponseId(parkings))
	require.Len(parkings, 2)
	assert.Equal("Donald", parkings[0].ID)
	assert.Equal("Fifi", parkings[1].ID)

	require.Len(response.Errors, 1)
	assert.Contains(response.Errors[0], "picsou")
}

func TestParkingsPRAPI(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	AddParkingsEntryPoint(router, &parkingsContext)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(http.StatusOK, w.Code)

	response := ParkingsResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	parkings := response.Parkings
	sort.Sort(ByParkingResponseId(parkings))
	require.Len(parkings, 4)
	require.Len(response.Errors, 0)
	assert.Equal("Donald", parkings[0].ID)
	assert.Equal("Fifi", parkings[1].ID)
	assert.Equal("Loulou", parkings[2].ID)
	assert.Equal("Riri", parkings[3].ID)
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

func TestGetParkingById(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"toto": {"DECC", "Décines Centre", updateTime, 1, 2, 3, 4},
	})

	p, err := parkingsContext.GetParkingById("toto")
	assert.Nil(err)
	assert.Equal("DECC", p.ID)
	assert.Equal("Décines Centre", p.Label)
	assert.Equal(time.Date(2018, 9, 17, 19, 29, 0, 0, loc), p.UpdatedTime)
	assert.Equal(1, p.AvailableStandardSpaces)
	assert.Equal(2, p.AvailableAccessibleSpaces)
	assert.Equal(3, p.TotalStandardSpaces)
	assert.Equal(4, p.TotalAccessibleSpaces)
}

func TestShouldErrorOnEmptyData(t *testing.T) {
	assert := assert.New(t)

	var parkingsContext ParkingsContext
	p, err := parkingsContext.GetParkingById("toto")
	assert.Error(err)
	assert.Empty(p)
}

func TestShouldErrorOnEmptyData2(t *testing.T) {
	assert := assert.New(t)

	var parkingsContext ParkingsContext
	p, errs := parkingsContext.GetParkingsByIds([]string{"toto", "titi"})
	assert.NotEmpty(errs)
	assert.Empty(p)
	for _, e := range errs {
		assert.Error(e)
	}
}
func TestGetParkingByIds(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	p, errs := parkingsContext.GetParkingsByIds([]string{"riri", "loulou"})
	assert.Nil(errs)
	require.Len(p, 2)
	assert.Equal("Riri", p[0].ID)
	assert.Equal("Loulou", p[1].ID)
}

func TestPartiallyGetParkingByIds(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	p, errs := parkingsContext.GetParkingsByIds([]string{"fifi", "donald"})
	require.Len(p, 1)
	assert.Equal("Fifi", p[0].ID)

	require.Len(errs, 1)
	assert.Error(errs[0])
}

func TestGetParking(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)
	updateTime, err := time.ParseInLocation("2006-01-02 15:04:05", "2018-09-17 19:29:00", loc)
	require.Nil(err)

	var parkingsContext ParkingsContext
	parkingsContext.UpdateParkings(map[string]Parking{
		"riri":   {"Riri", "First of the name", updateTime, 1, 2, 3, 4},
		"fifi":   {"Fifi", "Second of the name", updateTime, 1, 2, 3, 4},
		"loulou": {"Loulou", "Third of the name", updateTime, 1, 2, 3, 4},
	})

	parkings, err := parkingsContext.GetParkings()
	require.Nil(err)
	require.Len(parkings, 3)

	sort.Sort(ByParkingId(parkings))
	assert.Equal("Fifi", parkings[0].ID)
	assert.Equal("Loulou", parkings[1].ID)
	assert.Equal("Riri", parkings[2].ID)
}
