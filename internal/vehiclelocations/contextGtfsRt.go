package vehiclelocations

import (
	"fmt"
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
	cleanVl          time.Duration
	location         *time.Location
	mutex            sync.RWMutex
}

var start = time.Now()

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

	for idx, v := range d.vehiclesJourney {
		if v.CreateDate.Add(delay * time.Hour).Before(time.Now()) {
			delete(d.vehiclesJourney, idx)
		}
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
	d.vehicleLocations.lastVehicleOccupanciesUpdate = time.Now()
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *GtfsRtContext) InitContext(filesURI, externalURI url.URL, externalToken string, navitiaURI url.URL,
	navitiaToken string, loadExternalRefresh, locationsCleanVJ, locationsCleanVL, connectionTimeout time.Duration,
	location *time.Location, reloadActive bool) {

	d.connector = connectors.NewConnector(filesURI, externalURI, externalToken, loadExternalRefresh, connectionTimeout)
	d.navitia = NewNavitia(navitiaURI, navitiaToken, connectionTimeout)
	d.vehicleLocations = &VehicleLocations{}
	d.location = location
	d.cleanVj = locationsCleanVJ
	d.cleanVl = locationsCleanVL
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

func (d *GtfsRtContext) GetLastVehicleLocationsDataUpdate() time.Time {
	return d.vehicleLocations.GetLastVehicleLocationsDataUpdate()
}

func (d *GtfsRtContext) LoadLocationsData() bool {
	return d.vehicleLocations.LoadLocationsData()
}

func (d *GtfsRtContext) GetRereshTime() string {
	return d.connector.GetRefreshTime().String()
}

/********* PRIVATE FUNCTIONS *********/

func refreshVehicleLocations(context *GtfsRtContext, connector *connectors.Connector,
	navitia *Navitia) error {
	begin := time.Now()
	timeCleanVL := start.Add(context.cleanVl * time.Hour)

	// Get all data from Gtfs-rt flux
	gtfsRtData, err := loadDatafromConnector(connector)
	if err != nil {
		VehicleLocationsLoadingErrors.Inc()
		return errors.Errorf("loading external source: %s", err)
	}
	if gtfsRtData == nil || len(gtfsRtData.Vehicles) == 0 {
		return fmt.Errorf("no data to load from GTFS-RT")
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

	if timeCleanVL.Before(time.Now()) {
		context.CleanListVehicleLocations()
		start = time.Now()
	}

	// Clean list VehicleJourney for vehicle older than delay parameter
	context.CleanListOldVehicleJourney(context.cleanVj)

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
	date := time.Unix(int64(vehicleGtfsRt.Time), 0).UTC()
	dateLoc, err := time.ParseInLocation("2006-01-02 15:04:05 +0000 UTC", date.String(), location)
	if err != nil {
		return &VehicleLocation{}
	}

	vl, err := NewVehicleLocation(idGtfsrt, vehicleJourney.VehicleID, dateLoc, vehicleGtfsRt.Latitude,
		vehicleGtfsRt.Longitude, vehicleGtfsRt.Bearing, vehicleGtfsRt.Speed)
	if err != nil {
		return nil
	}
	return vl
}
