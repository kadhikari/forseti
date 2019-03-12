package sytralrt

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

type LineConsumer interface {
	Consume([]string, *time.Location) error
	Terminate()
}

// Departure represent a departure for a public transport vehicle
type Departure struct {
	Line          string    `json:"line"`
	Stop          string    `json:"stop"`
	Type          string    `json:"type"`
	Direction     string    `json:"direction"`
	DirectionName string    `json:"direction_name"`
	Datetime      time.Time `json:"datetime"`
	//VJ            string
	//Route         string
}

func NewDeparture(record []string, location *time.Location) (Departure, error) {
	if len(record) < 7 {
		return Departure{}, fmt.Errorf("Missing field in record")
	}
	dt, err := time.ParseInLocation("2006-01-02 15:04:05", record[5], location)
	if err != nil {
		return Departure{}, err
	}

	return Departure{
		Stop:          record[0],
		Line:          record[1],
		Type:          record[4],
		Datetime:      dt,
		Direction:     record[6],
		DirectionName: record[2],
	}, nil
}

// DepartureLineConsumer constructs a departure from a slice of strings
type DepartureLineConsumer struct {
	data map[string][]Departure
}

func makeDepartureLineConsumer() *DepartureLineConsumer {
	return &DepartureLineConsumer{make(map[string][]Departure)}
}

func (p *DepartureLineConsumer) Consume(line []string, loc *time.Location) error {

	departure, err := NewDeparture(line, loc)
	if err != nil {
		return err
	}

	p.data[departure.Stop] = append(p.data[departure.Stop], departure)
	return nil
}

func (p *DepartureLineConsumer) Terminate() {
	//sort the departures
	for _, v := range p.data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
}

// Parking defines details and spaces available for P+R parkings
type Parking struct {
	ID                        string    `json:"Id"`
	Label                     string    `json:"label"`
	UpdatedTime               time.Time `json:"updated_time"`
	AvailableStandardSpaces   int       `json:"available_space"`
	AvailableAccessibleSpaces int       `json:"available_accessible_space"`
	TotalStandardSpaces       int       `json:"available_normal_space"`
	TotalAccessibleSpaces     int       `json:"total_space"`
}

type ByParkingId []Parking

func (p ByParkingId) Len() int           { return len(p) }
func (p ByParkingId) Less(i, j int) bool { return p[i].ID < p[j].ID }
func (p ByParkingId) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// NewParking creates a new Parking object based on a line read from a CSV
func NewParking(record []string, location *time.Location) (*Parking, error) {
	if len(record) < 8 {
		return nil, fmt.Errorf("Missing field in Parking record")
	}

	updatedTime, err := time.ParseInLocation("2006-01-02 15:04:05", record[2], location)
	if err != nil {
		return nil, err
	}
	availableStd, err := strconv.Atoi(record[4])
	if err != nil {
		return nil, err
	}
	totalStd, err := strconv.Atoi(record[5])
	if err != nil {
		return nil, err
	}
	availableAcc, err := strconv.Atoi(record[6])
	if err != nil {
		return nil, err
	}
	totalAcc, err := strconv.Atoi(record[7])
	if err != nil {
		return nil, err
	}

	return &Parking{
		ID:                        record[0],    // COD_PAR_REL
		Label:                     record[1],    // LIB_PAR_REL
		UpdatedTime:               updatedTime,  // DATEHEURE_COMPTAGE
		AvailableStandardSpaces:   availableStd, // NB_TOT_PLACE_DISPO
		AvailableAccessibleSpaces: availableAcc, // NB_TOT_PLACE_PMR_DISPO
		TotalStandardSpaces:       totalStd,     // CAP_VEH_NOR
		TotalAccessibleSpaces:     totalAcc,     // CAP_VEH_PMR
	}, nil
}

// ParkingLineConsumer constructs a parking from a slice of strings
type ParkingLineConsumer struct {
	parkings map[string]Parking
}

func makeParkingLineConsumer() *ParkingLineConsumer {
	return &ParkingLineConsumer{
		parkings: make(map[string]Parking),
	}
}

func (p *ParkingLineConsumer) Consume(line []string, loc *time.Location) error {
	parking, err := NewParking(line, loc)
	if err != nil {
		return err
	}

	p.parkings[parking.ID] = *parking
	return nil
}

func (p *ParkingLineConsumer) Terminate() {}

// EquipmentDetail defines how a equipment object is represented in a response
type EquipmentDetail struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	EmbeddedType        string              `json:"embedded_type"`
	CurrentAvailability CurrentAvailability `json:"current_availaibity"`
}

type ByEquipmentId []EquipmentDetail

func (e ByEquipmentId) Len() int           { return len(e) }
func (e ByEquipmentId) Less(i, j int) bool { return e[i].ID < e[j].ID }
func (e ByEquipmentId) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

type CurrentAvailability struct {
	Status  string `json:"status"`
	Cause   Cause  `json:"cause"`
	Effect  Effect `json:"effect"`
	Periods Period `json:"periods"`
}

type Cause struct {
	Label string `json:"label"`
}

type Effect struct {
	Label string `json:"label"`
}

type Period struct {
	Begin time.Time `json:"begin"`
	End   time.Time `json:"end"`
}

func get_type(s string) (string, error) {
	switch {
	case s == "ASCENSEUR":
		return "elevator", nil
	case s == "ESCALIER":
		return "escalator", nil
	default:
		return "", fmt.Errorf("Unsupported EmbeddedType %s", s)
	}
}

