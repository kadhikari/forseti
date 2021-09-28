package vehicleoccupanciesv2

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
	VehicleOccupancies           map[int]*VehicleOccupancy
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

func (d *VehicleOccupanciesContext) UpdateVehicleOccupancies(vehicleOccupancies map[int]*VehicleOccupancy) {
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
		d.VehicleOccupancies = map[int]*VehicleOccupancy{}
	}

	d.VehicleOccupancies[vehicleoccupancy.Id] = vehicleoccupancy
}

func (d *VehicleOccupanciesContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *VehicleOccupanciesContext) GetVehiclesOccupancies() (vehicleOccupancies map[int]*VehicleOccupancy) {
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
			// Filter on stop_code
			foundStopCode := FoundStopCode(*vo, param.StopCodes)

			// Filter on vehiclejourney_code
			foundVjCode := FoundVjCode(*vo, param.VehicleJourneyCodes)

			//Fileter on datetime (default value Now)
			if vo.DateTime.Before(param.Date) {
				//println("****************************** DATE BEFORE")
				continue
			}
			//println("****************************** DATE AFTER")
			//println("****************************** ", foundStopCode, foundVjCode)

			if len(param.StopCodes) == 0 || len(param.VehicleJourneyCodes) == 0 {
				if foundVjCode && foundStopCode {
					occupancies = append(occupancies, *vo)
				}
			} else if foundVjCode || foundStopCode {
				occupancies = append(occupancies, *vo)
			}
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

func NewVehicleOccupancy(id int, sourceCode, stopCode string, direction int, date time.Time,
	occupancy string) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		Id:                 id,
		VehicleJourneyCode: sourceCode,
		StopCode:           stopCode,
		Direction:          direction,
		DateTime:           date,
		Occupancy:          occupancy,
	}, nil
}

func FoundVjCode(vp VehicleOccupancy, vjCodes []string) bool {
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

func FoundStopCode(vo VehicleOccupancy, stopCodes []string) bool {
	if len(stopCodes) == 0 {
		return true
	}
	for _, stopCode := range stopCodes {
		if vo.StopCode == stopCode {
			return true
		}
	}
	return false
}
