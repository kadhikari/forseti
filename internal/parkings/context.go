package parkings

import (
	"fmt"
	"sync"
	"time"
)

type ParkingsContext struct {
	parkings          *map[string]Parking
	lastParkingUpdate time.Time
	parkingsMutex     sync.RWMutex
}

func (d *ParkingsContext) UpdateParkings(parkings map[string]Parking) {
	d.parkingsMutex.Lock()
	defer d.parkingsMutex.Unlock()

	d.parkings = &parkings
	d.lastParkingUpdate = time.Now()
}

func (d *ParkingsContext) GetLastParkingsDataUpdate() time.Time {
	d.parkingsMutex.RLock()
	defer d.parkingsMutex.RUnlock()

	return d.lastParkingUpdate
}

func (d *ParkingsContext) GetParkingsByIds(ids []string) (parkings []Parking, errors []error) {
	for _, id := range ids {
		if p, err := d.GetParkingById(id); err == nil {
			parkings = append(parkings, p)
		} else {
			errors = append(errors, err)
		}
	}
	return
}

func (d *ParkingsContext) GetParkings() (parkings []Parking, e error) {
	var mapParkings map[string]Parking
	{
		d.parkingsMutex.RLock()
		defer d.parkingsMutex.RUnlock()

		if d.parkings == nil {
			e = fmt.Errorf("No parkings in the data")
			return
		}

		mapParkings = *d.parkings
	}

	// Convert Map of parkings to Slice !
	parkings = make([]Parking, 0, len(mapParkings))
	for _, p := range mapParkings {
		parkings = append(parkings, p)
	}

	return parkings, nil
}

func (d *ParkingsContext) GetParkingById(id string) (p Parking, e error) {
	var ok bool
	{
		d.parkingsMutex.RLock()
		defer d.parkingsMutex.RUnlock()

		if d.parkings == nil {
			e = fmt.Errorf("No parkings in the data")
			return
		}

		parkings := *d.parkings
		p, ok = parkings[id]
	}

	if !ok {
		e = fmt.Errorf("No parkings found with id: %s", id)
	}

	return p, e
}
