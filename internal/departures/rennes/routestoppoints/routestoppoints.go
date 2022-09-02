package routestoppoints

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const RouteStopPointsCsvFileName string = "arrets_chn.lar"
const routeStopPointsCsvNumOfFields int = 4

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates RouteStopPoint objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type RouteStopPoint struct {
	Id                  string
	StopPointInternalId string
	RouteId             string
	StopPointOrder      int
}

func newRouteStopPoint(record []string) (*RouteStopPoint, error) {
	if len(record) < routeStopPointsCsvNumOfFields {
		return nil, fmt.Errorf("missing field in RouteStopPoint record")
	}

	stopPointOrder, err := strconv.Atoi(record[3])
	if err != nil {
		return nil, err
	}

	return &RouteStopPoint{
		Id:                  record[0],
		StopPointInternalId: record[1],
		RouteId:             record[2],
		StopPointOrder:      stopPointOrder,
	}, nil
}

type routeStopPointCsvLineConsumer struct {
	routeStopPoints map[string]RouteStopPoint
}

func makeRouteStopPointCsvLineConsumer() *routeStopPointCsvLineConsumer {
	return &routeStopPointCsvLineConsumer{
		routeStopPoints: make(map[string]RouteStopPoint),
	}
}

func (c *routeStopPointCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	routeStopPoint, err := newRouteStopPoint(csvLine)
	if err != nil {
		return err
	}
	c.routeStopPoints[routeStopPoint.Id] = *routeStopPoint
	return nil
}

func (c *routeStopPointCsvLineConsumer) Terminate() {
}

func LoadRouteStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]RouteStopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, RouteStopPointsCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      routeStopPointsCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	routeStopPointsConsumer := makeRouteStopPointCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, routeStopPointsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return routeStopPointsConsumer.routeStopPoints, nil
}

// Sort stop point following their order in the route
func SortRouteStopPointsByOrder(routeStopPoints map[string]RouteStopPoint) map[string][]RouteStopPoint {
	unsorted := make(map[string][]RouteStopPoint)
	for _, routeStopPoint := range routeStopPoints {
		routeId := routeStopPoint.RouteId
		if _, ok := unsorted[routeId]; !ok {
			unsorted[routeId] = make([]RouteStopPoint, 0)
		}
		unsorted[routeId] = append(unsorted[routeId], routeStopPoint)
	}
	// sort all lists of route stop points following their order
	for _, routeStopPointList := range unsorted {
		sort.SliceStable(routeStopPointList, func(i, j int) bool {
			return routeStopPointList[i].StopPointOrder < routeStopPointList[j].StopPointOrder
		})
	}
	return unsorted
}

func GetLastStopPointInternalId(
	sortedRouteStopPoints map[string][]RouteStopPoint,
	routeId string,
) (string, error) {
	err := fmt.Errorf("route id not found: %s", routeId)
	if sortedRouteStopPointsList, ok := sortedRouteStopPoints[routeId]; ok {
		if sortedRouteStopPointsList == nil {
			return "", err
		}
		numberOfStopPoints := len(sortedRouteStopPointsList)
		if numberOfStopPoints == 0 {
			return "", err
		}
		return sortedRouteStopPointsList[numberOfStopPoints-1].StopPointInternalId, nil
	}
	return "", err
}
