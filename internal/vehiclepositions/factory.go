package vehiclepositions

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
)

// This module is used to declare the interface used by the different vehicle location contexts
// and to create them according to their base type.
// To implement an interface, the contexts must imperatively implement all the methods declared in it.

type IConnectors interface {
	InitContext(
		filesUrl url.URL,
		filesRefreshDuration time.Duration,
		serviceURI url.URL,
		serviceToken string,
		serviceRefreshDuration time.Duration,
		navitiaUrl url.URL,
		navitiaToken string,
		navitiaCoverageName string,
		connectionTimeout time.Duration,
		positionCleanVP time.Duration,
		location *time.Location,
		reloadActive bool,
	)

	RefreshVehiclePositionsLoop()

	GetVehiclePositions(param *VehiclePositionRequestParameter) (
		vehiclePositions []VehiclePosition, e error)

	GetLastVehiclePositionsDataUpdate() time.Time

	GetLastStatusUpdate() time.Time

	ManageVehiclePositionsStatus(activate bool)

	LoadPositionsData() bool

	GetRereshTime() string

	GetConnectorType() connectors.ConnectorType

	GetStatus() string

	SetStatus(status string)
}

// Patern factory
func ConnectorFactory(type_connector string) (IConnectors, error) {
	if type_connector == string(connectors.Connector_GRFS_RT) {
		return &GtfsRtContext{}, nil
	} else if type_connector == string(connectors.Connector_RENNES) {
		return &RennesContext{}, nil
	} else {
		return nil, fmt.Errorf("Wrong connector type passed")
	}
}
