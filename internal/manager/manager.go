package manager

import (
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/parkings"
	vehicleoccupanciesv2 "github.com/CanalTP/forseti/internal/vehicleoccupancies_v2"
	"github.com/CanalTP/forseti/internal/vehiclepositions"
)

// Data manager for all apis
type DataManager struct {
	freeFloatingsContext       *freefloatings.FreeFloatingsContext
	vehiculeOccupanciesContext vehicleoccupanciesv2.IVehicleOccupancy
	equipmentsContext          *equipments.EquipmentsContext
	departuresContext          *departures.DeparturesContext
	parkingsContext            *parkings.ParkingsContext
	vehiclePositionsContext    vehiclepositions.IConnectors
}

func (d *DataManager) SetEquipmentsContext(equipmentsContext *equipments.EquipmentsContext) {
	d.equipmentsContext = equipmentsContext
}

func (d *DataManager) GetEquipmentsContext() *equipments.EquipmentsContext {
	return d.equipmentsContext
}

func (d *DataManager) SetFreeFloatingsContext(freeFloatingsContext *freefloatings.FreeFloatingsContext) {
	d.freeFloatingsContext = freeFloatingsContext
}

func (d *DataManager) GetFreeFloatingsContext() *freefloatings.FreeFloatingsContext {
	return d.freeFloatingsContext
}

func (d *DataManager) SetDeparturesContext(departuresContext *departures.DeparturesContext) {
	d.departuresContext = departuresContext
}

func (d *DataManager) GetDeparturesContext() *departures.DeparturesContext {
	return d.departuresContext
}

func (d *DataManager) SetParkingsContext(parkingsContext *parkings.ParkingsContext) {
	d.parkingsContext = parkingsContext
}

func (d *DataManager) GetParkingsContext() *parkings.ParkingsContext {
	return d.parkingsContext
}

func (d *DataManager) SetVehicleOccupanciesContext(
	vehiculeOccupanciesContext vehicleoccupanciesv2.IVehicleOccupancy) {
	d.vehiculeOccupanciesContext = vehiculeOccupanciesContext
}

func (d *DataManager) GetVehicleOccupanciesContext() vehicleoccupanciesv2.IVehicleOccupancy {
	return d.vehiculeOccupanciesContext
}

func (d *DataManager) SetVehiclePositionsContext(
	vehiclePositionsContext vehiclepositions.IConnectors) {
	d.vehiclePositionsContext = vehiclePositionsContext
}

func (d *DataManager) GetVehiclePositionsContext() vehiclepositions.IConnectors {
	return d.vehiclePositionsContext
}
