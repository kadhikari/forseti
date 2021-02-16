package forseti

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
	"strings"
	"math"
)

var spFileName = "mapping_stops.csv"
var courseFileName = "extraction_des_courses.csv"

type FreeFloatingType int
const (
	BikeType FreeFloatingType = iota
	ScooterType
	MotorScooterType
	StationType
	CarType
	OtherType
	UnknownType
)

func (f FreeFloatingType) String() string {
	return [...]string{"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER", "UNKNOWN"}[f]
}

type FreeFloatingRequestParameter struct {
	distance int
	coord Coord
	count int
	types [] FreeFloatingType
}

func ParseFreeFloatingTypeFromParam(value string)  FreeFloatingType {
	switch strings.ToLower(value) {
	case "bike":
		return BikeType
	case "scooter":
		return ScooterType
	case "motorscooter":
		return MotorScooterType
	case "station":
		return StationType
	case "car":
		return CarType
	case "other":
		return OtherType
	default:
		return UnknownType
	}
}

func keepIt(ff FreeFloating, types [] FreeFloatingType) bool {
	if len(types) == 0 {
		return true
	}
	for _, value := range types {
		if strings.EqualFold(ff.Type,  value.String()) {
            return true
        }
	}
	return false
}

type DirectionType int

const (
	DirectionTypeUnknown DirectionType = iota
	DirectionTypeForward
	DirectionTypeBackward
	DirectionTypeBoth
)

func (d DirectionType) String() string {
	return [...]string{"unknown", "forward", "backward", "both"}[d]
}

