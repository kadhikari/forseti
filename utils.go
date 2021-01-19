package forseti
import (
	"strconv"
	"math"
	"strings"
)

func isTypeValid(strType string) bool {
	list := []string {"BIKE", "SCOOTER", "MOTORSCOOTER", "STATION", "CAR", "OTHER"}
	for _, b := range list {
		if strings.EqualFold(strType, b) {
			return true
		}
    }
    return false
}

func stringToInt(inputStr string, defaultValue int) int {
	input, err := strconv.Atoi(inputStr)
	if err != nil {
		input = defaultValue
	}
	return input
}

func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func coordDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
  	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}



