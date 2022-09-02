package routestoppoints

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var fixtureDir string

const defaultTimeout time.Duration = time.Second * 10

func TestMain(m *testing.M) {

	fixtureDir = os.Getenv("FIXTUREDIR")
	if fixtureDir == "" {
		panic("$FIXTUREDIR isn't set")
	}

	os.Exit(m.Run())
}

func TestLoadRouteStopPoints(t *testing.T) {

	const EXPECTED_NUM_OF_ROUTE_STOP_POINTS int = 12_508

	assert := assert.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referential", fixtureDir))
	assert.Nil(err)

	loadedRoutesStopPoints, err := LoadRouteStopPoints(*uri, defaultTimeout)
	assert.Nil(err)
	assert.Len(loadedRoutesStopPoints, EXPECTED_NUM_OF_ROUTE_STOP_POINTS)

	// Check the values read from the first line of the CSV
	{
		const EXPECTED_ID string = "284722693"
		const EXPECTED_STOP_POINT_INTERNAL_ID string = "1412"
		const EXPECTED_ROUTES_ID string = "284722688"
		const EXPECTED_STOP_POINT_ORDER int = 4251

		assert.Contains(loadedRoutesStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedRoutesStopPoints[EXPECTED_ID],
			RouteStopPoint{
				Id:                  EXPECTED_ID,
				StopPointInternalId: EXPECTED_STOP_POINT_INTERNAL_ID,
				RouteId:             EXPECTED_ROUTES_ID,
				StopPointOrder:      EXPECTED_STOP_POINT_ORDER,
			},
		)
	}

	// Check the values read from the last line of the CSV
	{
		const EXPECTED_ID string = "268501249"
		const EXPECTED_STOP_POINT_INTERNAL_ID string = "1004"
		const EXPECTED_ROUTES_ID string = "268501248"
		const EXPECTED_STOP_POINT_ORDER int = 0

		assert.Contains(loadedRoutesStopPoints, EXPECTED_ID)
		assert.Equal(
			loadedRoutesStopPoints[EXPECTED_ID],
			RouteStopPoint{
				Id:                  EXPECTED_ID,
				StopPointInternalId: EXPECTED_STOP_POINT_INTERNAL_ID,
				RouteId:             EXPECTED_ROUTES_ID,
				StopPointOrder:      EXPECTED_STOP_POINT_ORDER,
			},
		)
	}

}

func TestSortRouteStopPointsByOrder(t *testing.T) {

	const EXPECTED_NUM_OF_ROUTES int = 1811
	const UNEXPECTED_ROUTE_ID string = "123456789"
	const EXPECTED_ROUTE_ID string = "284721408"
	const EXPECTED_NUM_OF_STOP_POINTS int = 14
	EXPECTED_STOP_POINTS_ORDER := []int{
		0,
		764,
		1734,
		2394,
		2960,
		3060,
		3455,
		3555,
		4087,
		4589,
		5279,
		7796,
		8476,
		8840,
	}

	assert := assert.New(t)
	uri, err := url.Parse(fmt.Sprintf("file://%s/data_rennes/referential", fixtureDir))
	assert.Nil(err)

	unsortedRouteStopPoints, err := LoadRouteStopPoints(*uri, defaultTimeout)
	assert.Nil(err)

	sortedRouteStopPoints := SortRouteStopPointsByOrder(unsortedRouteStopPoints)
	assert.Len(sortedRouteStopPoints, EXPECTED_NUM_OF_ROUTES)
	assert.Contains(sortedRouteStopPoints, EXPECTED_ROUTE_ID)
	assert.NotContains(sortedRouteStopPoints, UNEXPECTED_ROUTE_ID)

	sortedRouteStopPointsList := sortedRouteStopPoints[EXPECTED_ROUTE_ID]
	assert.Len(sortedRouteStopPointsList, EXPECTED_NUM_OF_STOP_POINTS)

	gotStopPointsOrder := make([]int, 0, EXPECTED_NUM_OF_STOP_POINTS)
	for _, routeStopPoint := range sortedRouteStopPointsList {
		gotStopPointsOrder = append(
			gotStopPointsOrder,
			routeStopPoint.StopPointOrder,
		)
	}
	assert.Equal(
		EXPECTED_STOP_POINTS_ORDER,
		gotStopPointsOrder,
	)

}
