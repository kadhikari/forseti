package forseti
import (
	"strconv"
	"math"
	"strings"
	"time"
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

func coordDistance(from, to Coord) float64 {
	// convert to radians
  	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = from.Lat * math.Pi / 180
	lo1 = from.Lon * math.Pi / 180
	la2 = to.Lat * math.Pi / 180
	lo2 = to.Lon * math.Pi / 180

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}

func addDateAndTime(date, time time.Time) (dateTime time.Time) {
	return time.AddDate(date.Year(), int(date.Month())-1, date.Day()-1)
}

func intersects(date1, date2 time.Time, minute time.Duration) bool{
	d := (60 * time.Second)
	start1 := date1.Truncate(d)
	end1 := start1.Add(time.Minute * minute)
	start2 := date2.Truncate(d)
	end2 := start2.Add(time.Minute * minute)
	if  start1.Before(end2) && end1.After(start2) {
		return true
	}
	return false
}

func calculateOccupancy(charge int) int {
	if charge == 0 {return 0}
	occupancy := (charge * 100) / vehicleCapacity
	return occupancy
}
