package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

func LoadPredictionsData(predictionData *data.PredictionData, location *time.Location) []Prediction {
	predictions := make([]Prediction, 0)
	for _, predict := range *predictionData {
		predictions = append(predictions, *NewPrediction(predict, location))
	}
	return predictions
}

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
