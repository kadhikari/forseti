package vehiclelocations

// Declaration of the different structures loaded from Navitia.
// Methods for querying Navitia on the various data to be loaded.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/CanalTP/forseti/internal/utils"
)

const (
	URL_GET_LAST_LOAD       = "%s/status?"
	URL_GET_VEHICLE_JOURNEY = "%s/vehicle_journeys?filter=vehicle_journey.has_code(%s)&"
	STOP_POINT_CODE         = "gtfs_stop_code" // type code vehicle journey Navitia, the same of stop_id from Gtfs-rt
	URL_GET_ROUTES          = "%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s"
)

type Navitia struct {
	url               url.URL
	token             string
	connectionTimeout time.Duration
	lastLoadNavitia   string
	mutex             sync.Mutex
}

func (d *Navitia) GetUrl() url.URL {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.url
}

func (d *Navitia) GetToken() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.token
}

func (d *Navitia) CheckLastLoadChanged(lastLoadAt string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.lastLoadNavitia == "" || d.lastLoadNavitia != lastLoadAt {
		d.lastLoadNavitia = lastLoadAt
		return true
	}

	return false
}

func NewNavitia(url url.URL, token string, connectionTimeout time.Duration) *Navitia {
	return &Navitia{
		url:               url,
		token:             token,
		connectionTimeout: connectionTimeout,
	}
}

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
		StopTimes []struct {
			StopPoint struct {
				Codes []struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"codes"`
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
func GetStatusPublicationDate(navitia *Navitia) (string, error) {
	callUrl := fmt.Sprintf(URL_GET_LAST_LOAD, navitia.url.String())
	resp, err := CallNavitia(callUrl, navitia.token, navitia.connectionTimeout)
	if err != nil {
		return "", err
	}

	navitiaStatus := &Status{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaStatus)
	if err != nil {
		return "", err
	}

	return navitiaStatus.Status.PublicationDate, nil
}

// GetVehicleJourney get object vehicle journey from Navitia compared to GTFS-RT vehicle id.
func GetVehicleJourney(id_gtfsRt string, navitia *Navitia) (
	*VehicleJourney, error) {
	sourceCode := fmt.Sprint("source%2C", id_gtfsRt)
	callUrl := fmt.Sprintf(URL_GET_VEHICLE_JOURNEY, navitia.url.String(), sourceCode)
	resp, err := CallNavitia(callUrl, navitia.token, navitia.connectionTimeout)
	if err != nil {
		return nil, err
	}

	navitiaVJ := &NavitiaVehicleJourney{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaVJ)
	if err != nil {
		return nil, err
	}

	return CreateVehicleJourney(navitiaVJ, id_gtfsRt, time.Now()), nil
}

// This method call Navitia api with specific url and return a request response
func CallNavitia(callUrl string, token string, connectionTimeout time.Duration) (*http.Response, error) {
	resp, err := utils.GetHttpClient(callUrl, token, "Authorization", connectionTimeout)
	if err != nil {
		return nil, err
	}

	err = utils.CheckResponseStatus(resp)
	if err != nil {
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
