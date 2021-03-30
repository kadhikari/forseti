package departures

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type DeparturesContext struct {
	departures          *map[string][]Departure
	lastDepartureUpdate time.Time
	departuresMutex     sync.RWMutex
}

func (d *DeparturesContext) UpdateDepartures(departures map[string][]Departure) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.departures = &departures
	d.lastDepartureUpdate = time.Now()
}

func (d *DeparturesContext) GetLastDepartureDataUpdate() time.Time {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.lastDepartureUpdate
}

func (d *DeparturesContext) GetDeparturesByStops(stopsID []string) ([]Departure, error) {
	return d.GetDeparturesByStopsAndDirectionType(stopsID, DirectionTypeBoth)
}
func (d *DeparturesContext) GetDeparturesByStopsAndDirectionType(
	stopsID []string,
	directionType DirectionType) ([]Departure, error) {

	var departures []Departure
	{
		d.departuresMutex.RLock()
		defer d.departuresMutex.RUnlock()

		if d.departures == nil {
			return []Departure{}, fmt.Errorf("no departures")
		}
		for _, stopID := range stopsID {
			departures = append(departures, (*d.departures)[stopID]...)
		}
	}

	if departures == nil {
		//there is no departures for this stop, we return an empty slice
		return []Departure{}, nil
	}
	departures = filterDeparturesByDirectionType(departures, directionType)
	sort.Slice(departures, func(i, j int) bool {
		return departures[i].Datetime.Before(departures[j].Datetime)
	})
	return departures, nil
}

func keepDirection(departureDirectionType, wantedDirectionType DirectionType) bool {
	return (wantedDirectionType == departureDirectionType ||
		departureDirectionType == DirectionTypeUnknown ||
		wantedDirectionType == DirectionTypeBoth)
}

func filterDeparturesByDirectionType(departures []Departure, directionType DirectionType) []Departure {
	n := 0
	for _, d := range departures {
		if keepDirection(d.DirectionType, directionType) {
			departures[n] = d
			n++
		}
	}
	return departures[:n]
}
