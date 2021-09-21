package vehicleoccupancies

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/CanalTP/forseti/google_transit"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies GTFS-RT objects
--------------------------------------------------------------------- */
type VehicleOccupanciesGtfsRtContext struct {
	voContext *VehicleOccupanciesContext
	mutex     sync.RWMutex
}

var start = time.Now()

func (d *VehicleOccupanciesGtfsRtContext) GetVehicleOccupanciesContext() *VehicleOccupanciesContext {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.voContext == nil {
		d.voContext = &VehicleOccupanciesContext{}
	}
	return d.voContext
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListVehicleOccupancies() {
	d.voContext.CleanListVehicleOccupancies()
	logrus.Info("*** Clean list VehicleOccupancies")
}

func (d *VehicleOccupanciesGtfsRtContext) AddVehicleOccupancy(vehicleoccupancy *VehicleOccupancy) {
	d.voContext.AddVehicleOccupancy(vehicleoccupancy)
}

func (d *VehicleOccupanciesGtfsRtContext) UpdateOccupancy(vehicleoccupancy *VehicleOccupancy,
	vehGtfsRT VehicleGtfsRt, location *time.Location) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if vehicleoccupancy == nil {
		return
	}

	date := time.Unix(int64(vehGtfsRT.Time), 0).UTC()
	dateLoc, err := time.ParseInLocation("2006-01-02 15:04:05 +0000 UTC", date.String(), location)
	if err != nil {
		logrus.Info(err)
		dateLoc = vehicleoccupancy.DateTime
	}

	vehicleoccupancy.Occupancy = google_transit.VehiclePosition_OccupancyStatus_name[int32(vehGtfsRT.Occupancy)]
	vehicleoccupancy.DateTime = dateLoc
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleOccupanciesGtfsRtContext) InitContext(filesURI, externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ,
	occupancyCleanVO, connectionTimeout time.Duration, location *time.Location, occupancyActive bool) {

	d.voContext = &VehicleOccupanciesContext{}
	d.voContext.ManageVehicleOccupancyStatus(occupancyActive)
	d.voContext.SetRereshTime(loadExternalRefresh)
}

// main loop to refresh vehicle_occupancies from Gtfs-rt flux
func (d *VehicleOccupanciesGtfsRtContext) RefreshVehicleOccupanciesLoop(externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ,
	occupancyCleanVO, connectionTimeout time.Duration, location *time.Location) {

	// Wait 10 seconds before reloading vehicleoccupacy informations
	time.Sleep(10 * time.Second)
	for {
		err := refreshVehicleOccupancies(d, externalURI, externalToken, occupancyCleanVO, connectionTimeout,
			location)
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

func (d *VehicleOccupanciesGtfsRtContext) GetRereshTime() string {
	return d.voContext.GetRereshTime()
}

/********* PRIVATE FUNCTIONS *********/

func loadDataExternalSource(uri url.URL, token string) (*GtfsRt, error) {

	// External source http/REST
	if len(token) == 0 {
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
	external_token string, occupancyCleanVO, connectionTimeout time.Duration, location *time.Location) error {

	begin := time.Now()
	timeCleanVO := start.Add(occupancyCleanVO * time.Hour)

	// Get all data from Gtfs-rt flux
	gtfsRt, err := loadDataExternalSource(external_url, external_token)
	if err != nil {
		return errors.Errorf("loading external source: %s", err)
	}
	if gtfsRt == nil || len(gtfsRt.Vehicles) == 0 {
		return fmt.Errorf("no data to load from GTFS-RT")
	}

	// Clean list VehicleOccupancies for vehicle older than delay parameter: occupancyCleanVO
	if timeCleanVO.Before(time.Now()) {
		context.CleanListVehicleOccupancies()
		start = time.Now()
	}

	// Add or update vehicle occupancy with vehicle GTFS-RT
	for _, vehGtfsRT := range gtfsRt.Vehicles {
		if _, ok := context.voContext.VehicleOccupancies[vehGtfsRT.Trip]; !ok {
			newVehicleOccupancy := createOccupanciesFromDataSource(vehGtfsRT, location)
			if newVehicleOccupancy != nil {
				context.AddVehicleOccupancy(newVehicleOccupancy)
			}
		} else {
			stopCodeFind := false
			for _, vo := range context.voContext.VehicleOccupancies {
				if vo.SourceCode == vehGtfsRT.Trip && vo.StopCode == vehGtfsRT.StopId {
					context.UpdateOccupancy(vo, vehGtfsRT, location)
					stopCodeFind = true
					break
				}
			}
			if !stopCodeFind {
				newVehicleOccupancy := createOccupanciesFromDataSource(vehGtfsRT, location)
				if newVehicleOccupancy != nil {
					context.AddVehicleOccupancy(newVehicleOccupancy)
				}
			}
		}
	}

	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

// Create new Vehicle occupancy from VehicleJourney and VehicleGtfsRT data
func createOccupanciesFromDataSource(vehicleGtfsRt VehicleGtfsRt, location *time.Location) *VehicleOccupancy {

	date := time.Unix(int64(vehicleGtfsRt.Time), 0).UTC()
	dateLoc, err := time.ParseInLocation("2006-01-02 15:04:05 +0000 UTC", date.String(), location)
	if err != nil {
		logrus.Info(err)
		return &VehicleOccupancy{}
	}

	vo, err := NewVehicleOccupancy(vehicleGtfsRt.Trip, vehicleGtfsRt.StopId, -1, dateLoc,
		google_transit.VehiclePosition_OccupancyStatus_name[int32(vehicleGtfsRt.Occupancy)])
	if err != nil {
		return &VehicleOccupancy{}
	}
	return vo

}
