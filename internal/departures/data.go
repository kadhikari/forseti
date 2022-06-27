package departures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

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

// DepartureLineConsumer constructs a departure from a slice of strings
type DepartureLineConsumer struct {
	Data map[string][]Departure
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

func MakeDepartureLineConsumer() *DepartureLineConsumer {
	return &DepartureLineConsumer{make(map[string][]Departure)}
}

func (p *DepartureLineConsumer) Consume(line []string, loc *time.Location) error {

	departure, err := NewDeparture(line, loc)
	if err != nil {
		return err
	}

	p.Data[departure.Stop] = append(p.Data[departure.Stop], departure)
	return nil
}

func (p *DepartureLineConsumer) Terminate() {
	//sort the departures
	for _, v := range p.Data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
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
