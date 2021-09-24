package gtfsrtvehiclepositions

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/CanalTP/forseti/google_transit"
	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

/* ---------------------------------------------------------
// **************** GTFS-RT EXTERNAL SOURCE ****************
--------------------------------------------------------- */

// Structure and Consumer to creates Vehicle GTFS-RT objects
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

func LoadGtfsRt(connector *connectors.Connector) (*GtfsRt, error) {
	resp, err := utils.GetHttpClient_(connector.GetUrl(), connector.GetToken(), "Authorization",
		connector.GetConnectionTimeout())
	if err != nil {
		return nil, err
	}
	gtfsRtData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	gtfsRt, err := ParseVehiclesResponse(gtfsRtData)
	if err != nil {
		return nil, err
	}

	if len(gtfsRt.Vehicles) == 0 {
		return gtfsRt, fmt.Errorf("no data loaded from GTFS-RT")
	}

	logrus.Debug("*** Gtfs-rt size: ", len(gtfsRt.Vehicles))
	return gtfsRt, nil
}

// Method to parse data from GTFS-RT
func ParseVehiclesResponse(b []byte) (*GtfsRt, error) {
	fm := new(google_transit.FeedMessage)
	err := proto.Unmarshal(b, fm)
	if err != nil {
		return nil, err
	}

	strTimestamp := strconv.FormatUint(fm.Header.GetTimestamp(), 10)

	vehicles := make([]VehicleGtfsRt, 0, len(fm.GetEntity()))
	for _, entity := range fm.GetEntity() {
		var vehPos *google_transit.VehiclePosition = entity.GetVehicle()
		var pos *google_transit.Position = vehPos.GetPosition()
		var trip *google_transit.TripDescriptor = vehPos.GetTrip()

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
