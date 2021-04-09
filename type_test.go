package forseti

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
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
