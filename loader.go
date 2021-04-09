package forseti

import (
	"fmt"
	"io"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/text/encoding/charmap"

	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/parkings"
	"github.com/CanalTP/forseti/internal/vehicleoccupancies"
)

func init() {
	prometheus.MustRegister(departures.DepartureLoadingDuration)
	prometheus.MustRegister(departures.DepartureLoadingErrors)
	prometheus.MustRegister(parkings.ParkingsLoadingDuration)
	prometheus.MustRegister(parkings.ParkingsLoadingErrors)
	prometheus.MustRegister(equipments.EquipmentsLoadingDuration)
	prometheus.MustRegister(equipments.EquipmentsLoadingErrors)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingDuration)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingErrors)
	prometheus.MustRegister(vehicleoccupancies.VehicleOccupanciesLoadingDuration)
	prometheus.MustRegister(vehicleoccupancies.VehicleOccupanciesLoadingErrors)
}

func getCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" {
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	}

	return nil, fmt.Errorf("Unknown Charset")
}
