package vehiclepositions

import (
	"fmt"
	"sync"
	"time"

	"github.com/CanalTP/forseti/google_transit"
	gtfsrtvehiclepositions "github.com/CanalTP/forseti/internal/gtfsRt_vehiclepositions"
	"github.com/sirupsen/logrus"
)

/* -------------------------------------------------------------
// Structure and Consumer to creates Vehicle locations objects
------------------------------------------------------------- */
type VehiclePositions struct {
	vehiclePositions           map[string]*VehiclePosition
	lastVehiclePositionsUpdate time.Time
	loadOccupancyData          bool
	mutex                      sync.RWMutex
}

func (d *VehiclePositions) ManageVehiclePositionsStatus(activate bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.loadOccupancyData = activate
}

func (d *VehiclePositions) CleanListVehiclePositions(timeCleanVP time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	cpt := 0
	dateBefore := time.Now().UTC().Add(-timeCleanVP)

	if d.vehiclePositions != nil {
		for k, vo := range d.vehiclePositions {
			if vo.FeedCreatedAt.Before(dateBefore) {
				delete(d.vehiclePositions, k)
				cpt += 1
			}
		}
	}
	logrus.Info("*** Number of clean VehiclePositions: ", cpt)
	logrus.Info("*** Number of VehiclePositions: ", len(d.vehiclePositions))
}

func (d *VehiclePositions) AddVehiclePosition(vehiclelocation *VehiclePosition) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.vehiclePositions == nil {
		d.vehiclePositions = map[string]*VehiclePosition{}
	}

	d.vehiclePositions[vehiclelocation.VehicleJourneyCode] = vehiclelocation
	d.lastVehiclePositionsUpdate = time.Now().UTC()
}

func (d *VehiclePositions) UpdateVehiclePosition(vehicleGtfsRt gtfsrtvehiclepositions.VehicleGtfsRt,
	location *time.Location) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.vehiclePositions == nil {
		return
	}

	d.vehiclePositions[vehicleGtfsRt.Trip].Latitude = vehicleGtfsRt.Latitude
	d.vehiclePositions[vehicleGtfsRt.Trip].Longitude = vehicleGtfsRt.Longitude
	d.vehiclePositions[vehicleGtfsRt.Trip].Bearing = vehicleGtfsRt.Bearing
	d.vehiclePositions[vehicleGtfsRt.Trip].Speed = vehicleGtfsRt.Speed
	d.vehiclePositions[vehicleGtfsRt.Trip].Occupancy =
		google_transit.VehiclePosition_OccupancyStatus_name[int32(vehicleGtfsRt.Occupancy)]
	d.vehiclePositions[vehicleGtfsRt.Trip].FeedCreatedAt = time.Unix(int64(vehicleGtfsRt.Time), 0).UTC()
	d.lastVehiclePositionsUpdate = time.Now()
}

func (d *VehiclePositions) GetVehiclePositions(param *VehiclePositionRequestParameter) (
	vehicleOccupancies []VehiclePosition, e error) {
	var positions []VehiclePosition
	{
		d.mutex.RLock()
		defer d.mutex.RUnlock()

		if d.vehiclePositions == nil {
			e = fmt.Errorf("no vehicle_locations in the data")
			return
		}

		// Implement filter on parameters
		for _, vp := range d.vehiclePositions {

			foundIt := FoundIt(*vp, param.VehicleJourneyCodes)

			if vp.DateTime.Before(param.Date) {
				continue
			}

			if foundIt {
				positions = append(positions, *vp)
			}
		}
		return positions, nil
	}
}

func (d *VehiclePositions) GetLastVehiclePositionsDataUpdate() time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.lastVehiclePositionsUpdate
}

func (d *VehiclePositions) LoadPositionsData() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.loadOccupancyData
}

func NewVehiclePosition(sourceCode string, date time.Time, lat float32, lon float32, bearing float32,
	speed float32, occupancy string, feedCreateAt time.Time) (*VehiclePosition, error) {
	return &VehiclePosition{
		VehicleJourneyCode: sourceCode,
		DateTime:           date,
		Latitude:           lat,
		Longitude:          lon,
		Bearing:            bearing,
		Speed:              speed,
		Occupancy:          occupancy,
		FeedCreatedAt:      feedCreateAt,
	}, nil
}

func FoundIt(vp VehiclePosition, vjCodes []string) bool {
	if len(vjCodes) == 0 {
		return true
	}
	for _, vjCode := range vjCodes {
		if vp.VehicleJourneyCode == vjCode {
			return true
		}
	}
	return false
}
