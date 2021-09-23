package freefloatings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CanalTP/forseti/internal/data"
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

func TestFreeFloatingsAPIWithDataFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	// Load freefloatings from a json file
	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicles.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	data := &data.Data{}
	err = json.Unmarshal([]byte(jsonData), data)
	require.Nil(err)

	freeFloatings, err := LoadFreeFloatingsData(data)
	require.Nil(err)

	freeFloatingsContext := &FreeFloatingsContext{}

	freeFloatingsContext.UpdateFreeFloating(freeFloatings)
	c, router := gin.CreateTestContext(httptest.NewRecorder())
	AddFreeFloatingsEntryPoint(router, freeFloatingsContext)

	// Request without any parameter (coord is mandatory)
	c.Request = httptest.NewRequest("GET", "/free_floatings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(503, w.Code)

	var response FreeFloatingsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Len(response.FreeFloatings, 0)
	assert.Equal("Bad request: coord is mandatory", response.Error)

	// Request with coord in parameter
	response.Error = ""
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 3)
	assert.Empty(response.Error)
	// Verify distances
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
	assert.Equal(71.0, response.FreeFloatings[1].Distance)
	assert.Equal(74.0, response.FreeFloatings[2].Distance)

	// Request with coord, count in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 2)
	assert.Empty(response.Error)
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
	assert.Equal(71.0, response.FreeFloatings[1].Distance)
	// Verify paginate
	assert.Equal(0, response.Paginate.Start_page)
	assert.Equal(2, response.Paginate.Items_on_page)
	assert.Equal(2, response.Paginate.Items_per_page)
	assert.Equal(3, response.Paginate.Total_result)

	// Request with coord, type[] in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&type[]=BIKE&type[]=toto", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 1)

	// Verify attributes
	assert.Equal("cG9ueTpCSUtFOjEwMDQ0MQ==", response.FreeFloatings[0].Id)
	assert.Equal("NSCBH3", response.FreeFloatings[0].PublicId)
	assert.Equal("http://test1", response.FreeFloatings[0].Deeplink)
	assert.Equal("Pony", response.FreeFloatings[0].ProviderName)
	assert.Equal([]string{"ELECTRIC"}, response.FreeFloatings[0].Attributes)
	assert.Equal(55, response.FreeFloatings[0].Battery)
	assert.Equal("ASSIST", response.FreeFloatings[0].Propulsion)
	assert.Equal(48.847232, response.FreeFloatings[0].Coord.Lat)
	assert.Equal(2.377601, response.FreeFloatings[0].Coord.Lon)
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
	// Verify paginate
	assert.Equal(0, response.Paginate.Start_page)
	assert.Equal(1, response.Paginate.Items_on_page)
	assert.Equal(1, response.Paginate.Total_result)

	// At last a test to verify distance of the only element.
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 1)
	assert.Empty(response.Error)
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
	// Verify paginate
	assert.Equal(0, response.Paginate.Start_page)
	assert.Equal(1, response.Paginate.Items_on_page)
	assert.Equal(1, response.Paginate.Items_per_page)
	assert.Equal(3, response.Paginate.Total_result)

	// Request with coord, count and start_page in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=2&start_page=0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 2)
	assert.Equal(60.0, response.FreeFloatings[0].Distance)
	assert.Equal(71.0, response.FreeFloatings[1].Distance)
	// Verify paginate
	assert.Equal(0, response.Paginate.Start_page)
	assert.Equal(2, response.Paginate.Items_on_page)
	assert.Equal(2, response.Paginate.Items_per_page)
	assert.Equal(3, response.Paginate.Total_result)

	// Request with coord, count and start_page in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&count=2&start_page=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 1)
	assert.Equal(74.0, response.FreeFloatings[0].Distance)
	// Verify paginate
	assert.Equal(1, response.Paginate.Start_page)
	assert.Equal(1, response.Paginate.Items_on_page)
	assert.Equal(2, response.Paginate.Items_per_page)
	assert.Equal(3, response.Paginate.Total_result)

	// Request with coord, count and start_page in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET",
		"/free_floatings?coord=2.37715%3B48.846781&distance=2&start_page=1&count=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.Nil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 0)
	// Verify paginate
	assert.Equal(1, response.Paginate.Start_page)
	assert.Equal(0, response.Paginate.Items_on_page)
	assert.Equal(1, response.Paginate.Items_per_page)
	assert.Equal(0, response.Paginate.Total_result)

	// Request with coord, count and start_page in parameter
	response = FreeFloatingsResponse{}
	c.Request = httptest.NewRequest("GET", "/free_floatings?coord=2.37715%3B48.846781&start_page=-1&count=1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.Nil(response.FreeFloatings)
	assert.Len(response.FreeFloatings, 0)
	// Verify paginate
	assert.Equal(-1, response.Paginate.Start_page)
	assert.Equal(0, response.Paginate.Items_on_page)
	assert.Equal(1, response.Paginate.Items_per_page)
	assert.Equal(3, response.Paginate.Total_result)
}

func TestLoadFreeFloatingsFromFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/vehicles.json", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	jsonData, err := ioutil.ReadAll(reader)
	require.Nil(err)

	data := &data.Data{}
	err = json.Unmarshal([]byte(jsonData), data)
	require.Nil(err)

	freeFloatings, err := LoadFreeFloatingsData(data)

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

func TestNewFreeFloating(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	provider := data.ProviderNode{Name: "Pony"}
	ff := data.Vehicle{PublicId: "NSCBH3", Provider: provider, Id: "cG9ueTpCSUtFOjEwMDQ0MQ", Type: "BIKE",
		Latitude: 45.180335, Longitude: 5.7069425, Propulsion: "ASSIST", Battery: 85, Deeplink: "http://test"}
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

	freeFloatingsContext := &FreeFloatingsContext{}

	p1 := data.ProviderNode{Name: "Pony"}
	p2 := data.ProviderNode{Name: "Tier"}

	vehicles := make([]data.Vehicle, 0)
	freeFloatings := make([]FreeFloating, 0)
	v := data.Vehicle{PublicId: "NSCBH3", Provider: p1, Id: "cG9ueTpCSUtFOjEwMDQ0MQ", Type: "BIKE",
		Latitude: 48.847232, Longitude: 2.377601, Propulsion: "ASSIST", Battery: 85, Deeplink: "http://test1"}
	vehicles = append(vehicles, v)
	v = data.Vehicle{PublicId: "718WSK", Provider: p1, Id: "cG9ueTpCSUtFOjEwMDQ0MQ==4b", Type: "SCOOTER",
		Latitude: 48.847299, Longitude: 2.37772, Propulsion: "ELECTRIC", Battery: 85, Deeplink: "http://test2"}
	vehicles = append(vehicles, v)
	v = data.Vehicle{PublicId: "0JT9J6", Provider: p2, Id: "cG9ueTpCSUtFOjEwMDQ0MQ==55c", Type: "SCOOTER",
		Latitude: 48.847326, Longitude: 2.377734, Propulsion: "ELECTRIC", Battery: 85, Deeplink: "http://test3"}
	vehicles = append(vehicles, v)

	for _, vehicle := range vehicles {
		freeFloatings = append(freeFloatings, *NewFreeFloating(vehicle))
	}
	freeFloatingsContext.UpdateFreeFloating(freeFloatings)
	// init parameters:
	types := make([]FreeFloatingType, 0)
	types = append(types, StationType)
	types = append(types, ScooterType)
	coord := Coord{Lat: 48.846781, Lon: 2.37715}
	p := FreeFloatingRequestParameter{Distance: 500, Coord: coord, Count: 10, Types: types}
	free_floatings, paginate_freefloatings, err := freeFloatingsContext.GetFreeFloatings(&p)
	require.Nil(err)
	require.NotNil(paginate_freefloatings)
	require.Len(free_floatings, 2)

	assert.Equal("718WSK", free_floatings[0].PublicId)
	assert.Equal("Pony", free_floatings[0].ProviderName)
	assert.Equal("SCOOTER", free_floatings[0].Type)
	assert.Equal("0JT9J6", free_floatings[1].PublicId)
	assert.Equal("Tier", free_floatings[1].ProviderName)
}
