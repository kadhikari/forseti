package vehicleoccupancies

// Declaration of the different structures loaded from Navitia.
// Methods for querying Navitia on the various data to be loaded.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/utils"
)

const (
	URL_GET_LAST_LOAD              = "%s/status?"
	URL_GET_VEHICLE_JOURNEY        = "%s/vehicle_journeys?filter=vehicle_journey.has_code(%s)&since=%s&until=%s&depth=2&"
	URL_GET_VEHICLES_JOURNEYS_LINE = "%s/lines/%s/vehicle_journeys?since=%s&until=%s&depth=2&"
	VALUE_FROM_TYPE                = "source"
	URL_GET_ROUTES                 = "%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s"
)

// Structure to load the last date of modification static data
type Status struct {
	Status struct {
		PublicationDate string `json:"publication_date"`
	} `json:"status"`
}

// Structure to load vehicle journey Navitia
type NavitiaVehicleJourney struct {
	VehicleJourneys []struct {
		Codes []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"codes"`
		ID             string `json:"id"`
		JourneyPattern struct {
			Route struct {
				DirectionType string `json:"direction_type"`
			} `json:"route"`
		} `json:"journey_pattern"`
		StopTimes []struct {
			DepartureTime string `json:"departure_time"`
			StopPoint     struct {
				Codes []struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"codes"`
				ID string `json:"id"`
			} `json:"stop_point"`
		} `json:"stop_times"`
	} `json:"vehicle_journeys"`
}

// Structure and Consumer to creates Vehicle Journey objects
type VehicleJourney struct {
	VehicleID   string // vehicle journey id Navitia
	CodesSource string // vehicle source code
	Line        string
	Direction   string
	StopPoints  *[]StopPointVj
	CreateDate  time.Time
}

func NewVehicleJourney(vehicleId, codesSource, line, direction string, stopPoints []StopPointVj,
	date time.Time) *VehicleJourney {
	return &VehicleJourney{
		VehicleID:   vehicleId,
		CodesSource: codesSource,
		Line:        line,
		Direction:   direction,
		StopPoints:  &stopPoints,
		CreateDate:  date,
	}
}

// Structure and Consumer to creates Stop point from Vehicle Journey Navitia objects
type StopPointVj struct {
	Id            string // Stoppoint uri from navitia
	GtfsStopCode  string // StopPoint code gtfs-rt
	DepartureTime time.Time
}

func NewStopPointVj(id string, code string, hourDeparture string) StopPointVj {
	t, _ := time.ParseDuration(hourDeparture)
	return StopPointVj{
		Id:            id,
		GtfsStopCode:  code,
		DepartureTime: time.Now().Add(t),
	}
}

// GetStatusLastLoadAt get last_load_at field from the status url.
// This field take the last date at the static data reload.
func GetStatusPublicationDate(uri url.URL, token string, connectionTimeout time.Duration) (string, error) {
	callUrl := fmt.Sprintf(URL_GET_LAST_LOAD, uri.String())
	resp, err := CallNavitia(callUrl, token, connectionTimeout)
	if err != nil {
		return "", err
	}

	navitiaStatus := &Status{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaStatus)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return "", err
	}

	return navitiaStatus.Status.PublicationDate, nil
}

// GetVehicleJourney get object vehicle journey from Navitia compared to ODITI vehicle id.
func GetVehiclesJourneysWithLine(codeLine, line string, uri url.URL, token string, connectionTimeout time.Duration,
	location *time.Location) ([]VehicleJourney, error) {
	loc, _ := time.LoadLocation(location.String())
	dateBefore := time.Now().In(loc).Add(-1 * time.Hour).Format("20060102T150405")
	dateAfter := time.Now().In(loc).Add(1 * time.Hour).Format("20060102T150405")
	callUrl := fmt.Sprintf(URL_GET_VEHICLES_JOURNEYS_LINE, uri.String(), codeLine, dateBefore, dateAfter)

	resp, err := CallNavitia(callUrl, token, connectionTimeout)
	if err != nil {
		return nil, err
	}

	navitiaVJ := &NavitiaVehicleJourney{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaVJ)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	return CreateVehicleJourney(navitiaVJ, line, time.Now()), nil
}

// This method call Navitia api with specific url and return a request response
func CallNavitia(callUrl string, token string, connectionTimeout time.Duration) (*http.Response, error) {
	resp, err := utils.GetHttpClient(callUrl, token, "Authorization", connectionTimeout)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	err = utils.CheckResponseStatus(resp)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}
	return resp, nil
}

// CreateVehicleJourney create a new vehicle journey with all stop point from Navitia
func CreateVehicleJourney(navitiaVJ *NavitiaVehicleJourney, line string, createDate time.Time) []VehicleJourney {
	sp := make([]StopPointVj, 0)
	vjs := make([]VehicleJourney, 0)
	var stopPointVj StopPointVj
	var sourceCode string

	for _, vj := range navitiaVJ.VehicleJourneys {
		for i := 0; i < len(vj.StopTimes); i++ {
			for j := 0; j < len(vj.StopTimes[i].StopPoint.Codes); j++ {
				if vj.StopTimes[i].StopPoint.Codes[j].Type == VALUE_FROM_TYPE {
					stopCode := vj.StopTimes[i].StopPoint.Codes[j].Value
					stopId := vj.StopTimes[i].StopPoint.ID
					departure := vj.StopTimes[i].DepartureTime
					stopPointVj = NewStopPointVj(stopId, stopCode, departure)
				}
			}
			sp = append(sp, stopPointVj)
		}

		for _, code := range vj.Codes {
			if code.Type == VALUE_FROM_TYPE {
				sourceCode = code.Value
				break
			}
		}

		vehiclejourney := NewVehicleJourney(vj.ID, sourceCode, line, vj.JourneyPattern.Route.DirectionType, sp, createDate)
		vjs = append(vjs, *vehiclejourney)
	}
	return vjs
}
