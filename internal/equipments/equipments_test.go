package equipments

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

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

var defaultTimeout time.Duration = time.Second * 10

func TestEquipmentsAPI(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	equipmentURI, err := url.Parse(fmt.Sprintf("file://%s/NET_ACCESS.XML", fixtureDir))
	require.Nil(err)

	equipmentsContext := &EquipmentsContext{}

	c, router := gin.CreateTestContext(httptest.NewRecorder())
	AddEquipmentsEntryPoint(router, equipmentsContext)

	err = RefreshEquipments(equipmentsContext, *equipmentURI, defaultTimeout)
	assert.Nil(err)

	c.Request = httptest.NewRequest("GET", "/equipments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response EquipmentsApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)
	require.NotNil(response.Equipments)
	require.NotEmpty(response.Equipments)
	assert.Len(response.Equipments, 3)
	assert.Empty(response.Error)
}

func TestLoadEquipmentsData(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	uri, err := url.Parse(fmt.Sprintf("file://%s/NET_ACCESS.XML", fixtureDir))
	require.Nil(err)
	reader, err := utils.GetFileWithFS(*uri)
	require.Nil(err)

	eds, err := loadXmlEquipments(reader)
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

func TestDataManagerGetEquipments(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	loc, err := time.LoadLocation("Europe/Paris")
	require.Nil(err)

	var equipmentsContext EquipmentsContext
	equipments := make([]EquipmentDetail, 0)
	ess := make([]data.EquipementSource, 0)
	es := data.EquipementSource{ID: "toto", Name: "toto paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	es = data.EquipementSource{ID: "tata", Name: "tata paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	es = data.EquipementSource{ID: "titi", Name: "toto paris", Type: "ASCENSEUR", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	ess = append(ess, es)
	updatedAt := time.Now()

	for _, es := range ess {
		ed, err := NewEquipmentDetail(es, updatedAt, loc)
		require.Nil(err)
		equipments = append(equipments, *ed)
	}
	equipmentsContext.UpdateEquipments(equipments)
	equipDetails, err := equipmentsContext.GetEquipments()
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

	equipmentsWithBadEType := make([]data.EquipementSource, 0)
	es := data.EquipementSource{ID: "toto", Name: "toto paris", Type: "ASC", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	equipmentsWithBadEType = append(equipmentsWithBadEType, es)
	es = data.EquipementSource{ID: "tata", Name: "tata paris", Type: "ASC", Cause: "Problème technique",
		Effect: "Accès", Start: "2018-09-17", End: "2018-09-18", Hour: "23:30:00"}
	equipmentsWithBadEType = append(equipmentsWithBadEType, es)
	updatedAt := time.Now()

	for _, badEType := range equipmentsWithBadEType {
		ed, err := NewEquipmentDetail(badEType, updatedAt, loc)
		assert.Error(err)
		assert.Nil(ed)
	}
}
