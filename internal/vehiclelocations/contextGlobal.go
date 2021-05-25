package vehiclelocations

import (
	"fmt"
	"sync"
	"time"
)

/* -------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies objects
------------------------------------------------------------- */
type VehicleLocationsContext struct {
	VehicleLocations map[int]*VehicleLocation
	mutex            sync.RWMutex
}

func (d *VehicleLocationsContext) GetVehicleLocations(param *VehicleLocationRequestParameter) (
	vehicleOccupancies []VehicleLocation, e error) {
	var locations []VehicleLocation
	{
		d.mutex.RLock()
		defer d.mutex.RUnlock()

		if d.VehicleLocations == nil {
			e = fmt.Errorf("no vehicle_occupancies in the data")
			return
		}

		// Implement filter on parameters
		for _, vl := range d.VehicleLocations {
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

func NewVehicleLocation(vlId int, vjId, stopId string, date time.Time, lat float32, lon float32,
	bearing float32, speed float32) *VehicleLocation {
	return &VehicleLocation{
		Id:               vlId,
		VehicleJourneyId: vjId,
		StopId:           stopId,
		DateTime:         date,
		Latitude:         lat,
		Longitude:        lon,
		Bearing:          bearing,
		Speed:            speed,
	}
}
