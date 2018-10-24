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
