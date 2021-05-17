package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/CanalTP/forseti"
	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

/* ---------------------------------------------------------
// ***************** ODITI EXTERNAL SOURCE *****************
--------------------------------------------------------- */

func LoadPredictions(uri url.URL, token string, connectionTimeout time.Duration,
	location *time.Location) ([]Prediction, error) {
	//futuredata/getfuturedata?start_time=2021-02-15&end_time=2021-02-16
	begin := time.Now()
	start_date := begin.Format("2006-01-02")
	end_date := begin.AddDate(0, 0, 0).Format("2006-01-02")
	callUrl := fmt.Sprintf("%s/futuredata/getfuturedata?start_time=%s&end_time=%s", uri.String(), start_date, end_date)
	header := "Ocp-Apim-Subscription-Key"
	resp, err := utils.GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	err = utils.CheckResponseStatus(resp)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	predicts := &data.PredictionData{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(predicts)
	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	predictions := LoadPredictionsData(predicts, location)
	logrus.Info("*** predictions size: ", len(predictions))
	return predictions, nil
}

func LoadPredictionsData(predictionData *data.PredictionData, location *time.Location) []Prediction {
	predictions := make([]Prediction, 0)
	for _, predict := range *predictionData {
		predictions = append(predictions, *NewPrediction(predict, location))
	}
	return predictions
}

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
