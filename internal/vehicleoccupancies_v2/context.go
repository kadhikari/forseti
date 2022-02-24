package vehicleoccupanciesv2

import (
	"fmt"
	"sync"
	"time"
	"github.com/google/uuid"
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

func (d *VehicleOccupanciesContext) CleanListVehicleOccupancies(timeCleanVO time.Duration) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()
	cpt := 0

	if d.VehicleOccupancies != nil {
		for k, vo := range d.VehicleOccupancies {
			if vo.DateTime.Before(time.Now().Add(-timeCleanVO).UTC()) {
				delete(d.VehicleOccupancies, k)
				cpt += 1
			}
		}
	}
	logrus.Info("*** Number of clean VehicleOccupancies: ", cpt)
}
func (d *VehicleOccupanciesContext) AddVehicleOccupancy(vehicleoccupancy *VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	if d.VehicleOccupancies == nil {
		d.VehicleOccupancies = map[string]*VehicleOccupancy{}
	}
	key := getKeyVehicleOccupancy()
	d.VehicleOccupancies[key] = vehicleoccupancy
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
			// Filter on stop_code
			if ok := FoundStopCode(*vo, param.StopPointCodes); !ok {
				continue
			}

			// Filter on vehiclejourney_code
			if ok := FoundVjCode(*vo, param.VehicleJourneyCodes); !ok {
				continue
			}

			//Filter on datetime (default value Now)
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

func getKeyVehicleOccupancy() string {
	return uuid.New().String()
}

func NewVehicleOccupancy(sourceCode, stopPointCode string, direction int, date time.Time,
	occupancy string) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		VehicleJourneyCode: sourceCode,
		StopPointCode:      stopPointCode,
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

func FoundStopCode(vo VehicleOccupancy, stopPointCode []string) bool {
	if len(stopPointCode) == 0 {
		return true
	}
	for _, stopCode := range stopPointCode {
		if vo.StopPointCode == stopCode {
			return true
		}
	}
	return false
}
