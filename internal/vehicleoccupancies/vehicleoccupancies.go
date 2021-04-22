package vehicleoccupancies

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

var spFileName = "mapping_stops.csv"
var courseFileName = "extraction_courses.csv"

func ManageVehicleOccupancyStatus(context *VehicleOccupanciesContext, vehicleOccupanciesActive bool) {
	context.ManageVehicleOccupancyStatus(vehicleOccupanciesActive)
}

func RefreshVehicleOccupanciesLoop(context *VehicleOccupanciesContext,
	predictionURI url.URL,
	predictionToken string,
	predictionRefresh,
	connectionTimeout time.Duration,
	location *time.Location) {
	if len(predictionURI.String()) == 0 || predictionRefresh.Seconds() <= 0 {
		logrus.Debug("VehicleOccupancy data refreshing is disabled")
		return
	}

	if (context.courses == nil || context.stopPoints == nil) ||
		(len(*context.courses) == 0 || len(*context.stopPoints) == 0) {
		logrus.Error("VEHICULE_OCCUPANCIES: routine Vehicule_occupancies stopped, no stopPoints or courses loaded at start")
	} else {
		// Wait 10 seconds before reloading vehicleoccupacy informations
		time.Sleep(10 * time.Second)
		for {
			err := RefreshVehicleOccupancies(context, predictionURI, predictionToken, connectionTimeout, location)
			if err != nil {
				logrus.Error("Error while reloading VehicleOccupancy data: ", err)
			} else {
				logrus.Debug("vehicle_occupancies data updated")
			}
			time.Sleep(predictionRefresh)
		}
	}
}

func RefreshVehicleOccupancies(context *VehicleOccupanciesContext, predict_url url.URL, predict_token string,
	connectionTimeout time.Duration, location *time.Location) error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadOccupancyData() {
		return nil
	}
	if context.routeSchedules == nil {
		return errors.New("RouteSchedule contains no data")
	}
	begin := time.Now()
	predictions, err := LoadPredictions(predict_url, predict_token, connectionTimeout, location)
	if len(predictions) == 0 {
		return fmt.Errorf("Predictions contains no data: %s", err)
	}
	occupanciesWithCharge := CreateOccupanciesFromPredictions(context, predictions)

	context.UpdateVehicleOccupancies(occupanciesWithCharge)
	VehicleOccupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func RefreshRouteSchedulesLoop(context *VehicleOccupanciesContext, navitiaURI url.URL, navitiaToken string,
	routeScheduleRefresh, connectionTimeout time.Duration, location *time.Location) {

	if len(navitiaURI.String()) == 0 || routeScheduleRefresh.Seconds() <= 0 {
		logrus.Debug("RouteSchedule data refreshing is disabled")
		return
	}

	if (context.courses == nil || context.stopPoints == nil) ||
		(len(*context.courses) == 0 || len(*context.stopPoints) == 0) {
		logrus.Error("VEHICULE_OCCUPANCIES: routine Route_schedule stopped, no stopPoints or courses loaded at start")
	} else {
		for {
			err := LoadRoutesForAllLines(context, navitiaURI, navitiaToken, connectionTimeout, location)
			if err != nil {
				logrus.Error("Error while reloading RouteSchedule data: ", err)
			} else {
				logrus.Debug("RouteSchedule data updated")
			}
			time.Sleep(routeScheduleRefresh)
		}
	}
}

func LoadStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]StopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, spFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	stopPointsConsumer := makeStopPointLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, stopPointsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return stopPointsConsumer.stopPoints, nil
}

func LoadCourses(uri url.URL, connectionTimeout time.Duration) (map[string][]Course, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, courseFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		VehicleOccupanciesLoadingErrors.Inc()
		return nil, err
	}

	courseLineConsumer := makeCourseLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, courseLineConsumer, loadDataOptions)
	if err != nil {
		fmt.Println("LoadCourses Error: ", err.Error())
		return nil, err
	}
	return courseLineConsumer.courses, nil
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

