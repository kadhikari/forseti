package utils

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var vehicleCapacity = 100

func StringToInt(inputStr string, defaultValue int) int {
	input, err := strconv.Atoi(inputStr)
	if err != nil {
		input = defaultValue
	}
	return input
}

func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func CoordDistance(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
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

// In time part we have '0000-01-01' as date so subtract 1 from month and Day
func AddDateAndTime(date, time time.Time) (dateTime time.Time) {
	return time.AddDate(date.Year(), int(date.Month())-1, date.Day()-1)
}

func CalculateOccupancy(charge int) int {
	if charge == 0 {
		return 0
	}
	occupancy := (charge * 100) / vehicleCapacity
	return occupancy
}

func GetHttpClient(url, token, header string, connectionTimeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * connectionTimeout}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("content-type", "application/x-www-form-urlencoded; param=value")
	req.Header.Set(header, token)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func CheckResponseStatus(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusInternalServerError {
			mes := getMessageError(string(bodyBytes))
			return fmt.Errorf("ERROR %d: %s", resp.StatusCode, mes)

		} else {
			return fmt.Errorf("ERROR %d: no details for this error", resp.StatusCode)
		}
	}
	return nil
}

func Split(r rune) bool {
	return r == '{' || r == '}' || r == ':' || r == ','
}

func getMessageError(bodyString string) string {
	bodySplit := strings.FieldsFunc(bodyString, Split)
	for idx, field := range bodySplit {
		if strings.Contains(field, "message") {
			return strings.Trim(strings.TrimSpace(bodySplit[idx+1]), "\"")
		}
	}
	return "no details for this error"
}
