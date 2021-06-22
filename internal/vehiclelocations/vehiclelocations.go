package vehiclelocations

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

/* -------------------------------------------------------------
// Structure and Consumer to creates Vehicle locations objects
------------------------------------------------------------- */
type VehicleLocations struct {
	vehicleLocations             map[int]*VehicleLocation
	lastVehicleOccupanciesUpdate time.Time
	loadOccupancyData            bool
	mutex                        sync.RWMutex
}

func (d *VehicleLocations) ManageVehicleLocationsStatus(activate bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.loadOccupancyData = activate
}

func (d *VehicleLocations) CleanListVehicleLocations() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.vehicleLocations != nil {
		for k := range d.vehicleLocations {
			delete(d.vehicleLocations, k)
		}
	}
	logrus.Info("*** Clean list VehicleLocations")
}

func (d *VehicleLocations) AddVehicleLocation(vehiclelocation *VehicleLocation) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.vehicleLocations == nil {
		d.vehicleLocations = map[int]*VehicleLocation{}
	}

	d.vehicleLocations[vehiclelocation.Id] = vehiclelocation
	logrus.Debug("*** Vehicle Locations size: ", len(d.vehicleLocations))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleLocations) UpdateVehicleLocation(vehicleGtfsRt VehicleGtfsRt, location *time.Location) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.vehicleLocations == nil {
		return
	}

	idGtfsrt, _ := strconv.Atoi(vehicleGtfsRt.Trip)
	d.vehicleLocations[idGtfsrt].Latitude = vehicleGtfsRt.Latitude
	d.vehicleLocations[idGtfsrt].Longitude = vehicleGtfsRt.Longitude
	d.vehicleLocations[idGtfsrt].Bearing = vehicleGtfsRt.Bearing
	d.vehicleLocations[idGtfsrt].Speed = vehicleGtfsRt.Speed
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleLocations) GetVehicleLocations(param *VehicleLocationRequestParameter) (
	vehicleOccupancies []VehicleLocation, e error) {
	var locations []VehicleLocation
	{
		d.mutex.RLock()
		defer d.mutex.RUnlock()

		if d.vehicleLocations == nil {
			e = fmt.Errorf("no vehicle_locations in the data")
			return
		}

		// Implement filter on parameters
		for _, vl := range d.vehicleLocations {
			// Filter on vehiclejourney_id
			if len(param.VehicleJourneyId) > 0 && param.VehicleJourneyId != vl.VehicleJourneyId {
				continue
			}
			//Fileter on datetime (default value Now)
			if vl.DateTime.Before(param.Date) {
				continue
			}
			locations = append(locations, *vl)
		}
		return locations, nil
	}
}

func (d *VehicleLocations) GetLastVehicleLocationsDataUpdate() time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *VehicleLocations) LoadLocationsData() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.loadOccupancyData
}

func NewVehicleLocation(vlId int, vjId string, date time.Time, lat float32, lon float32,
	bearing float32, speed float32) (*VehicleLocation, error) {
	return &VehicleLocation{
		Id:               vlId,
		VehicleJourneyId: vjId,
		DateTime:         date,
		Latitude:         lat,
		Longitude:        lon,
		Bearing:          bearing,
		Speed:            speed,
	}, nil
}
