package rennes

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/departures"
	rennes_buslines "github.com/hove-io/forseti/internal/departures/rennes/buslines"
	rennes_dbinternallinks "github.com/hove-io/forseti/internal/departures/rennes/dbinternallinks"
	rennes_destinations "github.com/hove-io/forseti/internal/departures/rennes/destinations"
	rennes_routes "github.com/hove-io/forseti/internal/departures/rennes/routes"
	rennes_scheduledtimes "github.com/hove-io/forseti/internal/departures/rennes/scheduledtimes"
	rennes_stoppoints "github.com/hove-io/forseti/internal/departures/rennes/stoppoints"
	"github.com/sirupsen/logrus"
)

type RennesContext struct {
	connector                *connectors.Connector
	areDeparturesInitialized bool
	processingDate           time.Time
}

func (d *RennesContext) InitContext(
	externalURI url.URL,
	loadExternalRefresh,
	connectionTimeout time.Duration,
	processingDate time.Time,
) {

	if len(externalURI.String()) == 0 || loadExternalRefresh.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}
	d.connector = connectors.NewConnector(externalURI, externalURI, "", loadExternalRefresh, connectionTimeout)
	d.processingDate = processingDate
	d.areDeparturesInitialized = false
}

func (d *RennesContext) InitializeDeparturesLoop(context *departures.DeparturesContext) {
	context.SetPackageName(reflect.TypeOf(RennesContext{}).PkgPath())
	context.RefreshTime = d.connector.GetRefreshTime()

	// Wait 10 seconds before reloading external departures informations
	time.Sleep(10 * time.Second)
	for {
		hasBeenLoaded, err := InitializeDepartures(d, context)
		if err != nil {
			logrus.Error("Error while the initialization of the departures data: ", err)
		} else {
			logrus.Debug("Departures data are initialized")
		}
		if hasBeenLoaded {
			d.areDeparturesInitialized = true

			return
		}
		time.Sleep(d.connector.GetRefreshTime())
	}

}

func InitializeDepartures(rennesContext *RennesContext, context *departures.DeparturesContext) (bool, error) {

	loadedDepartures, err := LoadScheduledDeparturesFromDailyDataFiles(
		rennesContext.connector.GetFilesUri(),
		rennesContext.connector.GetConnectionTimeout(),
		&rennesContext.processingDate,
	)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return false, err
	}
	mappedDepartures := mapDeparturesFollowingStopPoint(loadedDepartures)

	context.UpdateDepartures(mappedDepartures)
	return true, nil
}

func mapDeparturesFollowingStopPoint(rennesDepartures []Departure) map[string][]departures.Departure {
	result := make(map[string][]departures.Departure, 0)
	for _, rennesDeparture := range rennesDepartures {
		appendedDepartures := departures.Departure{
			Line:          rennesDeparture.BusLineId,
			Stop:          rennesDeparture.StopPointId,
			Type:          fmt.Sprint(departures.DepartureTypeTheoretical),
			Direction:     rennesDeparture.DestinationId,
			DirectionName: rennesDeparture.DestinationName,
			Datetime:      rennesDeparture.Time,
			DirectionType: rennesDeparture.Direction,
		}
		// Initilize a new list of departures if necessary
		if _, ok := result[rennesDeparture.StopPointId]; !ok {
			result[rennesDeparture.StopPointId] = make([]departures.Departure, 0)
		}
		result[rennesDeparture.StopPointId] = append(
			result[rennesDeparture.StopPointId],
			appendedDepartures,
		)
	}
	return result
}

type Departure struct {
	DbInternalLinkId string
	StopPointId      string
	BusLineId        string
	Direction        departures.DirectionType
	DestinationId    string
	DestinationName  string
	Time             time.Time
}

func LoadScheduledDeparturesFromDailyDataFiles(
	uri url.URL,
	connectionTimeout time.Duration,
	processingDate *time.Time,
) ([]Departure, error) {

	var stopPoints map[string]rennes_stoppoints.StopPoint
	var busLines map[string]rennes_buslines.BusLine
	var routes map[string]rennes_routes.Route
	var destinations map[string]rennes_destinations.Destination
	var dbInternalLinks map[string]rennes_dbinternallinks.DbInternalLink

	stopPoints, err := rennes_stoppoints.LoadStopPoints(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the stop points: %v", err)
	}

	busLines, err = rennes_buslines.LoadBusLines(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the bus lines: %v", err)
	}

	routes, err = rennes_routes.LoadRoutes(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the routes: %v", err)
	}

	destinations, err = rennes_destinations.LoadDestinations(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the destinations: %v", err)
	}

	scheduledTimes, err := rennes_scheduledtimes.LoadScheduledTimes(uri, connectionTimeout, processingDate)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the scheduled times: %v", err)
	}

	dbInternalLinks, err = rennes_dbinternallinks.LoadDbInternalLinks(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the DB links: %v", err)
	}

	scheduledDepartures := make([]Departure, 0, len(scheduledTimes))
	for _, scheduledTime := range scheduledTimes {
		dbInternalLink := dbInternalLinks[scheduledTime.DbInternalLinkId]

		var busLine rennes_buslines.BusLine
		var direction departures.DirectionType
		var destinationId string
		if route, isLoaded := routes[dbInternalLink.RouteId]; isLoaded {
			busLine = busLines[route.BusLineInternalId]
			direction = route.Direction
			destinationId = route.DestinationId
		} else {
			continue
		}
		destinationName := destinations[destinationId].Name
		stopPoint := stopPoints[dbInternalLink.StopPointId]

		scheduledDepartures = append(scheduledDepartures,
			Departure{
				DbInternalLinkId: dbInternalLink.Id,
				StopPointId:      stopPoint.ExternalId,
				BusLineId:        busLine.ExternalId,
				Direction:        direction,
				DestinationId:    destinationId,
				DestinationName:  destinationName,
				Time:             scheduledTime.Time,
			},
		)
	}

	return scheduledDepartures, nil
}
