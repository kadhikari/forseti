package vehicleoccupancies

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies GTFS-RT objects
--------------------------------------------------------------------- */
type VehicleOccupanciesGtfsRtContext struct {
	voContext       *VehicleOccupanciesContext
	vehiclesJourney map[string]*VehicleJourney
	lastLoadNavitia string
	mutex           sync.RWMutex
}

var start = time.Now() // TODO: Just for test, delete to release

func (d *VehicleOccupanciesGtfsRtContext) GetVehicleOccupanciesContext() *VehicleOccupanciesContext {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.voContext == nil {
		d.voContext = &VehicleOccupanciesContext{}
	}
	return d.voContext
}

func (d *VehicleOccupanciesGtfsRtContext) CheckLastLoadChanged(lastLoadAt string) bool {
	if d.lastLoadNavitia == "" || d.lastLoadNavitia != lastLoadAt {
		d.lastLoadNavitia = lastLoadAt
		return true
	}

	return false
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListVehicleOccupancies() {
	d.voContext.CleanListVehicleOccupancies()
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListVehicleJourney() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.vehiclesJourney = nil
	logrus.Info("*** Clean list VehicleJourney")
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListOldVehicleJourney(delay time.Duration) {
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

func (d *VehicleOccupanciesGtfsRtContext) AddVehicleJourney(vehicleJourney *VehicleJourney) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.vehiclesJourney == nil {
		d.vehiclesJourney = map[string]*VehicleJourney{}
	}

	d.vehiclesJourney[vehicleJourney.CodesSource] = vehicleJourney
	logrus.Debug("*** Vehicle Journey size: ", len(d.vehiclesJourney))
}

func (d *VehicleOccupanciesGtfsRtContext) AddVehicleOccupancy(vehicleoccupancy *VehicleOccupancy) {
	d.voContext.AddVehicleOccupancy(vehicleoccupancy)
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleOccupanciesGtfsRtContext) InitContext(filesURI, externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location, occupancyActive bool) {

	d.voContext = &VehicleOccupanciesContext{}
	d.voContext.ManageVehicleOccupancyStatus(occupancyActive)
}

// main loop to refresh vehicle_occupancies from Gtfs-rt flux
func (d *VehicleOccupanciesGtfsRtContext) RefreshVehicleOccupanciesLoop(externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location) {

	// Wait 10 seconds before reloading vehicleoccupacy informations
	time.Sleep(10 * time.Second)
	for {
		err := refreshVehicleOccupancies(d, externalURI, externalToken, navitiaURI, navitiaToken,
			connectionTimeout)
		if err != nil {
			logrus.Error("Error while loading VehicleOccupancy GTFS-RT data: ", err)
		} else {
			logrus.Debug("vehicle_occupancies GTFS-RT data updated")
		}
		time.Sleep(loadExternalRefresh)
	}
}

func (d *VehicleOccupanciesGtfsRtContext) ManageVehicleOccupancyStatus(vehicleOccupanciesActive bool) {
	d.voContext.ManageVehicleOccupancyStatus(vehicleOccupanciesActive)
}

func (d *VehicleOccupanciesGtfsRtContext) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	return d.voContext.GetVehicleOccupancies(param)
}

func (d *VehicleOccupanciesGtfsRtContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	return d.voContext.GetLastVehicleOccupanciesDataUpdate()
}

func (d *VehicleOccupanciesGtfsRtContext) LoadOccupancyData() bool {
	return d.voContext.LoadOccupancyData()
}

/********* PRIVATE FUNCTIONS *********/

func loadDataExternalSource(uri url.URL, token string) (*GtfsRt, error) {

	// External source http/REST
	if token == "nil" {
		resp, err := http.Get(uri.String())
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		err = utils.CheckResponseStatus(resp)
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		gtfsRtData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		gtfsRt, err := parseVehiclesResponse(gtfsRtData)
		if err != nil {
			return nil, err
		}

		logrus.Debug("*** Gtfs-rt size: ", len(gtfsRt.Vehicles))
		return gtfsRt, nil

	} else {
		//TODO: add code in case of this service call external API
		return nil, nil
	}
}

func refreshVehicleOccupancies(context *VehicleOccupanciesGtfsRtContext, external_url url.URL,
	external_token string, navitiaURI url.URL, navitiaToken string, connectionTimeout time.Duration) error {

	begin := time.Now()
	newT := start.Add(time.Minute * 10) // TODO: Just for test, delete to release

	// Get all data from Gtfs-rt flux
	gtfsRt, err := loadDataExternalSource(external_url, external_token)
	if err != nil {
		return errors.Errorf("loading external source: %s", err)
	}
	if len(gtfsRt.Vehicles) == 0 {
		return fmt.Errorf("no data to load from GTFS-RT")
	}

	// Get status Last load from Navitia and check if data loaded recently
	publicationDate, err := GetStatusPublicationDate(navitiaURI, navitiaToken, connectionTimeout)
	if err != nil {
		logrus.Warning("Error while loading publication date from Navitia: ", err)
	}

	if context.CheckLastLoadChanged(publicationDate) {
		logrus.Info("New date of Navitia data loaded ")
		context.CleanListVehicleOccupancies()
		context.CleanListVehicleJourney()
	}

	// TODO: change to take param of config
	if newT.Before(time.Now()) {
		context.CleanListVehicleOccupancies()
		start = time.Now()
	}

	// Clean list VehicleJourney for vehicle older than delay parameter
	// TODO: add parameter in config
	context.CleanListOldVehicleJourney(1)

	manageListVehicleOccupancies(context, gtfsRt, navitiaURI, navitiaToken, connectionTimeout)

	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func manageListVehicleOccupancies(context *VehicleOccupanciesGtfsRtContext, gtfsRt *GtfsRt, navitiaURI url.URL,
	navitiaToken string, connectionTimeout time.Duration) {
	for _, vehGtfsRT := range gtfsRt.Vehicles {

		idGtfsrt, _ := strconv.Atoi(vehGtfsRT.Trip)
		var vj *VehicleJourney
		var err error

		if _, ok := context.vehiclesJourney[vehGtfsRT.Trip]; !ok {
			vj, err = GetVehicleJourney(vehGtfsRT.Trip, navitiaURI, navitiaToken, connectionTimeout)
			if err != nil {
				continue
			}

			// add in vehicle journey list
			context.AddVehicleJourney(vj)
		} else {
			vj = context.vehiclesJourney[vehGtfsRT.Trip]
		}

		var spfound = false
		for _, sp := range *vj.StopPoints {
			if sp.GtfsStopCode == vehGtfsRT.StopId {
				spfound = true
				break

			}
		}

		if _, ok := context.voContext.VehicleOccupancies[idGtfsrt]; !ok && !spfound {
			newVehicleOccupancy := createOccupanciesFromDataSource(*vj, vehGtfsRT)
			if newVehicleOccupancy != nil {
				context.AddVehicleOccupancy(newVehicleOccupancy)
			}
		}
	}
}

// Create new Vehicle occupancy from VehicleJourney and VehicleGtfsRT data
func createOccupanciesFromDataSource(vehicleJourney VehicleJourney,
	vehicleGtfsRt VehicleGtfsRt) *VehicleOccupancy {
	idGtfsrt, _ := strconv.Atoi(vehicleGtfsRt.Trip)
	date := time.Unix(int64(vehicleGtfsRt.Time), 0)
	for _, stopPoint := range *vehicleJourney.StopPoints {
		if stopPoint.GtfsStopCode == vehicleGtfsRt.StopId {
			vo, err := NewVehicleOccupancy(idGtfsrt, "", vehicleJourney.VehicleID, stopPoint.Id, -1, date,
				int(vehicleGtfsRt.Occupancy))
			if err != nil {
				continue
			}
			return vo
		}
	}
	logrus.Debugf("StopPoint: %s not found in Navitia for %s", vehicleGtfsRt.StopId, vehicleJourney.VehicleID)
	return nil
}
