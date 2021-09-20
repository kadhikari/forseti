package vehicleoccupancies

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

/* -------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies objects
------------------------------------------------------------- */
type VehicleOccupanciesContext struct {
	VehicleOccupancies           map[string]*VehicleOccupancy
	lastVehicleOccupanciesUpdate time.Time
	vehicleOccupanciesMutex      sync.RWMutex
	loadOccupancyData            bool
	refreshTime                  time.Duration
}

func (d *VehicleOccupanciesContext) ManageVehicleOccupancyStatus(activate bool) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.loadOccupancyData = activate
}

func (d *VehicleOccupanciesContext) LoadOccupancyData() bool {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	return d.loadOccupancyData
}

func (d *VehicleOccupanciesContext) UpdateVehicleOccupancies(vehicleOccupancies map[string]*VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.VehicleOccupancies = vehicleOccupancies
	logrus.Info("*** vehicleOccupancies size: ", len(d.VehicleOccupancies))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) CleanListVehicleOccupancies() {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	if d.VehicleOccupancies != nil {
		for k := range d.VehicleOccupancies {
			delete(d.VehicleOccupancies, k)
		}
	}
	logrus.Info("*** Clean list VehicleOccupancies")
}
func (d *VehicleOccupanciesContext) AddVehicleOccupancy(vehicleoccupancy *VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	if d.VehicleOccupancies == nil {
		d.VehicleOccupancies = map[string]*VehicleOccupancy{}
	}

	d.VehicleOccupancies[vehicleoccupancy.Id] = vehicleoccupancy
}

func (d *VehicleOccupanciesContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *VehicleOccupanciesContext) GetVehiclesOccupancies() (vehicleOccupancies map[string]*VehicleOccupancy) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.VehicleOccupancies
}

func (d *VehicleOccupanciesContext) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	var occupancies []VehicleOccupancy
	{
		d.vehicleOccupanciesMutex.RLock()
		defer d.vehicleOccupanciesMutex.RUnlock()

		if d.VehicleOccupancies == nil {
			e = fmt.Errorf("no vehicle_occupancies in the data")
			return
		}

		// Implement filter on parameters
		for _, vo := range d.VehicleOccupancies {
			// Filter on stop_id
			if len(param.StopId) > 0 && param.StopId != vo.StopCode {
				continue
			}
			// Filter on vehiclejourney_id
			if len(param.VehicleJourneyId) > 0 && param.VehicleJourneyId != vo.VehicleJourneyCode {
				continue
			}
			//Fileter on datetime (default value Now)
			if vo.DateTime.Before(param.Date) {
				continue
			}
			occupancies = append(occupancies, *vo)
		}
		return occupancies, nil
	}
}

func (d *VehicleOccupanciesContext) GetRereshTime() string {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()
	return d.refreshTime.String()
}

func (d *VehicleOccupanciesContext) SetRereshTime(newRefreshTime time.Duration) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()
	d.refreshTime = newRefreshTime
}

func NewVehicleOccupancy(sourceCode, stopCode string, direction int, date time.Time,
	occupancy string) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		Id:                 sourceCode,
		VehicleJourneyCode: sourceCode,
		StopCode:           stopCode,
		Direction:          direction,
		DateTime:           date,
		Occupancy:          occupancy,
	}, nil
}
