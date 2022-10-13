package vehiclepositions

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/hove-io/forseti/google_transit"
	"github.com/hove-io/forseti/internal/connectors"

	"github.com/sirupsen/logrus"

	rennes_lastupdate "github.com/hove-io/forseti/internal/departures/rennes/lastupdate"
	rennes_vp "github.com/hove-io/forseti/internal/departures/rennes/vehiclepositions"
)

// Aliases
type ExternalVehiculeId = rennes_vp.ExternalVehiculeId

const header string = "Api-Keokey"

type RennesContext struct {
	vehiclePositions           *VehiclePositions
	connector                  *connectors.Connector
	realTimeVehiclePosition    *rennes_vp.VehiclePosition
	realTimeVehicleJourneyCode *string
	navitiaUrl                 url.URL
	navitiaToken               string
	navitiaCoverage            string
	location                   *time.Location
	lastUpdate                 *time.Time
	mutex                      sync.RWMutex
}

func (d *RennesContext) GetAllVehiclePositions() *VehiclePositions {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.vehiclePositions == nil {
		d.vehiclePositions = &VehiclePositions{}
	}
	return d.vehiclePositions
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

// func (d *RennesContext) getRealTimeVehiclePosition() *rennes_vp.VehiclePosition {
// 	d.mutex.RLock()
// 	defer d.mutex.RUnlock()
// 	return d.realTimeVehiclePosition
// }

func (d *RennesContext) setRealTimeVehiclePosition(realTimeVehiclePosition *rennes_vp.VehiclePosition) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.realTimeVehiclePosition = realTimeVehiclePosition
}

func (d *RennesContext) setRealTimeVehicleJourneyCode(realTimeVehicleJourneyCode *string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.realTimeVehicleJourneyCode = realTimeVehicleJourneyCode
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

func (d *RennesContext) getNavitiaUrl() url.URL {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.navitiaUrl
}

func (d *RennesContext) setNavitiaUrl(navitiaUrl url.URL) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.navitiaUrl = navitiaUrl
}

func (d *RennesContext) getNavitiaToken() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.navitiaToken
}

func (d *RennesContext) setNavitiaToken(navitiaToken string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.navitiaToken = navitiaToken
}

func (d *RennesContext) getNavitiaCoverage() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.navitiaCoverage
}

func (d *RennesContext) setNavitiaCoverage(navitiaCoverage string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.navitiaCoverage = navitiaCoverage
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *RennesContext) InitContext(
	filesUrl url.URL,
	filesRefreshDuration time.Duration,
	serviceUrl url.URL,
	serviceToken string,
	serviceRefreshDuration time.Duration,
	navitiaUrl url.URL,
	navitiaToken string,
	navitiaCoverage string,
	connectionTimeout time.Duration,
	positionCleanVP time.Duration,
	location *time.Location,
	reloadActive bool,
) {

	d.setConnector(connectors.NewConnector(
		filesUrl,
		serviceUrl,
		serviceToken,
		filesRefreshDuration,
		serviceRefreshDuration,
		connectionTimeout,
	))

	d.setNavitiaUrl(navitiaUrl)
	d.setNavitiaToken(navitiaToken)
	d.setNavitiaCoverage(navitiaCoverage)
	d.setLocation(location)
	d.setLastUpdate(nil)
	d.setRealTimeVehiclePosition(nil)
	d.setRealTimeVehicleJourneyCode(nil)
	d.vehiclePositions = &VehiclePositions{}
	d.vehiclePositions.ManageVehiclePositionsStatus(reloadActive)
}

