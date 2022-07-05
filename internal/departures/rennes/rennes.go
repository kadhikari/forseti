package rennes

import (
	"fmt"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/departures"
	rennes_destinations "github.com/hove-io/forseti/internal/departures/rennes/destinations"
	rennes_estimatedstoptimes "github.com/hove-io/forseti/internal/departures/rennes/estimatedstoptimes"
	rennes_lastupdate "github.com/hove-io/forseti/internal/departures/rennes/lastupdate"
	rennes_lines "github.com/hove-io/forseti/internal/departures/rennes/lines"
	rennes_routes "github.com/hove-io/forseti/internal/departures/rennes/routes"
	rennes_routestoppoints "github.com/hove-io/forseti/internal/departures/rennes/routestoppoints"
	rennes_stoppoints "github.com/hove-io/forseti/internal/departures/rennes/stoppoints"
	rennes_stoptimes "github.com/hove-io/forseti/internal/departures/rennes/stoptimes"
	"github.com/sirupsen/logrus"
)

const header string = "Api-Keokey"

type RennesContext struct {
	connector      *connectors.Connector
	processingDate *time.Time
	departures     map[string]Departure
	lastUpdate     *time.Time
	mutex          sync.RWMutex
}

func (d *RennesContext) getConnector() *connectors.Connector {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.connector
}

func (d *RennesContext) setConnector(connector *connectors.Connector) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.connector = connector
}

func (d *RennesContext) setDepartures(departures map[string]Departure) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.departures = departures
}

func (d *RennesContext) getProcessingDate() *time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.processingDate
}

func (d *RennesContext) setProcessingDate(processingDate *time.Time) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.processingDate = processingDate
}

func (d *RennesContext) getLastUpdate() *time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.lastUpdate
}

func (d *RennesContext) setLastUpdate(lastUpdate *time.Time) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.lastUpdate = lastUpdate
}

func (d *RennesContext) haveTheoreticalDeparturesBeenLoaded() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.departures != nil
}

func (d *RennesContext) InitContext(
	departuresFilesUrl url.URL,
	departuresFilesRefresh time.Duration,
	departuresServiceUrl url.URL,
	departuresToken string,
	connectionTimeout time.Duration,
	processingDate *time.Time,
) {

	if len(departuresFilesUrl.String()) == 0 || departuresFilesRefresh.Seconds() <= 0 ||
		len(departuresServiceUrl.String()) == 0 ||
		len(departuresToken) <= 0 ||
		connectionTimeout.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}

	{
		d.setConnector(connectors.NewConnector(
			departuresFilesUrl,
			departuresServiceUrl,
			departuresToken,
			departuresFilesRefresh,
			connectionTimeout,
		))

		d.setProcessingDate(processingDate)
		d.setLastUpdate(nil)
		d.setDepartures(nil)
	}
}

func (d *RennesContext) RefereshDeparturesLoop(context *departures.DeparturesContext) {
	context.SetPackageName(reflect.TypeOf(RennesContext{}).PkgPath())
	context.RefreshTime = d.connector.GetRefreshTime()

	// Wait 10 seconds before reloading external departures informations
	time.Sleep(2 * time.Second)
	for {
		var loadedDepartures map[string][]departures.Departure = nil
		if !d.haveTheoreticalDeparturesBeenLoaded() {
			theoreticalDepartures, err := loadTheoreticalDepartures(d)
			if err != nil {
				logrus.Error("Error while the initialization of the theoretical departures: ", err)
			} else {
				logrus.Info("Theoretical departures are initialized")
			}
			if theoreticalDepartures != nil {
				d.setDepartures(theoreticalDepartures)
				d.setLastUpdate(nil)
				loadedDepartures = mapDeparturesByStopPointId(theoreticalDepartures)
			}
		} else {
			estimatedDepartures, newLastUpdate, err := tryToLoadEstimatedDepartures(d)
			if err != nil {
				logrus.Error("Error while the updating of the estimated departures: ", err)

			} else if estimatedDepartures != nil && newLastUpdate != nil {
				logrus.Infof("Estimated departures are updated, new last-update: %v", newLastUpdate.Format(time.RFC3339))
				d.setDepartures(estimatedDepartures)
				d.setLastUpdate(newLastUpdate)
				loadedDepartures = mapDeparturesByStopPointId(estimatedDepartures)
			}
		}
		if loadedDepartures != nil {
			context.UpdateDepartures(loadedDepartures)
		}
		time.Sleep(d.connector.GetRefreshTime())
	}

}

type Departure struct {
	StopTimeId       string
	RouteStopPointId string
	StopPointId      string
	LineId           string
	Direction        departures.DirectionType
	DestinationId    string
	DestinationName  string
	Time             time.Time
	Type             departures.DepartureType
}

