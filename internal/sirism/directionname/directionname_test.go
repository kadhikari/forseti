package directionname

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectionNameUnmarshalXML(t *testing.T) {

	var tests = []struct {
		xmlInnerText   string
		expectedResult DirectionName
	}{
		{
			xmlInnerText:   "<DirectionName>ALLER</DirectionName>",
			expectedResult: DirectionNameAller,
		},
		{
			xmlInnerText:   "<DirectionName>Aller</DirectionName>",
			expectedResult: DirectionNameAller,
		},
		{
			xmlInnerText:   "<DirectionName>aller</DirectionName>",
			expectedResult: DirectionNameAller,
		},
		{
			xmlInnerText:   "<DirectionName>A</DirectionName>",
			expectedResult: DirectionNameAller,
		},
		{
			xmlInnerText:   "<DirectionName>a</DirectionName>",
			expectedResult: DirectionNameAller,
		},
		{
			xmlInnerText:   "<DirectionName>RETOUR</DirectionName>",
			expectedResult: DirectionNameRetour,
		},
		{
			xmlInnerText:   "<DirectionName>Retour</DirectionName>",
			expectedResult: DirectionNameRetour,
		},
		{
			xmlInnerText:   "<DirectionName>retour</DirectionName>",
			expectedResult: DirectionNameRetour,
		},
		{
			xmlInnerText:   "<DirectionName>R</DirectionName>",
			expectedResult: DirectionNameRetour,
		},
		{
			xmlInnerText:   "<DirectionName>r</DirectionName>",
			expectedResult: DirectionNameRetour,
		},
	}

	assert := assert.New(t)
	for _, test := range tests {
		xmlBytes := []byte(test.xmlInnerText)
		var directionName DirectionName
		err := xml.Unmarshal(xmlBytes, &directionName)
		assert.Nil(err)
		assert.Equal(test.expectedResult, directionName)
	}

}
