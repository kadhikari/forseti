package vehicleoccupancies

// Declaration of the different structures loaded from Navitia.
// Methods for querying Navitia on the various data to be loaded.

import (
	"net/url"
	"time"
)

// Structure to load the last date of modification static data
type Status struct {
}

// Structure to load vehicle journey Navitia
type NavitiaVehicleJourney struct {
}

// Structure and Consumer to creates Vehicle Journey objects
type VehicleJourney struct {
}

func GetStatusLastLoadAt(uri url.URL, token string, connectionTimeout time.Duration) (string, error) {
	return "", nil
}

func GetVehicleJourney(id string, uri url.URL, token string, connectionTimeout time.Duration) (VehicleJourney,
	error) {
	return VehicleJourney{}, nil
}