func mapDeparturesByStopPointId(rennesDepartures map[string]Departure) map[string][]departures.Departure {
	result := make(map[string][]departures.Departure, 0)
	for _, rennesDeparture := range rennesDepartures {
		appendedDepartures := departures.Departure{
			Line:          rennesDeparture.LineId,
			Stop:          rennesDeparture.StopPointId,
			Type:          fmt.Sprint(rennesDeparture.Type),
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

func loadTheoreticalDepartures(
	rennesContext *RennesContext,
) (map[string]Departure, error) {

	theoreticalDepartures, err := loadTheoreticalDeparturesFromDailyDataFiles(
		rennesContext.getConnector().GetFilesUri(),
		rennesContext.getConnector().GetConnectionTimeout(),
		rennesContext.getProcessingDate(),
	)
	if err != nil {
		return nil, err
	}
	return theoreticalDepartures, nil
}

func loadTheoreticalDeparturesFromDailyDataFiles(
	uri url.URL,
	connectionTimeout time.Duration,
	processingDate *time.Time,
) (map[string]Departure, error) {

	var stopPoints map[string]rennes_stoppoints.StopPoint
	var lines map[string]rennes_lines.Line
	var routes map[string]rennes_routes.Route
	var destinations map[string]rennes_destinations.Destination
	var routeStopPoints map[string]rennes_routestoppoints.RouteStopPoint

	stopPoints, err := rennes_stoppoints.LoadStopPoints(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the stop points: %v", err)
	}

	lines, err = rennes_lines.LoadLines(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the lines: %v", err)
	}

	routes, err = rennes_routes.LoadRoutes(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the routes: %v", err)
	}

	destinations, err = rennes_destinations.LoadDestinations(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the destinations: %v", err)
	}

	stopTimes, err := rennes_stoptimes.LoadStopTimes(uri, connectionTimeout, processingDate)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the stop times: %v", err)
	}

	routeStopPoints, err = rennes_routestoppoints.LoadRouteStopPoints(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the route stop points: %v", err)
	}

	theoreticalDepartures := make(map[string]Departure, 0)
	for _, stopTime := range stopTimes {
		routeStopPoint := routeStopPoints[stopTime.RouteStopPointId]

		var line rennes_lines.Line
		var direction departures.DirectionType
		var destinationId string
		if route, isLoaded := routes[routeStopPoint.RouteId]; isLoaded {
			line = lines[route.LineInternalId]
			direction = route.Direction
			destinationId = route.DestinationId
		} else {
			continue
		}
		destinationName := destinations[destinationId].Name
		stopPoint := stopPoints[routeStopPoint.StopPointId]

		theoreticalDepartures[stopTime.Id] = Departure{
			StopTimeId:       stopTime.Id,
			RouteStopPointId: routeStopPoint.Id,
			StopPointId:      stopPoint.ExternalId,
			LineId:           line.ExternalId,
			Direction:        direction,
			DestinationId:    destinationId,
			DestinationName:  destinationName,
			Time:             stopTime.Time,
			Type:             departures.DepartureTypeTheoretical,
		}
	}

	return theoreticalDepartures, nil
}

func tryToLoadEstimatedDepartures(rennesContext *RennesContext) (map[string]Departure, *time.Time, error) {
	loadedLastUpdate, err := rennes_lastupdate.LoadLastUpdate(
		rennesContext.getConnector().GetUrl(),
		header,
		rennesContext.getConnector().GetToken(),
		rennesContext.getConnector().GetConnectionTimeout(),
		rennesContext.getProcessingDate(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error while the loading of the last-update: %v", err)
	}

	currentLastUpdate := rennesContext.getLastUpdate()

	if currentLastUpdate == nil || loadedLastUpdate.After(*currentLastUpdate) {
		var logMessage string
		if currentLastUpdate == nil {
			logMessage = fmt.Sprintf(
				"Update of the estimated departures, last-update: %s ...",
				loadedLastUpdate.Format(time.RFC3339),
			)
		} else {
			logMessage = fmt.Sprintf(
				"Update of the estimated departures, last-update: %s -> %s ...",
				currentLastUpdate.Format(time.RFC3339),
				loadedLastUpdate.Format(time.RFC3339),
			)
		}
		logrus.Info(logMessage)
		estimatedDepartures, err := loadEstimatedDepartures(rennesContext)
		if err != nil {
			return nil, nil, fmt.Errorf("error while the loading of the estimated stop times: %v", err)
		}
		return estimatedDepartures, loadedLastUpdate, nil

	} else {
		logrus.Infof(
			"No newer estimated departures available, last-update: %v",
			currentLastUpdate.Format(time.RFC3339),
		)
	}
	return nil, nil, nil
}

func loadEstimatedDepartures(rennesContext *RennesContext) (map[string]Departure, error) {

	var estimatedStopTimes map[string]rennes_stoptimes.StopTime
	estimatedStopTimes, err := rennes_estimatedstoptimes.LoadEstimatedStopTimes(
		rennesContext.connector.GetUrl(),
		header,
		rennesContext.getConnector().GetToken(),
		rennesContext.getConnector().GetConnectionTimeout(),
		rennesContext.getProcessingDate(),
	)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	departuresCopy := make(map[string]Departure, 0)
	for key, value := range rennesContext.departures {
		departuresCopy[key] = value
	}

	for stopTimeId, estimatedStopTime := range estimatedStopTimes {
		// Update the estimated time of the departure
		if departure, ok := departuresCopy[stopTimeId]; ok {
			departure.Time = estimatedStopTime.Time
			departure.Type = departures.DepartureTypeEstimated
			departuresCopy[stopTimeId] = departure
		}
	}
	return departuresCopy, nil

}
