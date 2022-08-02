package departures

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type DepartureType int

const (
	DepartureTypeTheoretical DepartureType = iota
	DepartureTypeEstimated
)

func (d DepartureType) String() string {
	var result string
	switch d {
	case DepartureTypeTheoretical:
		result = "T"
	default:
		result = "E"
	}
	return result
}
