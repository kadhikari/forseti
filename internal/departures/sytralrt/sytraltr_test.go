package sytralrt

import (
	"encoding/json"
	"fmt"
	"os"

	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

var fixtureDir string
var defaultTimeout time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestDeparturesApi(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	multipleURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(err)

	departuresContext := &departures.DeparturesContext{}

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	departures.AddDeparturesEntryPoint(router, departuresContext)

	{
		c.Request = httptest.NewRequest("GET", "/departures", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, c.Request)
		require.Equal(400, w.Code)
		response := departures.DeparturesResponse{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.Nil(err)
		assert.Nil(response.Departures)
		assert.NotEmpty(response.Message)
	}

	{
		c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, c.Request)
		require.Equal(503, w.Code)
		response := departures.DeparturesResponse{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.Nil(err)
		assert.Nil(response.Departures)
		assert.NotEmpty(response.Message)
	}

	// We load some data
	{
		sytralRtContext := &SytralRTContext{}
		sytralRtContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
		err = RefreshDepartures(sytralRtContext, departuresContext)
		assert.Nil(err)

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 4)
		}

		// these is no stop 5 in our dataset
		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=5", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			require.Empty(response.Departures)
		}
	}

	//load data with more than one stops
	{
		sytralContext := &SytralRTContext{}
		sytralContext.InitContext(*multipleURI, defaultTimeout, defaultTimeout)
		err = RefreshDepartures(sytralContext, departuresContext)
		assert.Nil(err)
		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 8)
			for _, departure := range *response.Departures {
				require.True(
					departure.DirectionType == departures.DirectionTypeForward ||
						departure.DirectionType == departures.DirectionTypeBackward,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=both", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 8)
			for _, departure := range *response.Departures {
				require.True(
					departure.DirectionType == departures.DirectionTypeForward ||
						departure.DirectionType == departures.DirectionTypeBackward,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=forward", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 4)
			for _, departure := range *response.Departures {
				require.Equal(
					departures.DirectionTypeForward,
					departure.DirectionType,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=outbound", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 4)
			for _, departure := range *response.Departures {
				require.Equal(
					departures.DirectionTypeForward,
					departure.DirectionType,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=backward", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 2)
			for _, departure := range *response.Departures {
				require.Equal(
					departures.DirectionTypeBackward,
					departure.DirectionType,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=inbound", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(200, w.Code)

			response := departures.DeparturesResponse{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.Nil(err)
			assert.Empty(response.Message)
			require.NotNil(response.Departures)
			assert.NotEmpty(response.Departures)
			assert.Len(*response.Departures, 2)
			for _, departure := range *response.Departures {
				require.Equal(
					departures.DirectionTypeBackward,
					departure.DirectionType,
				)
			}
		}

		{
			c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=aller", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, c.Request)
			require.Equal(400, w.Code)
		}
	}
}

func TestLoadDepartureData(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("file://%s/oneline.txt", fixtureDir))
	require.Nil(t, err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(t, err)

	consumer := makeDepartureLineConsumer()
	departures := consumer.Data
	err = utils.LoadData(reader, consumer)
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
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(t, err)

	consumer := makeDepartureLineConsumer()
	departures := consumer.Data
	err = utils.LoadData(reader, consumer)
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

func checkFirst(t *testing.T, departures []departures.Departure) {
	require.Len(t, departures, 4)
	assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures[0].Datetime.String())
	assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures[1].Datetime.String())
	assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[2].Datetime.String())
	assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[3].Datetime.String())
}

func checkSecond(t *testing.T, departures []departures.Departure) {
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

	departuresContext := &departures.DeparturesContext{}
	{
		syntralContext := &SytralRTContext{}
		syntralContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
		err = RefreshDepartures(syntralContext, departuresContext)
		assert.Nil(t, err)
		departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
		require.Nil(t, err)
		checkFirst(t, departures)
	}

	{
		syntralContext := &SytralRTContext{}
		syntralContext.InitContext(*secondURI, defaultTimeout, defaultTimeout)
		err = RefreshDepartures(syntralContext, departuresContext)
		assert.Nil(t, err)
		departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
		require.Nil(t, err)
		checkSecond(t, departures)
	}
}

func TestMultipleStopsID(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(t, err)

	syntralContext := &SytralRTContext{}
	syntralContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
	departuresContext := &departures.DeparturesContext{}
	err = RefreshDepartures(syntralContext, departuresContext)
	assert.Nil(t, err)
	departures, err := departuresContext.GetDeparturesByStops([]string{"3", "4"})
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

	departures, err = departuresContext.GetDeparturesByStops([]string{"3", "832813923", "4"})
	require.Nil(t, err)
	require.Len(t, departures, 8)
}

func TestByDirectionType(t *testing.T) {
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(t, err)

	syntralContext := &SytralRTContext{}
	syntralContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
	departuresContext := &departures.DeparturesContext{}
	err = RefreshDepartures(syntralContext, departuresContext)
	assert.Nil(t, err)
	{
		departures, err := departuresContext.GetDeparturesByStopsAndDirectionType(
			[]string{"3", "4"}, departures.DirectionTypeForward,
		)
		require.Nil(t, err)
		require.Len(t, departures, 4)
		assert.Equal(t, "2018-09-17 20:38:37 +0200 CEST", departures[0].Datetime.String())
		assert.Equal(t, "2018-09-17 20:39:37 +0200 CEST", departures[1].Datetime.String())
		assert.Equal(t, "2018-09-17 21:01:55 +0200 CEST", departures[2].Datetime.String())
		assert.Equal(t, "2018-09-17 21:02:55 +0200 CEST", departures[3].Datetime.String())
	}

	{
		departures, err := departuresContext.GetDeparturesByStopsAndDirectionType(
			[]string{"3"}, departures.DirectionTypeBackward,
		)
		require.Nil(t, err)
		require.Len(t, departures, 2)
		assert.Equal(t, "2018-09-17 20:28:37 +0200 CEST", departures[0].Datetime.String())
		assert.Equal(t, "2018-09-17 20:52:55 +0200 CEST", departures[1].Datetime.String())
	}

	{
		departures, err := departuresContext.GetDeparturesByStopsAndDirectionType(
			[]string{"5"}, departures.DirectionTypeForward,
		)
		require.Nil(t, err)
		require.Len(t, departures, 4)
	}

	{
		departures, err := departuresContext.GetDeparturesByStopsAndDirectionType(
			[]string{"5"}, departures.DirectionTypeBackward,
		)
		require.Nil(t, err)
		require.Len(t, departures, 0)
	}
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

	departuresContext := &departures.DeparturesContext{}

	{
		reader, err := utils.GetFile(*misssingFieldURI, defaultTimeout)
		require.Nil(t, err)

		err = utils.LoadData(reader, makeDepartureLineConsumer())
		require.Error(t, err)

		{
			syntralContext := &SytralRTContext{}
			syntralContext.InitContext(*firstURI, defaultTimeout, defaultTimeout)
			err := RefreshDepartures(syntralContext, departuresContext)
			require.Nil(t, err)
			departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
			require.Nil(t, err)
			checkFirst(t, departures)
		}

		{
			syntralContext := &SytralRTContext{}
			syntralContext.InitContext(*misssingFieldURI, defaultTimeout, defaultTimeout)
			err := RefreshDepartures(syntralContext, departuresContext)
			require.Error(t, err)
			//data hasn't been updated
			departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
			require.Nil(t, err)
			checkFirst(t, departures)
		}

		{
			syntralContext := &SytralRTContext{}
			syntralContext.InitContext(*secondURI, defaultTimeout, defaultTimeout)
			err := RefreshDepartures(syntralContext, departuresContext)
			require.Nil(t, err)
			departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
			require.Nil(t, err)
			checkSecond(t, departures)
		}
	}

	{
		reader, err := utils.GetFile(*invalidDateURI, defaultTimeout)
		require.Nil(t, err)
		err = utils.LoadData(reader, makeDepartureLineConsumer())
		require.Error(t, err)
		{
			syntralContext := &SytralRTContext{}
			syntralContext.InitContext(*invalidDateURI, defaultTimeout, defaultTimeout)
			err := RefreshDepartures(syntralContext, departuresContext)
			require.Error(t, err)
			departures, err := departuresContext.GetDeparturesByStops([]string{"3"})
			require.Nil(t, err)
			//we still get old departures
			checkSecond(t, departures)
		}
	}
}

func TestNewDeparture(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	d, err := newDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3"}, location)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	assert.Equal(departures.DirectionTypeUnknown, d.DirectionType)

	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)
}

func TestNewDepartureWithDirectionType(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	d, err := newDeparture(
		[]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3", "vjid", "said", "ALL"},
		location,
	)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	assert.Equal(departures.DirectionTypeForward, d.DirectionType)

	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)

	d, err = newDeparture(
		[]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00", "3", "vjid", "said", "RET"},
		location,
	)
	require.Nil(err)

	assert.Equal("1", d.Stop)
	assert.Equal("2", d.Line)
	assert.Equal("dest", d.DirectionName)
	assert.Equal("3", d.Direction)
	assert.Equal("E", d.Type)
	assert.Equal(departures.DirectionTypeBackward, d.DirectionType)

	//Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location)
	assert.Equal(time.Date(2018, 9, 17, 20, 28, 0, 0, location), d.Datetime)
}

func TestNewDepartureMissingField(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	_, err = newDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28:00"}, location)
	require.Error(err)
}

func TestNewDepartureInvalidDate(t *testing.T) {
	require := require.New(t)
	location, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	_, err = newDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 20:28", "3"}, location)
	require.Error(err)

	_, err = newDeparture([]string{"1", "2", "dest", "", "E", "2018-09-17 25:28:00", "3"}, location)
	require.Error(err)
}

func TestDataManagerLastUpdate(t *testing.T) {
	require := require.New(t)
	begin := time.Now()

	departuresContext := &departures.DeparturesContext{}
	departuresContext.UpdateDepartures(nil)

	lastDepartureUpdate := departuresContext.GetLastDepartureDataUpdate()
	require.True(lastDepartureUpdate.After(begin))

	departuresContext.UpdateDepartures(nil)
	require.True(departuresContext.GetLastDepartureDataUpdate().After(lastDepartureUpdate))
}

func TestParseDirectionType(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(departures.DirectionTypeForward, departures.ParseDirectionType("ALL"))
	assert.Equal(departures.DirectionTypeBackward, departures.ParseDirectionType("RET"))
	assert.Equal(departures.DirectionTypeUnknown, departures.ParseDirectionType(""))
	assert.Equal(departures.DirectionTypeUnknown, departures.ParseDirectionType("foo"))
}
