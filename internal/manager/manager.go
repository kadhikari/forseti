package manager

import (
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/parkings"
	"github.com/CanalTP/forseti/internal/vehicleoccupancies"
)

// Data manager for all apis
type DataManager struct {
	freeFloatingsContext       *freefloatings.FreeFloatingsContext
	vehiculeOccupanciesContext vehicleoccupancies.IVehicleOccupancy
	equipmentsContext          *equipments.EquipmentsContext
	departuresContext          *departures.DeparturesContext
	parkingsContext            *parkings.ParkingsContext
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

func (d *DataManager) SetVehiculeOccupanciesContext(
	vehiculeOccupanciesContext vehicleoccupancies.IVehicleOccupancy) {
	d.vehiculeOccupanciesContext = vehiculeOccupanciesContext
}

func (d *DataManager) GetVehiculeOccupanciesContext() vehicleoccupancies.IVehicleOccupancy {
	return d.vehiculeOccupanciesContext
}
