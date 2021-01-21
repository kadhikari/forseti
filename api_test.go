package forseti

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"
	"io/ioutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeparturesApi(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	multipleURI, err := url.Parse(fmt.Sprintf("file://%s/multiple.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/departures", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(400, w.Code)
	response := DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Departures)
	assert.NotEmpty(response.Message)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)
	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Nil(response.Departures)
	assert.NotEmpty(response.Message)

	//we load some data
	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 4)

	//these is no stop 5 in our dataset
	c.Request = httptest.NewRequest("GET", "/departures?stop_id=5", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	require.Empty(response.Departures)

	//load data with more than one stops
	err = RefreshDepartures(&manager, *multipleURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 8)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=both", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 8)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&stop_id=4&direction_type=forward", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 4)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=backward", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	response = DeparturesResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Empty(response.Message)
	require.NotNil(response.Departures)
	assert.NotEmpty(response.Departures)
	assert.Len(*response.Departures, 2)

	c.Request = httptest.NewRequest("GET", "/departures?stop_id=3&direction_type=aller", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(400, w.Code)
}

func TestStatusApiExist(t *testing.T) {
	require := require.New(t)
	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	require.Equal(200, w.Code)
}

func TestStatusApiHasLastUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshDepartures(&manager, *firstURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Equal(response.Status, "ok")
	assert.True(response.LastDepartureUpdate.After(startTime))
	assert.True(response.LastDepartureUpdate.Before(time.Now()))
}

func TestStatusApiHasLastParkingUpdateTime(t *testing.T) {
	startTime := time.Now()
	assert := assert.New(t)
	require := require.New(t)
	parkingURI, err := url.Parse(fmt.Sprintf("file://%s/parkings.txt", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshParkings(&manager, *parkingURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.True(response.LastParkingUpdate.After(startTime))
	assert.True(response.LastParkingUpdate.Before(time.Now()))
}

func TestParkingsPRAPI(t *testing.T) {
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
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
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

func TestParkingsPRAPIwithParkingsID(t *testing.T) {
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
		"donald": {"Donald", "Donald THE Duck", updateTime, 1, 2, 3, 4},
	})

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	c.Request = httptest.NewRequest("GET", "/parkings/P+R?ids[]=donald&ids[]=picsou&ids[]=fifi", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
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

func TestEquipmentsAPI(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	equipmentURI, err := url.Parse(fmt.Sprintf("file://%s/NET_ACCESS.XML", fixtureDir))
	require.Nil(err)

	var manager DataManager

	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	err = RefreshEquipments(&manager, *equipmentURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/equipments", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response EquipmentsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.Equipments)
	require.NotEmpty(response.Equipments)
	assert.Len(response.Equipments, 3)
	assert.Empty(response.Error)
}

func TestFreeFloatingsAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Load freefloatings from a json file
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

	var manager DataManager
	manager.UpdateFreeFloating(freeFloatings)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	engine = SetupRouter(&manager, engine)

	// Request without any parameter (coord is mandatory)
	c.Request = httptest.NewRequest("GET", "/free_floatings", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)

	var response FreeFloatingsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Len(response.FreeFloatings, 0)
	assert.Equal("Bad request: coord is mandatory",response.Error)

	// Request with coord in parameter
	response.Error = ""
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 3)
	assert.Empty(response.Error)

	// Request with coord, count in parameter
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=2", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 2)
	assert.Empty(response.Error)

	// Request with coord, type[] in parameter
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&type[]=BIKE&type[]=toto", nil)
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 1)
}

func TestParameterTypes(t *testing.T) {
	// valid types : {"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER"}
	// As toto is not a valid type it will not be added in types
	assert := assert.New(t)
	p := Parameter{}
	types := make([]string, 0)
	types = append(types, "STATION")
	types = append(types, "toto")
	types = append(types, "MOTORSCOOTER")
	types = append(types, "OTHER")

	updateParameterTypes(&p, types)
	assert.Len(p.types, 3)
}
