package vehiclelocations

import (
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle locations GTFS-RT objects
--------------------------------------------------------------------- */
type GtfsRtContext struct {
	vehicleLocations *VehicleLocations
	vehiclesJourney  map[string]*VehicleJourney
	connector        *connectors.Connector
	navitia          *Navitia
	cleanVj          time.Duration
	cleanVl          time.Time
	location         *time.Location
	mutex            sync.RWMutex
}

func (d *GtfsRtContext) GetAllVehicleLocations() *VehicleLocations {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.vehicleLocations == nil {
		d.vehicleLocations = &VehicleLocations{}
	}
	return d.vehicleLocations
}

func (d *GtfsRtContext) CleanListVehicleLocations() {
	d.vehicleLocations.CleanListVehicleLocations()
}

func (d *GtfsRtContext) CleanListVehicleJourney() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.vehiclesJourney = nil
	logrus.Info("*** Clean list VehicleJourney")
}

func (d *GtfsRtContext) GetVehicleLocations(param *VehicleLocationRequestParameter) (
	vehicleLocations []VehicleLocation, e error) {
	return d.vehicleLocations.GetVehicleLocations(param)
}

func (d *GtfsRtContext) CleanListOldVehicleJourney(delay time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	nbVj := len(d.vehiclesJourney) // TODO: Just for test, delete to release
	del := false                   // TODO: Just for test, delete to release

	for idx, v := range d.vehiclesJourney {
		if v.CreateDate.Add(delay * time.Hour).Before(time.Now()) {
			delete(d.vehiclesJourney, idx)
			del = true
		}
	}
	if del {
		logrus.Debugf("*** Clean old VehicleJourney until %d hour - %d/%d", delay, nbVj, len(d.vehiclesJourney))
	}
}

func (d *GtfsRtContext) AddVehicleJourney(vehicleJourney *VehicleJourney) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.vehiclesJourney == nil {
		d.vehiclesJourney = map[string]*VehicleJourney{}
	}

	d.vehiclesJourney[vehicleJourney.CodesSource] = vehicleJourney
	logrus.Debug("*** Vehicle Journey size: ", len(d.vehiclesJourney))
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *GtfsRtContext) InitContext(filesURI, externalURI url.URL, externalToken string, navitiaURI url.URL,
	navitiaToken string, loadExternalRefresh, connectionTimeout time.Duration, location *time.Location, reloadActive bool) {

	d.connector = connectors.NewConnector(filesURI, externalURI, externalToken, loadExternalRefresh, connectionTimeout)
	d.navitia = NewNavitia(navitiaURI, navitiaToken, connectionTimeout)
	d.vehicleLocations = &VehicleLocations{}
	d.location = location
	d.vehicleLocations.ManageVehicleLocationsStatus(reloadActive)
}

// main loop to refresh vehicle_locations
func (d *GtfsRtContext) RefreshVehicleLocationsLoop() {
	// Wait 10 seconds before reloading vehiclelocation informations
	time.Sleep(10 * time.Second)
	for {
		err := refreshVehicleLocations(d, d.connector, d.navitia)
		if err != nil {
			logrus.Error("Error while loading VehicleLocations GTFS-RT data: ", err)
		} else {
			logrus.Debug("vehicle_locations GTFS-RT data updated")
		}
		time.Sleep(d.connector.GetRefreshTime())
	}
}

/********* PRIVATE FUNCTIONS *********/

func refreshVehicleLocations(context *GtfsRtContext, connector *connectors.Connector,
	navitia *Navitia) error {
	begin := time.Now()

	// Get all data from Gtfs-rt flux
	gtfsRtData, err := loadDatafromConnector(connector)
	if err != nil {
		VehicleLocationsLoadingErrors.Inc()
		return errors.Errorf("loading external source: %s", err)
	}

	// Get status Last load from Navitia and check if data loaded recently
	publicationDate, err := GetStatusPublicationDate(navitia)
	if err != nil {
		VehicleLocationsLoadingErrors.Inc()
		logrus.Warning("Error while loading publication date from Navitia: ", err)
	}

	if navitia.CheckLastLoadChanged(publicationDate) {
		context.CleanListVehicleLocations()
		context.CleanListVehicleJourney()
	}

	if context.cleanVl.Before(time.Now()) { // TODO: change to take param of config
		context.CleanListVehicleLocations()
	}

	// Clean list VehicleJourney for vehicle older than delay parameter
	context.CleanListOldVehicleJourney(context.cleanVj) // TODO: add parameter in config

	manageListVehicleLocations(context, gtfsRtData, navitia)

	VehicleLocationsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func loadDatafromConnector(connector *connectors.Connector) (*GtfsRt, error) {

	gtfsRtData, err := LoadGtfsRt(connector)
	if err != nil {
		return nil, err
	}

	logrus.Debug("*** Gtfs-rt size: ", len(gtfsRtData.Vehicles))
	return gtfsRtData, nil
}

func manageListVehicleLocations(context *GtfsRtContext, gtfsRt *GtfsRt, navitia *Navitia) {

	for _, vehGtfsRT := range gtfsRt.Vehicles {
		idGtfsrt, _ := strconv.Atoi(vehGtfsRT.Trip)
		var vj *VehicleJourney
		var err error

		if _, ok := context.vehiclesJourney[vehGtfsRT.Trip]; !ok {
			vj, err = GetVehicleJourney(vehGtfsRT.Trip, navitia)
			if err != nil {
				VehicleLocationsLoadingErrors.Inc()
				continue
			}

			// add in vehicle journey list
			context.AddVehicleJourney(vj)
		} else {
			vj = context.vehiclesJourney[vehGtfsRT.Trip]
		}

		// if gtfs-rt vehicle not exist in map of vehicle locations
		if _, ok := context.vehicleLocations.vehicleLocations[idGtfsrt]; !ok {
			// add in vehicle locations list
			newVehicleLocation := createVehicleLocationFromDataSource(*vj, vehGtfsRT, context.location)
			if newVehicleLocation != nil {
				context.vehicleLocations.AddVehicleLocation(newVehicleLocation)
			}
		} else {
			context.vehicleLocations.UpdateVehicleLocation(vehGtfsRT, context.location)
		}
	}
}

// Create new Vehicle occupancy from VehicleJourney and VehicleGtfsRT data
func createVehicleLocationFromDataSource(vehicleJourney VehicleJourney,
	vehicleGtfsRt VehicleGtfsRt, location *time.Location) *VehicleLocation {

	idGtfsrt, _ := strconv.Atoi(vehicleGtfsRt.Trip)
	timestamp := time.Unix(int64(vehicleGtfsRt.Time), 0)
	date, err := time.ParseInLocation("2006-01-02T15:04:05Z", timestamp.Format(time.RFC3339), location)
	if err != nil {
		return &VehicleLocation{}
	}

	vl, err := NewVehicleLocation(idGtfsrt, vehicleJourney.VehicleID, date, vehicleGtfsRt.Latitude,
		vehicleGtfsRt.Longitude, vehicleGtfsRt.Bearing, vehicleGtfsRt.Speed)
	if err != nil {
		return nil
	}
	return vl
}
