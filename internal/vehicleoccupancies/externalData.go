package vehicleoccupancies

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/data"
	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

/* ---------------------------------------------------------
// ***************** ODITI EXTERNAL SOURCE *****************
--------------------------------------------------------- */

func LoadPredictions(connector *connectors.Connector, location *time.Location) ([]Prediction, error) {
	//futuredata/getfuturedata?start_time=2021-02-15&end_time=2021-02-16
	begin := time.Now()
	start_date := begin.Format("2006-01-02")
	end_date := begin.AddDate(0, 0, 0).Format("2006-01-02")
	urlPath := connector.GetUrl()
	callUrl := fmt.Sprintf("%s/futuredata/getfuturedata?start_time=%s&end_time=%s", urlPath.String(), start_date, end_date)
	header := "Ocp-Apim-Subscription-Key"
	resp, err := utils.GetHttpClient(callUrl, connector.GetToken(), header, connector.GetConnectionTimeout())
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
