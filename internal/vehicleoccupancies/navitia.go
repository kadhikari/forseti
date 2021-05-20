package vehicleoccupancies

// Declaration of the different structures loaded from Navitia.
// Methods for querying Navitia on the various data to be loaded.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
)

const (
	URL_GET_LAST_LOAD       = "%s/status?"
	URL_GET_VEHICLE_JOURNEY = "%s/vehicle_journeys?filter=vehicle_journey.has_code(%s)&"
	STOP_POINT_CODE         = "gtfs_stop_code" // type code vehicle journey Navitia, the same of stop_id from Gtfs-rt
	URL_GET_ROUTES          = "%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s"
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
		Name      string `json:"name"`
		StopTimes []struct {
			StopPoint struct {
				Codes []struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"codes"`
				Coord struct {
					Lat string `json:"lat"`
					Lon string `json:"lon"`
				} `json:"coord"`
				ID string `json:"id"`
			} `json:"stop_point"`
		} `json:"stop_times"`
		ID string `json:"id"`
	} `json:"vehicle_journeys"`
}

// Structure and Consumer to creates Vehicle Journey objects
type VehicleJourney struct {
	VehicleID   string // vehicle journey id Navitia
	CodesSource string // vehicle id from gtfs-rt
	StopPoints  *[]StopPointVj
	CreateDate  time.Time
}

func NewVehicleJourney(vehicleId string, codesSource string, stopPoints []StopPointVj,
	date time.Time) *VehicleJourney {
	return &VehicleJourney{
		VehicleID:   vehicleId,
		CodesSource: codesSource,
		StopPoints:  &stopPoints,
		CreateDate:  date,
	}
}

// Structure and Consumer to creates Stop point from Vehicle Journey Navitia objects
type StopPointVj struct {
	Id           string // Stoppoint uri from navitia
	GtfsStopCode string // StopPoint code gtfs-rt
}

func NewStopPointVj(id string, code string) StopPointVj {
	return StopPointVj{
		Id:           id,
		GtfsStopCode: code,
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

// GetVehicleJourney get object vehicle journey from Navitia compared to GTFS-RT vehicle id.
func GetVehicleJourney(id_gtfsRt string, uri url.URL, token string, connectionTimeout time.Duration) (
	*VehicleJourney, error) {
	sourceCode := fmt.Sprint("source%2C", id_gtfsRt)
	callUrl := fmt.Sprintf(URL_GET_VEHICLE_JOURNEY, uri.String(), sourceCode)
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

	return CreateVehicleJourney(navitiaVJ, id_gtfsRt, time.Now()), nil
}

// Get all Routes for Oditi service, just lines 40 and 45
func LoadRoutesWithDirection(startIndex int, uri url.URL, token, direction string,
	connectionTimeout time.Duration, location *time.Location) ([]RouteSchedule, error) {
	// Call and load routes for line=40
	dateTime := time.Now().Truncate(24 * time.Hour).Format("20060102T000000")
	callUrl := fmt.Sprintf(URL_GET_ROUTES, uri.String(), "line:0:004004040:40", direction, dateTime)
	resp, err := CallNavitia(callUrl, token, connectionTimeout)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	navitiaRoutes := &data.NavitiaRoutes{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaRoutes)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	sens := 0
	if direction == "backward" {
		sens = 1
	}
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, location)

	// Call and load routes for line=45
	callUrl = fmt.Sprintf(URL_GET_ROUTES, uri.String(), "line:0:004004029:45", direction, dateTime)
	resp, err = CallNavitia(callUrl, token, connectionTimeout)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	navitiaRoutes = &data.NavitiaRoutes{}
	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaRoutes)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	startIndex += len(routeSchedules)

	rs45 := LoadRouteSchedulesData(startIndex+1, navitiaRoutes, sens, location)
	// Concat two arrays
	for i := range rs45 {
		routeSchedules = append(routeSchedules, rs45[i])
	}
	return routeSchedules, nil
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
func CreateVehicleJourney(navitiaVJ *NavitiaVehicleJourney, id_gtfsRt string, createDate time.Time) *VehicleJourney {
	sp := make([]StopPointVj, 0)
	var stopPointVj StopPointVj
	for i := 0; i < len(navitiaVJ.VehicleJourneys[0].StopTimes); i++ {
		for j := 0; j < len(navitiaVJ.VehicleJourneys[0].StopTimes[i].StopPoint.Codes); j++ {
			if navitiaVJ.VehicleJourneys[0].StopTimes[i].StopPoint.Codes[j].Type == STOP_POINT_CODE {
				stopCode := navitiaVJ.VehicleJourneys[0].StopTimes[i].StopPoint.Codes[j].Value
				stopId := navitiaVJ.VehicleJourneys[0].StopTimes[i].StopPoint.ID
				stopPointVj = NewStopPointVj(stopId, stopCode)
			}
		}
		sp = append(sp, stopPointVj)
	}
	vj := NewVehicleJourney(navitiaVJ.VehicleJourneys[0].ID, id_gtfsRt, sp, createDate)
	return vj
}

func LoadRouteSchedulesData(startIndex int, navitiaRoutes *data.NavitiaRoutes, direction int,
	location *time.Location) []RouteSchedule {
	// Read RouteSchedule response body in json
	// Normally there is one vehiclejourney for each departure.
	routeScheduleList := make([]RouteSchedule, 0)
	lineCode := navitiaRoutes.RouteSchedules[0].DisplayInformations.Code
	for i := 0; i < len(navitiaRoutes.RouteSchedules[0].Table.Rows); i++ {
		for j := 0; j < len(navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes); j++ {
			rs, err := NewRouteSchedule(
				lineCode,
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].StopPoint.ID,
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes[j].Links[0].Value,
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes[j].DateTime,
				direction,
				startIndex,
				(i == 0), // StopPoint of departure for a distinct vehiclejourney
				location)
			if err != nil {
				continue
			}
			routeScheduleList = append(routeScheduleList, *rs)
			startIndex++
		}
	}
	return routeScheduleList
}
