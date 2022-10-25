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

const FilterDeltaDuration time.Duration = 2 * time.Minute

func getActualProcessingDate(
	location *time.Location,
	serviceSwitchTime *time.Time) time.Time {
	localNowTime := time.Now().In(location)
	localServiceStartTime, _ := rennes_stoptimes.CombineDateAndTimeInLocation(
		&localNowTime,
		serviceSwitchTime,
		location,
	)
	if localNowTime.Before(localServiceStartTime) {
		localNowTime = localNowTime.AddDate(
			0,  /* year */
			0,  /* month */
			-1, /* day */
		)
	}
	// set the processing date to midnight
	return time.Date(
		localNowTime.Year(),
		localNowTime.Month(),
		localNowTime.Day(),
		0, /* hour */
		0, /* minute */
		0, /* second */
		0, /* nanosecond */
		localNowTime.Location(),
	)
}

type RennesContext struct {
	connector         *connectors.Connector
	processingDate    *time.Time
	location          *time.Location
	serviceSwitchTime *time.Time
	departures        map[string]Departure
	lastUpdate        *time.Time
	mutex             sync.RWMutex
}

func (d *RennesContext) getConnector() *connectors.Connector {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.connector
}

func (d *RennesContext) setConnector(connector *connectors.Connector) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.connector = connector
}

func (d *RennesContext) getDepartures() map[string]Departure {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.departures
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

func (d *RennesContext) getLocation() *time.Location {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.location
}

func (d *RennesContext) setLocation(location *time.Location) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.location = location
}

func (d *RennesContext) getServiceSwitchTime() *time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.serviceSwitchTime
}

func (d *RennesContext) setServiceSwitchTime(serviceSwitchTime *time.Time) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.serviceSwitchTime = serviceSwitchTime
}

func (d *RennesContext) computeDailyServiceStartTime() time.Time {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	serviceStartTime := time.Date(
		d.processingDate.Year(),
		d.processingDate.Month(),
		d.processingDate.Day(),
		d.serviceSwitchTime.Hour(),
		d.serviceSwitchTime.Minute(),
		d.serviceSwitchTime.Second(),
		d.serviceSwitchTime.Nanosecond(),
		d.location,
	)
	return serviceStartTime
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
	departuresServiceRefresh time.Duration,
	departuresToken string,
	connectionTimeout time.Duration,
	location *time.Location,
	serviceSwitchTime *time.Time,
) {

	if len(departuresFilesUrl.String()) == 0 || departuresFilesRefresh.Seconds() <= 0 ||
		len(departuresServiceUrl.String()) == 0 || departuresServiceRefresh.Seconds() <= 0 ||
		len(departuresToken) <= 0 ||
		connectionTimeout.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}

	d.setConnector(connectors.NewConnector(
		departuresFilesUrl,
		departuresServiceUrl,
		departuresToken,
		departuresFilesRefresh,
		departuresServiceRefresh,
		connectionTimeout,
	))

	actualProcessingDate := getActualProcessingDate(
		location,
		serviceSwitchTime,
	)

	d.setLocation(location)
	d.setServiceSwitchTime(serviceSwitchTime)
	d.UpdateProcessingDateAndCleanDeparturesList(&actualProcessingDate, nil)
}

func (d *RennesContext) UpdateProcessingDateAndCleanDeparturesList(
	processingDate *time.Time,
	context *departures.DeparturesContext,
) {
	d.setLastUpdate(nil)
	d.setDepartures(nil)
	d.setProcessingDate(processingDate)
	if context != nil {
		context.DropDepartures()
	}
	dailyServiceStartTime := d.computeDailyServiceStartTime()
	logrus.Infof("The depatures list is reset, the processing date is: %v", processingDate)
	logrus.Infof("The depatures list is reset, the daily service start time is: %v", dailyServiceStartTime)
}

func (d *RennesContext) RefereshDeparturesLoop(context *departures.DeparturesContext) {
	context.SetPackageName(reflect.TypeOf(RennesContext{}).PkgPath())
	context.SetFilesRefeshTime(d.connector.GetFilesRefreshTime())
	context.SetWsRefeshTime(d.connector.GetWsRefreshTime())

	// Wait 10 seconds before reloading external departures informations
	time.Sleep(10 * time.Second)
	for {
		var loadedDepartures map[string][]departures.Departure = nil
		var refreshTime time.Duration
		// Clean the departures list if necessary
		{
			currentProcessingDate := *d.getProcessingDate()
			actualProcessingDate := getActualProcessingDate(
				d.getLocation(), d.getServiceSwitchTime(),
			)
			if actualProcessingDate != currentProcessingDate {
				d.UpdateProcessingDateAndCleanDeparturesList(&actualProcessingDate, context)
			}
		}

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
				loadedDepartures = filterTooOldDepartures(loadedDepartures)
				refreshTime = time.Duration(0) // skip the refresh time
			} else {
				refreshTime = context.GetFilesRefeshTime()
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
				loadedDepartures = filterTooOldDepartures(loadedDepartures)
			}
			refreshTime = context.GetWsRefeshTime()
		}
		if loadedDepartures != nil {
			context.UpdateDepartures(loadedDepartures)
		}
		time.Sleep(refreshTime)
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

