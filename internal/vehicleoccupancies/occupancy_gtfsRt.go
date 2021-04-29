package vehicleoccupancies

type VehicleOccupancyStatus int

const (
	// The vehicle is considered empty by most measures, and has few or no
	// passengers onboard, but is still accepting passengers.
	VehicleOccupancy_EMPTY VehicleOccupancyStatus = 0
	// The vehicle has a relatively large percentage of seats available.
	// What percentage of free seats out of the total seats available is to be
	// considered large enough to fall into this category is determined at the
	// discretion of the producer.
	VehicleOccupancy_MANY_SEATS_AVAILABLE VehicleOccupancyStatus = 1
	// The vehicle has a relatively small percentage of seats available.
	// What percentage of free seats out of the total seats available is to be
	// considered small enough to fall into this category is determined at the
	// discretion of the feed producer.
	VehicleOccupancy_FEW_SEATS_AVAILABLE VehicleOccupancyStatus = 2
	// The vehicle can currently accommodate only standing passengers.
	VehicleOccupancy_STANDING_ROOM_ONLY VehicleOccupancyStatus = 3
	// The vehicle can currently accommodate only standing passengers
	// and has limited space for them.
	VehicleOccupancy_CRUSHED_STANDING_ROOM_ONLY VehicleOccupancyStatus = 4
	// The vehicle is considered full by most measures, but may still be
	// allowing passengers to board.
	VehicleOccupancy_FULL VehicleOccupancyStatus = 5
	// The vehicle is not accepting additional passengers.
	VehicleOccupancy_NOT_ACCEPTING_PASSENGERS VehicleOccupancyStatus = 6
)

var OditiMatchMatrixGtfsRT = [4][2]int{
	// EMPTY for value equal 0
	{0, 25},  // MANY_SEATS_AVAILABLE for value between > 0 and 25
	{25, 50}, // FEW_SEATS_AVAILABLE for value between > 25 and 50
	{50, 75}, // STANDING_ROOM_ONLY for value between > 50 and 50
	{75, 99}, // CRUSHED_STANDING_ROOM_ONLY for value between > 75 and 50
	// FULL for value equal or better than 100
}

func GetOccupancyStatusForOditi(Oditi_charge int) VehicleOccupancyStatus {
	var s int = 1
	if Oditi_charge == 0 {
		return VehicleOccupancy_EMPTY
	}
	if Oditi_charge >= 100 {
		return VehicleOccupancy_FULL
	}

	for idx, p := range OditiMatchMatrixGtfsRT {
		if InBetween(Oditi_charge, p[0], p[1]) {
			s = s + idx
			break
		}
	}
	return VehicleOccupancyStatus(s)
}

func InBetween(charge, min, max int) bool {
	if (charge > min) && (charge <= max) {
		return true
	} else {
		return false
	}
}