// main loop to refresh vehicle_positions
func (d *RennesContext) RefreshVehiclePositionsLoop() {
	// Wait 10 seconds before reloading vehicleposition informations
	time.Sleep(10 * time.Second)

	// Get  Navitia server parameters
	navitiaUrl := d.getNavitiaUrl()
	navitiaToken := d.getNavitiaToken()
	navitiaCoverage := d.getNavitiaCoverage()
	for {
		refreshTime := d.getConnector().GetWsRefreshTime()

		newVehiclePosition, newLastUpdate, err := tryToLoadRealTimeVehiclePosition(d)
		if err != nil {
			logrus.Errorf("error while the updating of the vehicle positions: %v", err)
		} else if newLastUpdate != nil {
			d.setRealTimeVehicleJourneyCode(nil)
			{
				var utcCurrentDatetime *time.Time
				{
					utcCurrentDatetime = nil
				}
				// {
				// 	*utcCurrentDatetime = time.Date(
				// 		2022, time.October, 29,
				// 		10, 0, 0,
				// 		000_000_000,
				// 		time.UTC,
				// 	)
				// }

				vehicleJourneyCode, err := GetVehicleJourneyCodeFromNavitia(
					navitiaToken,
					d.getConnector().GetConnectionTimeout(),
					d.getLocation(),
					navitiaUrl,
					navitiaCoverage,
					utcCurrentDatetime,
				)
				if err != nil {
					logrus.Errorf(
						"error occurred cannot get the vehicle journey code from navitia: %v",
						err,
					)
				}
				d.setRealTimeVehicleJourneyCode(&vehicleJourneyCode)
			}

			d.setLastUpdate(newLastUpdate)
			d.setRealTimeVehiclePosition(newVehiclePosition)
			d.updateVehiclePositions()
			logrus.Infof("vehicle positions are updated, new last-update: %v", newLastUpdate.Format(time.RFC3339))
		}

		time.Sleep(refreshTime)
	}
}

func (d *RennesContext) GetConnectorType() connectors.ConnectorType {
	return connectors.Connector_RENNES
}

func (d *RennesContext) GetVehiclePositions(param *VehiclePositionRequestParameter) (
	vehiclePositions []VehiclePosition, e error) {
	return d.vehiclePositions.GetVehiclePositions(param)
}

func (d *RennesContext) GetLastVehiclePositionsDataUpdate() time.Time {
	return d.vehiclePositions.GetLastVehiclePositionsDataUpdate()
}

func (d *RennesContext) ManageVehiclePositionsStatus(vehiclePositionsActive bool) {
	d.vehiclePositions.ManageVehiclePositionsStatus(vehiclePositionsActive)
}

func (d *RennesContext) LoadPositionsData() bool {
	return d.vehiclePositions.LoadPositionsData()
}

func (d *RennesContext) GetRereshTime() string {
	refreshTime := d.getConnector().GetWsRefreshTime()
	return refreshTime.String()
}

/********* PRIVATE FUNCTIONS *********/

func (d *RennesContext) updateVehiclePositions() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.lastUpdate == nil {
		logrus.Debug("cannot udpate vehicle positions, no daily last update time loaded")
		return
	}
	now := time.Now().In(d.location)
	feedCreatedAt := d.lastUpdate.UTC()

	vehiclePositionMap := make(map[string]*VehiclePosition)
	if d.realTimeVehiclePosition != nil && d.realTimeVehicleJourneyCode != nil {

		var newVehiclePosition VehiclePosition
		{
			vehicleOccupancyStatusCode := int32(d.realTimeVehiclePosition.OccupancyStatus)
			vehicleOccupancyStatusName := google_transit.VehiclePosition_OccupancyStatus_name[vehicleOccupancyStatusCode]

			newVehiclePosition = VehiclePosition{
				VehicleJourneyCode: *d.realTimeVehicleJourneyCode,
				DateTime:           now,
				Latitude:           float32(d.realTimeVehiclePosition.Latitude),
				Longitude:          float32(d.realTimeVehiclePosition.Longitude),
				Bearing:            float32(d.realTimeVehiclePosition.Heading),
				Speed:              0.0,
				Occupancy:          vehicleOccupancyStatusName,
				FeedCreatedAt:      feedCreatedAt,
				Distance:           0.0,
			}
		}

		vehiclePositionMap[newVehiclePosition.VehicleJourneyCode] = &newVehiclePosition
	}

	d.vehiclePositions.UpdateVehiclePositionRennes(
		vehiclePositionMap,
		now.UTC(),
	)
}

