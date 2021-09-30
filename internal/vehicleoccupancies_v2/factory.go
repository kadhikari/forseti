package vehicleoccupanciesv2

// This module is used to declare the interface used by the different vehicle occupancy contexts
// and to create them according to their base type.
// To implement an interface, the contexts must imperatively implement all the methods declared in it.

import (
	"fmt"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
)

type IVehicleOccupancy interface {
	InitContext(filesURI, externalURI url.URL,
		externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ,
		occupancyCleanVO, connectionTimeout time.Duration, location *time.Location, occupancyActive bool)

	RefreshVehicleOccupanciesLoop(predictionURI url.URL, externalToken string,
		navitiaURI url.URL, navitiaToken string, loadExternalRefresh, occupancyCleanVJ, occupancyCleanVO,
		connectionTimeout time.Duration, location *time.Location)

	GetVehicleOccupancies(param *VehicleOccupancyRequestParameter) (
		vehicleOccupancies []VehicleOccupancy, e error)

	ManageVehicleOccupancyStatus(vehicleOccupanciesActive bool)

	GetLastVehicleOccupanciesDataUpdate() time.Time

	LoadOccupancyData() bool

	GetRereshTime() string
}

// Patern factory Vehicle occupancies
func VehicleOccupancyFactory(type_vehicleoccupancy string) (IVehicleOccupancy, error) {
	if type_vehicleoccupancy == string(connectors.Connector_GRFS_RT) {
		return &VehicleOccupanciesGtfsRtContext{}, nil
		/*} else if type_vehicleoccupancy == string(connectors.Connector_ODITI) {
		return &VehicleOccupanciesOditiContext{}, nil*/
	} else {
		return nil, fmt.Errorf("Wrong vehicleoccupancy type passed")
	}
}
