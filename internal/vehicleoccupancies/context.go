package vehicleoccupancies

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
)

type VehicleOccupanciesContext struct {
	vehicleOccupancies           *map[int]VehicleOccupancy
	lastVehicleOccupanciesUpdate time.Time
	vehicleOccupanciesMutex      sync.RWMutex
	loadOccupancyData            bool

	stopPoints     *map[string]StopPoint
	courses        *map[string][]Course
	routeSchedules *[]RouteSchedule
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

func (d *VehicleOccupanciesContext) InitStopPoint(stopPoints map[string]StopPoint) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.stopPoints = &stopPoints
	fmt.Println("*** stopPoints size: ", len(*d.stopPoints))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) GetStopId(name string, sens int) (id string) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	key := name + strconv.Itoa(sens)
	return (*d.stopPoints)[key].Id
}

func (d *VehicleOccupanciesContext) InitCourse(courses map[string][]Course) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.courses = &courses
	fmt.Println("*** courses size: ", len(*d.courses))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) InitRouteSchedule(routeSchedules []RouteSchedule) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.routeSchedules = &routeSchedules
	fmt.Println("*** routeSchedules size: ", len(*d.routeSchedules))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) UpdateVehicleOccupancies(vehicleOccupancies map[int]VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.vehicleOccupancies = &vehicleOccupancies
	fmt.Println("*** vehicleOccupancies size: ", len(*d.vehicleOccupancies))
	d.lastVehicleOccupanciesUpdate = time.Now()
}

func (d *VehicleOccupanciesContext) GetLastVehicleOccupanciesDataUpdate() time.Time {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return d.lastVehicleOccupanciesUpdate
}

func (d *VehicleOccupanciesContext) GetStopPoints() map[string]StopPoint {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.stopPoints
}

func (d *VehicleOccupanciesContext) GetCourses() map[string][]Course {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.courses
}

func (d *VehicleOccupanciesContext) GetCourseFirstTime(prediction Prediction) (date_time time.Time, e error) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, course := range (*d.courses)[prediction.LineCode] {
		if prediction.Course == course.Course && int(prediction.Date.Weekday()) == course.DayOfWeek {
			return course.FirstTime, nil
		}
	}
	return time.Now(), fmt.Errorf("No corresponding data found")
}

func (d *VehicleOccupanciesContext) GetVehicleJourneyId(predict Prediction, dataTime time.Time) (vj string) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	minDiff := math.Inf(1)
	result := ""
	for _, rs := range *d.routeSchedules {
		if rs.Departure &&
			predict.LineCode == rs.LineCode &&
			predict.Direction == rs.Direction &&
			math.Abs(rs.DateTime.Sub(dataTime).Seconds()) < minDiff {
			minDiff = math.Abs(rs.DateTime.Sub(dataTime).Seconds())
			result = rs.VehicleJourneyId
		}
	}
	return result
}

func (d *VehicleOccupanciesContext) GetRouteSchedule(vjId, stopId string, direction int) (routeSchedule *RouteSchedule) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()
	for _, rs := range *d.routeSchedules {
		if rs.VehicleJourneyId == vjId && rs.StopId == stopId && rs.Direction == direction {
			return &rs
		}
	}
	return nil
}

func (d *VehicleOccupanciesContext) GetRouteSchedules() (routeSchedules []RouteSchedule) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.routeSchedules
}

func (d *VehicleOccupanciesContext) GetVehiclesOccupancies() (vehicleOccupancies map[int]VehicleOccupancy) {
	d.vehicleOccupanciesMutex.RLock()
	defer d.vehicleOccupanciesMutex.RUnlock()

	return *d.vehicleOccupancies
}

func (d *VehicleOccupanciesContext) GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
	vehicleOccupancies []VehicleOccupancy, e error) {
	var occupancies []VehicleOccupancy
	{
		d.vehicleOccupanciesMutex.RLock()
		defer d.vehicleOccupanciesMutex.RUnlock()

		if d.vehicleOccupancies == nil {
			e = fmt.Errorf("No vehicle_occupancies in the data")
			return
		}

		// Implement filter on parameters
		for _, vo := range *d.vehicleOccupancies {
			// Filter on stop_id
			if len(param.StopId) > 0 && param.StopId != vo.StopId {
				continue
			}
			// Filter on vehiclejourney_id
			if len(param.VehicleJourneyId) > 0 && param.VehicleJourneyId != vo.VehicleJourneyId {
				continue
			}
			//Fileter on datetime (default value Now)
			if vo.DateTime.Before(param.Date) {
				continue
			}
			occupancies = append(occupancies, vo)
		}
		return occupancies, nil
	}
}

func NewVehicleOccupancy(rs RouteSchedule, occupancy int) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		Id:               rs.Id,
		LineCode:         rs.LineCode,
		VehicleJourneyId: rs.VehicleJourneyId,
		StopId:           rs.StopId,
		Direction:        rs.Direction,
		DateTime:         rs.DateTime,
		Occupancy:        occupancy,
	}, nil
}
