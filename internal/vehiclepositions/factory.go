package vehiclepositions

import (
	"fmt"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
)

// This module is used to declare the interface used by the different vehicle location contexts
// and to create them according to their base type.
// To implement an interface, the contexts must imperatively implement all the methods declared in it.

type IConnectors interface {
	InitContext(ilesURI, externalURI url.URL, externalToken string, loadExternalRefresh, positionCleanVO,
		connectionTimeout time.Duration, location *time.Location, reloadActive bool)

	RefreshVehiclePositionsLoop()

	GetVehiclePositions(param *VehiclePositionRequestParameter) (
		vehiclePositions []VehiclePosition, e error)

	GetLastVehiclePositionsDataUpdate() time.Time

	LoadPositionsData() bool

	GetRereshTime() string
}

// Patern factory
func ConnectorFactory(type_connector string) (IConnectors, error) {
	if type_connector == string(connectors.Connector_GRFS_RT) {
		return &GtfsRtContext{}, nil
	} else {
		return nil, fmt.Errorf("Wrong connector type passed")
	}
}
