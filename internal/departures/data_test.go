package departures

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepartureTypeString(t *testing.T) {
	require := require.New(t)
	{
		require.Equal(fmt.Sprint(DepartureTypeTheoretical), "T")
	}
	{
		require.Equal(fmt.Sprint(DepartureTypeEstimated), "E")
	}
}

func TestParseDirectionTypeFromNavitia(t *testing.T) {
	assert := assert.New(t)

	var tests = []struct {
		directionTypeStr string
		errorIsExpected  bool
		expectedResult   DirectionType
	}{
		{
			directionTypeStr: "FORWARD",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "Forward",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "forward",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "BACKWARD",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "Backward",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "backward",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "OUTBOUND",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "Outbound",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "outbound",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeForward,
		},
		{
			directionTypeStr: "INBOUND",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "Inbound",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "inbound",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBackward,
		},
		{
			directionTypeStr: "",
			errorIsExpected:  false,
			expectedResult:   DirectionTypeBoth,
		},
		{
			directionTypeStr: "foo",
			errorIsExpected:  true,
			expectedResult:   DirectionTypeUnknown,
		},
		{
			directionTypeStr: "ALL",
			errorIsExpected:  true,
			expectedResult:   DirectionTypeUnknown,
		},
	}

	for _, test := range tests {
		getResult, err := ParseDirectionTypeFromNavitia(test.directionTypeStr)
		if test.errorIsExpected {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
			assert.Equal(
				test.expectedResult,
				getResult,
			)
		}
	}
}
