package vehiclepositions

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/hove-io/forseti/google_transit"
	"github.com/hove-io/forseti/internal/connectors"
	gtfsrtvehiclepositions "github.com/hove-io/forseti/internal/gtfsRt_vehiclepositions"
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

func (d *GtfsRtContext) InitContext(
	filesUrl url.URL,
	filesRefreshDuration time.Duration,
	externalServiceUrl url.URL,
	externalServiceToken string,
	externalServiceRefreshDuration time.Duration,
	navitiaUrl url.URL,
	navitiaToken string,
	navitiaCoverageName string,
	connectionTimeout time.Duration,
	positionCleanVP time.Duration,
	location *time.Location,
	reloadActive bool,
) {
	d.connector = connectors.NewConnector(
		filesUrl,
		externalServiceUrl,
		externalServiceToken,
		filesRefreshDuration,
		externalServiceRefreshDuration,
		connectionTimeout,
	)
	d.vehiclePositions = &VehiclePositions{}
	d.location = location
	d.cleanVp = positionCleanVP
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
		time.Sleep(d.connector.GetWsRefreshTime())
	}
}

func (d *GtfsRtContext) GetLastVehiclePositionsDataUpdate() time.Time {
	return d.vehiclePositions.GetLastVehiclePositionsDataUpdate()
}

func (d *GtfsRtContext) GetLastStatusUpdate() time.Time {
	return d.vehiclePositions.GetLastStatusUpdate()
}

func (d *GtfsRtContext) ManageVehiclePositionsStatus(vehiclePositionsActive bool) {
	d.vehiclePositions.ManageVehiclePositionsStatus(vehiclePositionsActive)
}

func (d *GtfsRtContext) LoadPositionsData() bool {
	return d.vehiclePositions.LoadPositionsData()
}

func (d *GtfsRtContext) GetRereshTime() string {
	return d.connector.GetFilesRefreshTime().String()
}

func (d *GtfsRtContext) GetStatus() string {
	return d.vehiclePositions.GetStatus()
}

func (d *GtfsRtContext) SetStatus(status string) {
	d.vehiclePositions.SetStatus(status)
}

func (d *GtfsRtContext) GetConnectorType() connectors.ConnectorType {
	return connectors.Connector_GRFS_RT
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
	cpt := 0
	for _, vehGtfsRT := range gtfsRt.Vehicles {
		if vehGtfsRT.Latitude == 0 && vehGtfsRT.Longitude == 0 {
			cpt += 1
			continue
		}
		oldVp := context.vehiclePositions.vehiclePositions[vehGtfsRT.Trip]
		if oldVp == nil {
			newVehiclePosition := createVehiclePositionFromDataSource(vehGtfsRT, context.location)
			if newVehiclePosition != nil {
				context.vehiclePositions.AddVehiclePosition(newVehiclePosition)
			}
		} else {
			context.vehiclePositions.UpdateVehiclePositionGtfsRt(vehGtfsRT, context.location)
		}
	}
	if cpt > 0 {
		logrus.Warn("Count gtfs-rt items ignored ", cpt, ", reason: items without coordinates")
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
