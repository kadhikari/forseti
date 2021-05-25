package vehiclelocations

import (
	"fmt"
)

// This module is used to declare the interface used by the different vehicle location contexts
// and to create them according to their base type.
// To implement an interface, the contexts must imperatively implement all the methods declared in it.

type IVehicleLocation interface {
	InitContext()

	//RefreshVehicleLocationsLoop()

	GetVehicleLocations(param *VehicleLocationRequestParameter) (
		vehicleLocations []VehicleLocation, e error)

	/*ManageVehicleLocationStatus()

	GetLastVehicleLocationsDataUpdate()

	LoadLocationData()*/
}

// Patern factory Vehicle occupancies
func VehicleLocationFactory(type_vehiclelocation string) (IVehicleLocation, error) {
	if type_vehiclelocation == "gtfs" {
		return &VehicleLocationsGtfsRtContext{}, nil
	} else {
		return nil, fmt.Errorf("Wrong vehiclelocation type passed")
	}
}
