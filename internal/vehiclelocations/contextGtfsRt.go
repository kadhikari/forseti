package vehiclelocations

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies GTFS-RT objects
--------------------------------------------------------------------- */
type VehicleLocationsGtfsRtContext struct {
	globalContext *VehicleLocationsContext
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleLocationsGtfsRtContext) InitContext() {

}

func (d *VehicleLocationsGtfsRtContext) RefreshVehicleLocationsLoop() {}

func (d *VehicleLocationsGtfsRtContext) GetVehicleLocations(param *VehicleLocationRequestParameter) (
	vehicleLocations []VehicleLocation, e error) {
	return d.globalContext.GetVehicleLocations(param)
}