type State = rennes_vp.State

func tryToLoadRealTimeVehiclePosition(
	d *RennesContext,
) (*rennes_vp.VehiclePosition, *time.Time, error) {
	loadedLastUpdate, err := rennes_lastupdate.LoadLastUpdate(
		d.getConnector().GetUrl(),
		header,
		d.getConnector().GetToken(),
		d.getConnector().GetConnectionTimeout(),
		d.getLocation(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error while the loading of the last-update: %v", err)
	}

	currentLastUpdate := d.getLastUpdate()

	if currentLastUpdate == nil || loadedLastUpdate.After(*currentLastUpdate) {
		var logMessage string
		if currentLastUpdate == nil {
			logMessage = fmt.Sprintf(
				"update of the real time vehicle positions, last-update: %s ...",
				loadedLastUpdate.Format(time.RFC3339),
			)
		} else {
			logMessage = fmt.Sprintf(
				"update of the real time vehicle positions, last-update: %s -> %s ...",
				currentLastUpdate.Format(time.RFC3339),
				loadedLastUpdate.Format(time.RFC3339),
			)
		}
		logrus.Info(logMessage)
		loadedVehiclePositions, err := loadRealTimeVehiclePosition(d)
		if err != nil {
			return nil, loadedLastUpdate, err
		}
		cityNavetteVehiclePosition := exctractCityNavetteVehiclePosition(loadedVehiclePositions)
		if cityNavetteVehiclePosition != nil {
			logrus.Infof(
				"realtime vehicle position has been found for the vehicle %s lcoated at (lat=%f, lon=%f)",
				cityNavetteVehiclePosition.ExternalVehiculeId,
				cityNavetteVehiclePosition.Latitude,
				cityNavetteVehiclePosition.Longitude,
			)
		} else {
			logrus.Info("no realtime vehicle position has been found")
		}
		return cityNavetteVehiclePosition, loadedLastUpdate, nil
	} else {
		logrus.Infof(
			"no newer vehicle positions available, last-update: %v",
			currentLastUpdate.Format(time.RFC3339),
		)
	}
	return nil, nil, nil

}

func loadRealTimeVehiclePosition(d *RennesContext) ([]rennes_vp.VehiclePosition, error) {
	vehiclesPositions, err := rennes_vp.LoadVehiclePositions(
		d.getConnector().GetUrl(),
		header,
		d.getConnector().GetToken(),
		d.getConnector().GetConnectionTimeout(),
	)
	if err != nil {
		VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}
	return vehiclesPositions, nil
}

func exctractCityNavetteVehiclePosition(
	vehiclePositions []rennes_vp.VehiclePosition,
) *rennes_vp.VehiclePosition {
	var CITY_NAVETTE_VEHICLE_IDS = []string{
		"7001",
		"7002",
	}

	var cityNavetteVehiclePositions []rennes_vp.VehiclePosition
	{
		cityNavetteVehiclePositions = make([]rennes_vp.VehiclePosition, 0)
		for _, vehiclePosition := range vehiclePositions {
			isCityNavette := false
			for _, vehicleId := range CITY_NAVETTE_VEHICLE_IDS {
				if vehiclePosition.ExternalVehiculeId == ExternalVehiculeId(vehicleId) {
					isCityNavette = true
					break
				}
			}
			if isCityNavette {
				cityNavetteVehiclePositions = append(cityNavetteVehiclePositions, vehiclePosition)
			}
		}
	}

	if len(cityNavetteVehiclePositions) != 0 {
		for _, cityNavetteVehiclePosition := range cityNavetteVehiclePositions {
			if cityNavetteVehiclePosition.State.IsInOperation() {
				return &cityNavetteVehiclePosition
			} else {
				logrus.Warnf(
					"vehicle positions for the vehicle %s is ignored, its state is %s",
					cityNavetteVehiclePosition.ExternalVehiculeId,
					string(cityNavetteVehiclePosition.State),
				)
			}
		}
	}

	return nil
}