func LoadRoutesWithDirection(startIndex int, uri url.URL, token, direction string,
	connectionTimeout time.Duration, location *time.Location) ([]RouteSchedule, error) {
	// Call and load routes for line=40
	dateTime := time.Now().Truncate(24 * time.Hour).Format("20060102T000000")
	callUrl := fmt.Sprintf("%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s", uri.String(),
		"line:0:004004040:40", direction, dateTime)
	header := "Authorization"
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
	callUrl = fmt.Sprintf("%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s", uri.String(),
		"line:0:004004029:45", direction, dateTime)
	resp, err = utils.GetHttpClient(callUrl, token, header, connectionTimeout)
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

func LoadRoutesForAllLines(context *VehicleOccupanciesContext, navitia_url url.URL, navitia_token string,
	connectionTimeout time.Duration, location *time.Location) error {
	// Load Forward RouteSchedules (sens=0) for lines 40 and 45
	startIndex := 1
	routeSchedules, err := LoadRoutesWithDirection(
		startIndex,
		navitia_url,
		navitia_token,
		"forward",
		connectionTimeout,
		location)
	if err != nil {
		return err
	}
	// Load Backward RouteSchedules (sens=1) for lines 40 and 45
	startIndex = len(routeSchedules) + 1
	backward, err := LoadRoutesWithDirection(
		startIndex,
		navitia_url,
		navitia_token,
		"backward",
		connectionTimeout,
		location)
	if err != nil {
		return err
	}

	// Concat two arrays
	for i := range backward {
		routeSchedules = append(routeSchedules, backward[i])
	}

	context.InitRouteSchedule(routeSchedules)
	return nil
}

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

func CreateOccupanciesFromPredictions(context *VehicleOccupanciesContext,
	predictions []Prediction) map[int]VehicleOccupancy {
	// create vehicleOccupancy with "Charge" using StopPoints and Courses in the manager for each element in Prediction
	occupanciesWithCharge := make(map[int]VehicleOccupancy)
	var vehicleJourneyId = ""
	for _, predict := range predictions {
		if predict.Order == 0 {
			firstTime, err := context.GetCourseFirstTime(predict)
			if err != nil {
				continue
			}
			// Schedule datettime = predict.Date + firstTime
			dateTime := utils.AddDateAndTime(predict.Date, firstTime)
			vehicleJourneyId = context.GetVehicleJourneyId(predict, dateTime)
		}

		if len(vehicleJourneyId) > 0 {
			// Fetch StopId from manager corresponding to predict.StopName and predict.Direction
			stopId := context.GetStopId(predict.StopName, predict.Direction)
			rs := context.GetRouteSchedule(vehicleJourneyId, stopId, predict.Direction)
			if rs != nil {
				vo, err := NewVehicleOccupancy(*rs, utils.CalculateOccupancy(predict.Occupancy))

				if err != nil {
					continue
				}
				occupanciesWithCharge[vo.Id] = *vo
			}
		}
	}
	return occupanciesWithCharge
}

func LoadAllForVehicleOccupancies(context *VehicleOccupanciesContext, files_uri, navitia_url,
	predict_url url.URL, navitia_token, predict_token string, connectionTimeout time.Duration,
	location *time.Location) error {
	// Load referential Stoppoints file
	stopPoints, err := LoadStopPoints(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	context.InitStopPoint(stopPoints)

	// Load referential course file
	courses, err := LoadCourses(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	context.InitCourse(courses)

	// Load all RouteSchedules for line 40 and 45
	err = LoadRoutesForAllLines(context, navitia_url, navitia_token, connectionTimeout, location)
	if err != nil {
		return err
	}

	// Load predictions for external service and update VehicleOccupancies with charge
	err = RefreshVehicleOccupancies(context, predict_url, predict_token, connectionTimeout, location)
	if err != nil {
		return err
	}
	return nil
}
