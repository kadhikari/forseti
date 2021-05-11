package vehicleoccupancies

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/CanalTP/forseti"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

/* ---------------------------------------------------------------------
// Structure and Consumer to creates Vehicle occupancies GTFS-RT objects
--------------------------------------------------------------------- */
type VehicleOccupanciesGtfsRtContext struct {
	voContext           *VehicleOccupanciesContext
	vehiclesJourney     *[]VehicleJourney
	vehiclesGtfsRt      *[]VehicleGtfsRt
	lastLoadNavitia     string
	lastTimestampGtfsRt string
}

func (d *VehicleOccupanciesGtfsRtContext) GetVehicleJourneys() (vehicleJourneys []VehicleJourney) {
	return *d.vehiclesJourney
}

func (d *VehicleOccupanciesGtfsRtContext) GetVehiclesGtfsRts() (vehiclesGtfsRts []VehicleGtfsRt) {
	return *d.vehiclesGtfsRt
}

func (d *VehicleOccupanciesGtfsRtContext) GetLastLoadNavitia() (lastLoadNavitia string) {
	return d.lastLoadNavitia
}

func (d *VehicleOccupanciesGtfsRtContext) GetLastTimestampGtfsRt() (lastTimestampGtfsRt string) {
	return d.lastTimestampGtfsRt
}

func (d *VehicleOccupanciesGtfsRtContext) UpdateVehicleOccupancies(
	vehicleOccupancies map[int]VehicleOccupancy) {
}

func (d *VehicleOccupanciesGtfsRtContext) UpdateVehicleOccupancy(
	vehicleOccupancy VehicleOccupancy) {
}

func (d *VehicleOccupanciesGtfsRtContext) CheckLastLoadChanged(lastLoadAt string) bool {
	return false
}

func (d *VehicleOccupanciesGtfsRtContext) CheckLastChangedGtfsRT(lastChanged string) bool {
	if d.lastTimestampGtfsRt == "" || d.lastTimestampGtfsRt != lastChanged {
		d.lastTimestampGtfsRt = lastChanged
		return true
	}

	return false
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListOldVehicleOccupancies() {
}

func (d *VehicleOccupanciesGtfsRtContext) CleanListOldVehicleJourney() {
}

func (d *VehicleOccupanciesGtfsRtContext) AddListVehicleJourney(VehicleJourney) {
}

/********* INTERFACE METHODS IMPLEMENTS *********/

func (d *VehicleOccupanciesGtfsRtContext) RefreshVehicleOccupanciesLoop(externalURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location) {

	refreshVehicleOccupanciesLoop(d, externalURI, externalToken, navitiaURI, navitiaToken,
		loadExternalRefresh, connectionTimeout, location)
}

func (d *VehicleOccupanciesGtfsRtContext) LoadDataExternalSource(uri url.URL, token string,
	connectionTimeout time.Duration, location *time.Location) (*GtfsRt, error) {

	// External source http/REST
	if token == "nil" {
		resp, err := http.Get(uri.String())
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		err = utils.CheckResponseStatus(resp)
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		gtfsRtData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			VehicleOccupanciesLoadingErrors.Inc()
			return nil, err
		}

		gtfsRt, err := parseVehiclesResponse(gtfsRtData)
		if err != nil {
			return nil, err
		}

		return gtfsRt, nil

	} else {
		//TODO: add code in case of this service call external API
		return nil, nil
	}
}

/********* PRIVATE FUNCTIONS *********/

// main loop to refresh vehicle_occupancies from Gtfs-rt flux
func refreshVehicleOccupanciesLoop(context *VehicleOccupanciesGtfsRtContext, predictionURI url.URL,
	externalToken string, navitiaURI url.URL, navitiaToken string, loadExternalRefresh,
	connectionTimeout time.Duration, location *time.Location) {

	// Wait 10 seconds before reloading vehicleoccupacy informations
	time.Sleep(10 * time.Second)
	for {
		err := refreshVehicleOccupancies(context, predictionURI, externalToken, navitiaURI, navitiaToken,
			connectionTimeout, location)
		if err != nil {
			logrus.Error("Error while loading VehicleOccupancy GTFS-RT data: ", err)
		} else {
			logrus.Debug("vehicle_occupancies GTFS-RT data updated")
		}
		time.Sleep(loadExternalRefresh)
	}
}