// MarshalJSON marshals the enum as a quoted json string
func (d DirectionType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(d.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a quoted json string to the enum value
func (d *DirectionType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*d, err = ParseDirectionTypeFromNavitia(j)
	if err != nil {
		return err
	}
	return nil
}

func ParseDirectionType(value string) DirectionType {
	switch value {
	case "ALL": //Aller => forward in french
		return DirectionTypeForward
	case "RET": // Retour => backward in french
		return DirectionTypeBackward
	default:
		return DirectionTypeUnknown

	}
}

func ParseDirectionTypeFromNavitia(value string) (DirectionType, error) {
	switch value {
	case "forward":
		return DirectionTypeForward, nil
	case "backward":
		return DirectionTypeBackward, nil
	case "", "both":
		return DirectionTypeBoth, nil
	case "unknown":
		return DirectionTypeUnknown, nil
	default:
		return DirectionTypeUnknown, fmt.Errorf("impossible to parse %s", value)

	}
}

type LineConsumer interface {
	Consume([]string, *time.Location) error
	Terminate()
}

// Departure represent a departure for a public transport vehicle
type Departure struct {
	Line          string        `json:"line"`
	Stop          string        `json:"stop"`
	Type          string        `json:"type"`
	Direction     string        `json:"direction"`
	DirectionName string        `json:"direction_name"`
	Datetime      time.Time     `json:"datetime"`
	DirectionType DirectionType `json:"direction_type,omitempty"`
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
	var directionType DirectionType
	if len(record) >= 10 {
		directionType = ParseDirectionType(record[9])
	}

	return Departure{
		Stop:          record[0],
		Line:          record[1],
		Type:          record[4],
		Datetime:      dt,
		Direction:     record[6],
		DirectionName: record[2],
		DirectionType: directionType,
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

type CurrentAvailability struct {
	Status    string    `json:"status"`
	Cause     Cause     `json:"cause"`
	Effect    Effect    `json:"effect"`
	Periods   []Period  `json:"periods"`
	UpdatedAt time.Time `json:"updated_at"`
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

func EmbeddedType(s string) (string, error) {
	switch {
	case s == "ASCENSEUR":
		return "elevator", nil
	case s == "ESCALIER":
		return "escalator", nil
	default:
		return "", fmt.Errorf("Unsupported EmbeddedType %s", s)
	}
}

// NewEquipmentDetail creates a new EquipmentDetail object from the object EquipementSource
func NewEquipmentDetail(es EquipementSource, updatedAt time.Time, location *time.Location) (*EquipmentDetail, error) {
	start, err := time.ParseInLocation("2006-01-02", es.Start, location)
	if err != nil {
		return nil, err
	}

	end, err := time.ParseInLocation("2006-01-02", es.End, location)
	if err != nil {
		return nil, err
	}

	hour, err := time.ParseInLocation("15:04:05", es.Hour, location)
	if err != nil {
		return nil, err
	}

	// Add time part to end date
	end = hour.AddDate(end.Year(), int(end.Month())-1, end.Day()-1)

	etype, err := EmbeddedType(es.Type)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	return &EquipmentDetail{
		ID:           es.ID,
		Name:         es.Name,
		EmbeddedType: etype,
		CurrentAvailability: CurrentAvailability{
			Status:    GetEquipmentStatus(start, end, now),
			Cause:     Cause{Label: es.Cause},
			Effect:    Effect{Label: es.Effect},
			Periods:   []Period{Period{Begin: start, End: end}},
			UpdatedAt: updatedAt},
	}, nil
}

//FreeFloating defines how the object is represented in a response
type Coord struct {
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
}

type FreeFloating struct {
	PublicId string `json:"public_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	Id string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Coord Coord `json:"coord,omitempty"`
	Propulsion string `json:"propulsion,omitempty"`
	Battery int `json:"battery,omitempty"`
	Deeplink string `json:"deeplink,omitempty"`
	Attributes [] string `json:"attributes,omitempty"`
	Distance float64 `json:"distance,omitempty"`
}

type ByDistance []FreeFloating

func (ff ByDistance) Len() int {return len(ff)}
func (ff ByDistance) Less(i, j int) bool {return ff[i].Distance < ff[j].Distance}
func (ff ByDistance) Swap(i, j int)      { ff[i], ff[j] = ff[j], ff[i] }

// NewFreeFloating creates a new FreeFloating object from the object Vehicle
func NewFreeFloating(ve Vehicle) (*FreeFloating) {
	return &FreeFloating{
		PublicId:		ve.PublicId,
		ProviderName: 	ve.Provider.Name,
		Id:				ve.Id,
		Type:			ve.Type,
		Coord:			Coord{Lat: ve.Latitude, Lon: ve.Longitude},
		Propulsion:		ve.Propulsion,
		Battery:		ve.Battery,
		Deeplink:		ve.Deeplink,
		Attributes:		ve.Attributes,
	}
}

// Structure and Consumer to creates StopPoint objects based on a line read from a CSV
type StopPoint struct {
	Id 			string
	Name 		string
}

func NewStopPoint(record []string) (*StopPoint, error) {
	if len(record) < 3 {
		return nil, fmt.Errorf("Missing field in StopPoint record")
	}

	return &StopPoint{
		Id:				fmt.Sprintf("%s:%s", "stop_point", record[2]),
		Name: 			record[1],
	}, nil
}

type StopPointLineConsumer struct {
	stopPoints map[string]StopPoint
}

func makeStopPointLineConsumer() *StopPointLineConsumer {
	return &StopPointLineConsumer{
		stopPoints: make(map[string]StopPoint),
	}
}

func (c *StopPointLineConsumer) Consume(line []string, loc *time.Location) error {
	stopPoint, err := NewStopPoint(line)
	if err != nil {
		return err
	}

	c.stopPoints[stopPoint.Id] = *stopPoint
	return nil
}
func (c *StopPointLineConsumer) Terminate() {}

// Structure and Consumer to creates Course objects based on a line read from a CSV
type Course struct {
	LineCode string
	Course string
	Dow int
	FirstDate time.Time
	FirstTime time.Time
}

func NewCourse(record []string,location *time.Location) (*Course, error) {
	if len(record) < 9 {
		return nil, fmt.Errorf("Missing field in Course record")
	}
	dow, err := strconv.Atoi(record[2])
	if err != nil {
		return nil, err
	}
	firstDate, err := time.ParseInLocation("2006-01-02", record[6], location)
	if err != nil {
		return nil, err
	}
	firstTime, err := time.ParseInLocation("15:04:05", record[3], location)
	if err != nil {
		return nil, err
	}
	return &Course{
		LineCode:	record[0],
		Course:		record[1],
		Dow: 		dow,
		FirstDate: 	firstDate,
		FirstTime:	firstTime,
	}, nil
}

type CourseLineConsumer struct {
	courses map[string][]Course
}

func makeCourseLineConsumer() *CourseLineConsumer {
	return &CourseLineConsumer{make(map[string][]Course)}
}

func (c *CourseLineConsumer) Consume(line []string, loc *time.Location) error {
	course, err := NewCourse(line, loc)
	if err != nil {
		return err
	}

	c.courses[course.LineCode] = append(c.courses[course.LineCode], *course)
	return nil
}
func (p *CourseLineConsumer) Terminate() {}

type Prediction struct {
	LineCode    string
	Sens		int
	Date      	string
	Course    	string
	StopId 		string
	Charge    	int
	CreatedAt 	time.Time
}

func NewPrediction(p PredictionNode) (*Prediction) {
	return &Prediction{
		LineCode: 	p.Line,
		Sens: 		p.Sens,
		Date:		p.Date,
		Course:		p.Course,
		StopId:		p.StopId,
		Charge:		int(p.Charge),
	}
}

// Structures and functions to read files for vehicle_occupancies are here
type VehicleOccupancy struct {
	LineCode string `json:"line_code,omitempty"`
	VehicleJourneyId string `json:"vj_id,omitempty"`
	StoppointId string `json:"stop_id,omitempty"`
	Sens int `json:"sens,omitempty"`
	DateTime string `json:"date_time,omitempty"`
	Charge int `json:"charge,omitempty"`
}

func NewVehicleOccupancy(stop_id, vj_id, date_time string, sens int) (*VehicleOccupancy, error) {
	return &VehicleOccupancy{
		LineCode: "toto",
		VehicleJourneyId: vj_id,
		StoppointId: stop_id,
		Sens: sens,
		DateTime: date_time,
		Charge: 0,
	}, nil
}

// Data manager for all apis
type DataManager struct {
	departures          *map[string][]Departure
	lastDepartureUpdate time.Time
	departuresMutex     sync.RWMutex

	parkings          *map[string]Parking
	lastParkingUpdate time.Time
	parkingsMutex     sync.RWMutex

	equipments          *[]EquipmentDetail
	lastEquipmentUpdate time.Time
	equipmentsMutex     sync.RWMutex

	freeFloatings          *[]FreeFloating
	lastFreeFloatingUpdate time.Time
	freeFloatingsMutex     sync.RWMutex

	stopPoints						*map[string]StopPoint
	courses							*[]Course
	vehicleOccupancies 				*[]VehicleOccupancy
	lastVehicleOccupanciesUpdate 	time.Time
	vehicleOccupanciesMutex     	sync.RWMutex
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

func (d *DataManager) GetDeparturesByStops(stopsID []string) ([]Departure, error) {
	return d.GetDeparturesByStopsAndDirectionType(stopsID, DirectionTypeBoth)
}
func (d *DataManager) GetDeparturesByStopsAndDirectionType(
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

func (d *DataManager) UpdateEquipments(equipments []EquipmentDetail) {
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
	var equipmentDetails []EquipmentDetail
	{
		d.equipmentsMutex.RLock()
		defer d.equipmentsMutex.RUnlock()

		if d.equipments == nil {
			e = fmt.Errorf("No equipments in the data")
			return
		}

		equipmentDetails = *d.equipments
	}

	return equipmentDetails, nil
}

// GetEquipmentStatus returns availability of equipment
func GetEquipmentStatus(start time.Time, end time.Time, now time.Time) string {
	if now.Before(start) || now.After(end) {
		return "available"
	} else {
		return "unavailable"
	}

}

func (d *DataManager) UpdateFreeFloating(freeFloatings []FreeFloating) {
	d.freeFloatingsMutex.Lock()
	defer d.freeFloatingsMutex.Unlock()

	d.freeFloatings = &freeFloatings
	d.lastFreeFloatingUpdate = time.Now()
}

func (d *DataManager) GetLastFreeFloatingsDataUpdate() time.Time {
	d.freeFloatingsMutex.RLock()
	defer d.freeFloatingsMutex.RUnlock()

	return d.lastFreeFloatingUpdate
}

func (d *DataManager) GetFreeFloatings(param * FreeFloatingRequestParameter) (freeFloatings []FreeFloating, e error) {
	resp := make([]FreeFloating, 0)
	{
		d.freeFloatingsMutex.RLock()
		defer d.freeFloatingsMutex.RUnlock()

		if d.freeFloatings == nil {
			e = fmt.Errorf("No free-floatings in the data")
			return
		}

		// Implementation of filters: distance, type[]
		for _, ff := range *d.freeFloatings {
			// Filter on type[]
			keep := keepIt(ff, param.types)

			if keep == false {
				continue
			}

			// Calculate distance from coord in the request
			distance := coordDistance(param.coord, ff.Coord)
			ff.Distance = math.Round(distance)
			if int(distance) > param.distance {
				continue
			}

			// Keep the wanted object
			if keep == true {
				resp = append(resp, ff)
			}

			if len(resp) == param.count {
				break
			}
		}
		sort.Sort(ByDistance(resp))
	}
	return resp, nil
}

func (d *DataManager) InitOccupancies(vehicleOccupancies []VehicleOccupancy) {
	d.vehicleOccupanciesMutex.Lock()
	defer d.vehicleOccupanciesMutex.Unlock()

	d.vehicleOccupancies = &vehicleOccupancies
	d.lastVehicleOccupanciesUpdate = time.Now()
}
