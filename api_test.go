package sytralrt

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeparturesApi(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	firstURI, err := url.Parse(fmt.Sprintf("file://%s/first.txt", fixtureDir))
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
	err = RefreshDepartures(&manager, *firstURI)
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

	err = RefreshDepartures(&manager, *firstURI)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response StatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	assert.Equal(response.Status, "ok")
	assert.True(response.LastDataUpdate.After(startTime))
	assert.True(response.LastDataUpdate.Before(time.Now()))
}
