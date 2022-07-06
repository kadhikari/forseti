package routes

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const RoutesCsvFileName string = "chainages.pty"
const routesCsvNumOfFields int = 5

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates Routes objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type Route struct {
	Id             string // ID of the route
	LineInternalId string
	Direction      departures.DirectionType
	DestinationId  string
}

func newRoute(record []string) (*Route, error) {
	if len(record) < routesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in Route record")
	}

	var direction departures.DirectionType
	directionName := record[2]
	if directionName == "A" { // Aller = Forward in french
		direction = departures.DirectionTypeForward
	} else if directionName == "R" { // Retour = Backward in french
		direction = departures.DirectionTypeBackward
	} else {
		return nil, fmt.Errorf("unexpected direction in Route record: expected 'A' or 'R', got '%s'", directionName)
	}

	return &Route{
		Id:             record[0],
		LineInternalId: record[3],
		Direction:      direction,
		DestinationId:  record[4],
	}, nil
}

type routeCsvLineConsumer struct {
	routes map[string]Route
}

func makeRouteCsvLineConsumer() *routeCsvLineConsumer {
	return &routeCsvLineConsumer{
		routes: make(map[string]Route),
	}
}

func (c *routeCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	route, err := newRoute(csvLine)
	if err != nil {
		return err
	}
	c.routes[route.Id] = *route
	return nil
}

func (c *routeCsvLineConsumer) Terminate() {
}

func LoadRoutes(uri url.URL, connectionTimeout time.Duration) (map[string]Route, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, RoutesCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      routesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	routeConsumer := makeRouteCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, routeConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return routeConsumer.routes, nil
}
