package main

import (
	"fmt"
	"time"
)

// Departure represent a departure for a public transport vehicle
type Departure struct {
	Line          string
	Stop          string
	Type          string
	Direction     string
	DirectionName string
	Datetime      time.Time
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

type DataManager struct {
	departures *map[string][]Departure
}

func (d *DataManager) UpdateDepartures(departures map[string][]Departure) {
	d.departures = &departures
}

func (d *DataManager) GetDeparturesByStop(stopID string) ([]Departure, error) {
	if d.departures == nil {
		return []Departure{}, fmt.Errorf("no departures")
	}
	departures := (*d.departures)[stopID]
	if departures == nil {
		//there is no departures for this stop, we return an empty slice
		return []Departure{}, nil
	}
	return departures, nil
}
