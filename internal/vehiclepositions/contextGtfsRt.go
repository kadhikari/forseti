package vehiclepositions

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/CanalTP/forseti/google_transit"
	"github.com/CanalTP/forseti/internal/connectors"
	gtfsrtvehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle positions GTFS-RT objects
--------------------------------------------------------------------- */
type GtfsRtContext struct {
	vehiclePositions *VehiclePositions
	connector        *connectors.Connector
	cleanVp          time.Duration
	location         *time.Location
	mutex            sync.RWMutex
}

var start = time.Now()

func (d *GtfsRtContext) GetAllVehiclePositions() *VehiclePositions {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.vehiclePositions == nil {
		d.vehiclePositions = &VehiclePositions{}
	}
	return d.vehiclePositions
}

func (d *GtfsRtContext) CleanListVehiclePositions(timeCleanVP time.Duration) {
	d.vehiclePositions.CleanListVehiclePositions(timeCleanVP)
}

func (d *GtfsRtContext) GetVehiclePositions(param *VehiclePositionRequestParameter) (
	vehiclePositions []VehiclePosition, e error) {
	return d.vehiclePositions.GetVehiclePositions(param)
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *GtfsRtContext) InitContext(filesURI, externalURI url.URL, externalToken string, loadExternalRefresh,
	positionsCleanVP, connectionTimeout time.Duration, location *time.Location, reloadActive bool) {

	d.connector = connectors.NewConnector(filesURI, externalURI, externalToken, loadExternalRefresh, connectionTimeout)
	d.vehiclePositions = &VehiclePositions{}
	d.location = location
	d.cleanVp = positionsCleanVP
	d.vehiclePositions.ManageVehiclePositionsStatus(reloadActive)
}

// main loop to refresh vehicle_positions
func (d *GtfsRtContext) RefreshVehiclePositionsLoop() {
	// Wait 10 seconds before reloading vehicleposition informations
	time.Sleep(10 * time.Second)
	for {
		err := refreshVehiclePositions(d, d.connector)
		if err != nil {
			logrus.Error("Error while loading VehiclePositions GTFS-RT data: ", err)
		} else {
			logrus.Info("Vehicle_positions data updated, list size: ", len(d.vehiclePositions.vehiclePositions))
		}
		time.Sleep(d.connector.GetRefreshTime())
	}
}

func (d *GtfsRtContext) GetLastVehiclePositionsDataUpdate() time.Time {
	return d.vehiclePositions.GetLastVehiclePositionsDataUpdate()
}

func (d *GtfsRtContext) ManageVehiclePositionsStatus(vehiclePositionsActive bool) {
	d.vehiclePositions.ManageVehiclePositionsStatus(vehiclePositionsActive)
}

func (d *GtfsRtContext) LoadPositionsData() bool {
	return d.vehiclePositions.LoadPositionsData()
}

func (d *GtfsRtContext) GetRereshTime() string {
	return d.connector.GetRefreshTime().String()
}

/********* PRIVATE FUNCTIONS *********/

func refreshVehiclePositions(context *GtfsRtContext, connector *connectors.Connector) error {

	if !context.LoadPositionsData() {
		return nil
	}

	begin := time.Now()
	timeCleanVP := start.Add(context.cleanVp)

	// Get all data from Gtfs-rt flux
	gtfsRt, err := loadDatafromConnector(connector)
	if err != nil {
		VehiclePositionsLoadingErrors.Inc()
		return errors.Errorf("loading external source: %s", err)
	}
	if gtfsRt == nil || len(gtfsRt.Vehicles) == 0 {
		return fmt.Errorf("no data to load from GTFS-RT")
	}

	if timeCleanVP.Before(time.Now()) {
		context.CleanListVehiclePositions(context.cleanVp)
		start = time.Now()
	}

	// Add or update vehicle position with vehicle GTFS-RT
	for _, vehGtfsRT := range gtfsRt.Vehicles {
		oldVp := context.vehiclePositions.vehiclePositions[vehGtfsRT.Trip]
		if oldVp == nil {
			newVehiclePosition := createVehiclePositionFromDataSource(vehGtfsRT, context.location)
			if newVehiclePosition != nil {
				context.vehiclePositions.AddVehiclePosition(newVehiclePosition)
			}
		} else {
			context.vehiclePositions.UpdateVehiclePosition(vehGtfsRT, context.location)
		}
	}

	VehiclePositionsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func loadDatafromConnector(connector *connectors.Connector) (*gtfsrtvehiclepositions.GtfsRt, error) {

	gtfsRtData, err := gtfsrtvehiclepositions.LoadGtfsRt(connector)
	if err != nil {
		return nil, err
	}

	return gtfsRtData, nil
}

// Create new Vehicle position from VehicleGtfsRT data
func createVehiclePositionFromDataSource(vehicleGtfsRt gtfsrtvehiclepositions.VehicleGtfsRt,
	location *time.Location) *VehiclePosition {

	date := time.Unix(int64(vehicleGtfsRt.Time), 0).UTC().Format("2006-01-02T15:04:05Z")
	d, erro := time.Parse("2006-01-02T15:04:05Z", date)
	if erro != nil {
		logrus.Error("Impossible to parse datetime, reason: ", erro)
		return &VehiclePosition{}
	}
	dateLoc, err := time.ParseInLocation("2006-01-02T15:04:05Z", date, location)
	if err != nil {
		logrus.Error("Impossible to parse datetime with location, reason: ", erro)
		return &VehiclePosition{}
	}

	vp, err := NewVehiclePosition(vehicleGtfsRt.Trip, dateLoc, vehicleGtfsRt.Latitude,
		vehicleGtfsRt.Longitude, vehicleGtfsRt.Bearing, vehicleGtfsRt.Speed,
		google_transit.VehiclePosition_OccupancyStatus_name[int32(vehicleGtfsRt.Occupancy)], d)
	if err != nil {
		logrus.Error("Impossible to create new vehicle position, reason: ", erro)
		return nil
	}
	return vp
}