func filterTooOldDepartures(mappedDepartures map[string][]departures.Departure) map[string][]departures.Departure {
	var lowerBound time.Time = time.Now().UTC().Add(-FilterDeltaDuration)
	mappedAndFilteredDepartures := make(map[string][]departures.Departure, len(mappedDepartures))
	for stopPointId, departureList := range mappedDepartures {
		filteredDepartureList := make([]departures.Departure, 0)
		for _, d := range departureList {
			if !d.Datetime.Before(lowerBound) {
				filteredDepartureList = append(filteredDepartureList, d)
			}
		}
		mappedAndFilteredDepartures[stopPointId] = filteredDepartureList
	}
	return mappedAndFilteredDepartures
}

func loadTheoreticalDepartures(
	rennesContext *RennesContext,
) (map[string]Departure, error) {

	dailyServiceStartTime := rennesContext.computeDailyServiceStartTime()

	theoreticalDepartures, err := loadTheoreticalDeparturesFromDailyDataFiles(
		rennesContext.getConnector().GetFilesUri(),
		rennesContext.getConnector().GetConnectionTimeout(),
		&dailyServiceStartTime,
	)
	if err != nil {
		return nil, err
	}
	return theoreticalDepartures, nil
}

func loadTheoreticalDeparturesFromDailyDataFiles(
	uri url.URL,
	connectionTimeout time.Duration,
	dailyServiceStartTime *time.Time,
) (map[string]Departure, error) {

	var stopPoints map[string]rennes_stoppoints.StopPoint
	var lines map[string]rennes_lines.Line
	var routes map[string]rennes_routes.Route
	var destinations map[string]rennes_destinations.Destination
	var unsortedRouteStopPoints map[string]rennes_routestoppoints.RouteStopPoint

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

	stopTimes, err := rennes_stoptimes.LoadStopTimes(uri, connectionTimeout, dailyServiceStartTime)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the stop times: %v", err)
	}

	unsortedRouteStopPoints, err = rennes_routestoppoints.LoadRouteStopPoints(uri, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("an unexpected error occurred while the loadings of the route stop points: %v", err)
	}

	sortedRouteStopPoints := rennes_routestoppoints.SortRouteStopPointsByOrder(
		unsortedRouteStopPoints,
	)

	theoreticalDepartures := make(map[string]Departure, 0)
	for _, stopTime := range stopTimes {
		routeStopPoint := unsortedRouteStopPoints[stopTime.RouteStopPointId]

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
		var lastStopPointExternalId string
		{
			lastStopPointInternalId, err := rennes_routestoppoints.GetLastStopPointInternalId(
				sortedRouteStopPoints,
				routeStopPoint.RouteId,
			)
			if err != nil {
				continue
			}
			lastStopPointExternalId = stopPoints[lastStopPointInternalId].ExternalId
		}
		destinationName := destinations[destinationId].Name
		stopPoint := stopPoints[routeStopPoint.StopPointInternalId]

		theoreticalDepartures[stopTime.Id] = Departure{
			StopTimeId:       stopTime.Id,
			RouteStopPointId: routeStopPoint.Id,
			StopPointId:      stopPoint.ExternalId,
			LineId:           line.ExternalId,
			Direction:        direction,
			DestinationId:    lastStopPointExternalId,
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
		rennesContext.getLocation(),
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
			"No newer estimated departure available, last-update: %v",
			currentLastUpdate.Format(time.RFC3339),
		)
	}
	return nil, nil, nil
}

func loadEstimatedDepartures(rennesContext *RennesContext) (map[string]Departure, error) {

	var estimatedStopTimes map[string]rennes_stoptimes.StopTime
	dailyServiceStartTime := rennesContext.computeDailyServiceStartTime()
	estimatedStopTimes, err := rennes_estimatedstoptimes.LoadEstimatedStopTimes(
		rennesContext.connector.GetUrl(),
		header,
		rennesContext.getConnector().GetToken(),
		rennesContext.getConnector().GetConnectionTimeout(),
		&dailyServiceStartTime,
	)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	departuresCopy := make(map[string]Departure, 0)
	for key, value := range rennesContext.getDepartures() {
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