func refreshVehicleOccupancies(context *VehicleOccupanciesGtfsRtContext, external_url url.URL,
	external_token string, navitiaURI url.URL, navitiaToken string, connectionTimeout time.Duration,
	location *time.Location) error {

	begin := time.Now()

	// Get all data from Gtfs-rt flux
	gtfsRt, err := context.LoadDataExternalSource(external_url, external_token, connectionTimeout, location)
	if err != nil {
		return errors.Errorf("loading external source: %s", err)
	}

	// Get status Last load from Navitia and check if data loaded recently
	lastLoadAt, _ := GetStatusLastLoadAt(navitiaURI, navitiaToken, connectionTimeout)

	if context.CheckLastLoadChanged(lastLoadAt) {
		context.CleanListOldVehicleOccupancies()
	}

	// Clean list VehicleJourney for vehicle older than delay parameter
	// TODO: add parameter in config
	context.CleanListOldVehicleJourney()

	// Check last modifiation from gtfs-rt flux
	if context.CheckLastChangedGtfsRT(gtfsRt.Timestamp) {

		mapVehicleOccupancies := *context.voContext.VehicleOccupancies
		// if gtfs-rt vehicle exist in map of vehicle occupancies
		if veh, ok := mapVehicleOccupancies[12]; ok { // change "12" by id from gtfs-rt data
			veh.Occupancy = 1 // change "1" by occupancy from gtfs-rt data
			context.UpdateVehicleOccupancy(veh)
		} else {
			// Get status Last load from Navitia and check if data loaded recently
			vj, _ := GetVehicleJourney("id_code", navitiaURI, navitiaToken, connectionTimeout)

			// add in vehicle journey list
			context.AddListVehicleJourney(*vj)

			// add in vehicle occupancy list
			occupancy := createOccupanciesFromDataSource(*vj, gtfsRt.Vehicles[12])
			context.UpdateVehicleOccupancy(occupancy)
		}
	}

	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func createOccupanciesFromDataSource(vehicleJourney VehicleJourney,
	vehicles VehicleGtfsRt) VehicleOccupancy {

	return VehicleOccupancy{}
}

// Method to parse data from GTFS-RT
func parseVehiclesResponse(b []byte) (*GtfsRt, error) {
	fm := new(forseti.FeedMessage)
	err := proto.Unmarshal(b, fm)
	if err != nil {
		return nil, err
	}

	strTimestamp := strconv.FormatUint(fm.Header.GetTimestamp(), 10)

	vehicles := make([]VehicleGtfsRt, 0, len(fm.GetEntity()))
	for _, entity := range fm.GetEntity() {
		var vehPos *forseti.VehiclePosition = entity.GetVehicle()
		var pos *forseti.Position = vehPos.GetPosition()
		var trip *forseti.TripDescriptor = vehPos.GetTrip()

		veh := VehicleGtfsRt{
			VehicleID: vehPos.GetVehicle().GetId(),
			StopId:    vehPos.GetStopId(),
			Label:     vehPos.GetVehicle().GetLabel(),
			Time:      vehPos.GetTimestamp(),
			Speed:     pos.GetSpeed(),
			Bearing:   pos.GetBearing(),
			Route:     trip.GetRouteId(),
			Trip:      trip.GetTripId(),
			Latitude:  pos.GetLatitude(),
			Longitude: pos.GetLongitude(),
			Occupancy: uint32(vehPos.GetOccupancyStatus()),
		}
		vehicles = append(vehicles, veh)
	}

	gtfsRt := NewGtfsRt(strTimestamp, vehicles)
	return gtfsRt, nil
}

/* ---------------------------------------------------------
// Structure and Consumer to creates Vehicle GTFS-RT objects
--------------------------------------------------------- */
type GtfsRt struct {
	Timestamp string
	Vehicles  []VehicleGtfsRt
}

type VehicleGtfsRt struct {
	VehicleID string
	StopId    string
	Label     string
	Time      uint64
	Speed     float32
	Bearing   float32
	Route     string
	Trip      string
	Latitude  float32
	Longitude float32
	Occupancy uint32
}

func NewGtfsRt(timestamp string, v []VehicleGtfsRt) *GtfsRt {
	return &GtfsRt{
		Timestamp: timestamp,
		Vehicles:  v,
	}
}
