package equipments

import (
	"fmt"
	"sync"
	"time"
)

type EquipmentsContext struct {
	equipments          *[]EquipmentDetail
	lastEquipmentUpdate time.Time
	equipmentsMutex     sync.RWMutex
}

func (d *EquipmentsContext) GetEquipments() (equipments []EquipmentDetail, e error) {
	var equipmentDetails []EquipmentDetail
	{
		d.equipmentsMutex.RLock()
		defer d.equipmentsMutex.RUnlock()

		if d.equipments == nil {
			e = fmt.Errorf("No equipments in the data")
			return
		}

		equipmentDetails = *d.equipments
	}

	return equipmentDetails, nil
}

func (d *EquipmentsContext) UpdateEquipments(equipments []EquipmentDetail) {
	d.equipmentsMutex.Lock()
	defer d.equipmentsMutex.Unlock()

	d.equipments = &equipments
	d.lastEquipmentUpdate = time.Now()
}

func (d *EquipmentsContext) GetLastEquipmentsDataUpdate() time.Time {
	d.equipmentsMutex.RLock()
	defer d.equipmentsMutex.RUnlock()

	return d.lastEquipmentUpdate
}
