package departures

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type DeparturesContext struct {
	departures          *map[string][]Departure
	lastDepartureUpdate time.Time
	lastStatusUpdate    time.Time
	departuresMutex     sync.RWMutex
	packageName         string
	filesRefreshTime    time.Duration
	wsRefreshTime       time.Duration
	connectorType       string
	status              string
	message             string
}

func (d *DeparturesContext) UpdateDepartures(departures map[string][]Departure) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.departures = &departures
	d.lastDepartureUpdate = time.Now()
}

func (d *DeparturesContext) DropDepartures() {
	emptyMap := make(map[string][]Departure)
	d.UpdateDepartures(emptyMap)
}

func (d *DeparturesContext) GetLastDepartureDataUpdate() time.Time {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.lastDepartureUpdate
}

func (d *DeparturesContext) GetLastStatusUpdate() time.Time {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.lastStatusUpdate
}

func (d *DeparturesContext) SetLastStatusUpdate(t time.Time) {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	d.lastStatusUpdate = t
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

func (d *DeparturesContext) GetPackageName() string {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	return d.packageName
}

func (d *DeparturesContext) SetPackageName(pathPackage string) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	paths := strings.Split(pathPackage, "/")
	size := len(paths)
	d.packageName = paths[size-1]
}

func (d *DeparturesContext) GetFilesRefeshTime() time.Duration {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.filesRefreshTime
}

func (d *DeparturesContext) SetFilesRefeshTime(filesRefreshTime time.Duration) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.filesRefreshTime = filesRefreshTime
}

func (d *DeparturesContext) GetWsRefeshTime() time.Duration {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.wsRefreshTime
}

func (d *DeparturesContext) SetWsRefeshTime(wsRefreshTime time.Duration) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.wsRefreshTime = wsRefreshTime
}

func (d *DeparturesContext) GetStatus() string {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	return d.status
}

func (d *DeparturesContext) SetStatus(status string) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.status = status
}

func (d *DeparturesContext) GetMessage() string {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()
	return d.message
}

func (d *DeparturesContext) SetMessage(message string) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()
	d.message = message
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

func (d *DeparturesContext) GetConnectorType() string {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()
	return d.connectorType
}

func (d *DeparturesContext) SetConnectorType(connectorType string) {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()
	d.connectorType = connectorType
}