// NewEquipmentDetail creates a new Equipment object based on a line read from a CSV
func NewEquipmentDetail(record []string, location *time.Location) (*EquipmentDetail, error) {
	start, err := time.ParseInLocation("2006-01-02", record[5], location)
	if err != nil {
		return nil, err
	}

	end, err := time.ParseInLocation("2006-01-02", record[6], location)
	if err != nil {
		return nil, err
	}

	hour, err := time.ParseInLocation("15:04:05", record[7], location)
	if err != nil {
		return nil, err
	}

	basehour, err := time.ParseInLocation("15:04:05", "00:00:00", location)
	if err != nil {
		return nil, err
	}

	etype, err := get_type(record[2])
	if err != nil {
		return nil, err
	}

	now := time.Now()
	end = end.Add(hour.Sub(basehour))

	return &EquipmentDetail{
		ID:           record[0],
		Name:         record[1],
		EmbeddedType: etype,
		CurrentAvailability: CurrentAvailability{
			Status:  GetAvailabilityStatus(start, end, now),
			Cause:   Cause{Label: record[3]},
			Effect:  Effect{Label: record[4]},
			Periods: Period{Begin: start, End: end}},
	}, nil
}

// EquipmentLineConsumer constructs a parking from a slice of strings
type EquipmentLineConsumer struct {
	equipments map[string]EquipmentDetail
}

func makeEquipmentLineConsumer() *EquipmentLineConsumer {
	return &EquipmentLineConsumer{
		equipments: make(map[string]EquipmentDetail),
	}
}

func (e *EquipmentLineConsumer) Consume(line []string, loc *time.Location) error {
	equipment, err := NewEquipmentDetail(line, loc)
	if err != nil {
		return err
	}

	e.equipments[equipment.ID] = *equipment
	return nil
}

func (p *EquipmentLineConsumer) Terminate() {}

type DataManager struct {
	departures          *map[string][]Departure
	lastDepartureUpdate time.Time
	departuresMutex     sync.RWMutex

	parkings          *map[string]Parking
	lastParkingUpdate time.Time
	parkingsMutex     sync.RWMutex

	equipments          *map[string]EquipmentDetail
	lastEquipmentUpdate time.Time
	equipmentsMutex     sync.RWMutex
}

func (d *DataManager) UpdateDepartures(departures map[string][]Departure) {
	d.departuresMutex.Lock()
	defer d.departuresMutex.Unlock()

	d.departures = &departures
	d.lastDepartureUpdate = time.Now()
}

func (d *DataManager) GetLastDepartureDataUpdate() time.Time {
	d.departuresMutex.RLock()
	defer d.departuresMutex.RUnlock()

	return d.lastDepartureUpdate
}

func (d *DataManager) GetDeparturesByStop(stopID string) ([]Departure, error) {

	var departures []Departure
	{
		d.departuresMutex.RLock()
		defer d.departuresMutex.RUnlock()

		if d.departures == nil {
			return []Departure{}, fmt.Errorf("no departures")
		}

		departures = (*d.departures)[stopID]
	}

	if departures == nil {
		//there is no departures for this stop, we return an empty slice
		return []Departure{}, nil
	}
	return departures, nil
}

func (d *DataManager) UpdateParkings(parkings map[string]Parking) {
	d.parkingsMutex.Lock()
	defer d.parkingsMutex.Unlock()

	d.parkings = &parkings
	d.lastParkingUpdate = time.Now()
}

func (d *DataManager) GetLastParkingsDataUpdate() time.Time {
	d.parkingsMutex.RLock()
	defer d.parkingsMutex.RUnlock()

	return d.lastParkingUpdate
}

func (d *DataManager) GetParkingsByIds(ids []string) (parkings []Parking, errors []error) {
	for _, id := range ids {
		if p, err := d.GetParkingById(id); err == nil {
			parkings = append(parkings, p)
		} else {
			errors = append(errors, err)
		}
	}
	return
}

func (d *DataManager) GetParkings() (parkings []Parking, e error) {
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

func (d *DataManager) GetParkingById(id string) (p Parking, e error) {
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

func (d *DataManager) UpdateEquipments(equipments map[string]EquipmentDetail) {
	d.equipmentsMutex.Lock()
	defer d.equipmentsMutex.Unlock()

	d.equipments = &equipments
	d.lastEquipmentUpdate = time.Now()
}

func (d *DataManager) GetLastEquipmentsDataUpdate() time.Time {
	d.equipmentsMutex.RLock()
	defer d.equipmentsMutex.RUnlock()

	return d.lastEquipmentUpdate
}

func (d *DataManager) GetEquipments() (equipments []EquipmentDetail, e error) {
	var mapequipments map[string]EquipmentDetail
	{
		d.equipmentsMutex.RLock()
		defer d.equipmentsMutex.RUnlock()

		if d.equipments == nil {
			e = fmt.Errorf("No equipments in the data")
			return
		}

		mapequipments = *d.equipments
	}

	// Convert Map of equipments to Slice !
	equipments = make([]EquipmentDetail, 0, len(mapequipments))
	for _, ed := range mapequipments {
		equipments = append(equipments, ed)
	}

	return equipments, nil
}

// GetAvailabilityStatus returns availability of equipment
func GetAvailabilityStatus(start time.Time, end time.Time, now time.Time) string {
	if now.Before(start) && now.After(end) {
		return "available"
	} else {
		return "unavailable"
	}

}
