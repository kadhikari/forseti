package forseti

import (
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/text/encoding/charmap"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/CanalTP/forseti/internal/departures"
	"github.com/CanalTP/forseti/internal/equipments"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/utils"
)

var (
	parkingsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "parkings",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	parkingsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "parkings",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})

	occupanciesLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_occupancies",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	occupanciesLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "vehicle_occupancies",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})
)

func init() {
	prometheus.MustRegister(departures.DepartureLoadingDuration)
	prometheus.MustRegister(departures.DepartureLoadingErrors)
	prometheus.MustRegister(parkingsLoadingDuration)
	prometheus.MustRegister(parkingsLoadingErrors)
	prometheus.MustRegister(equipments.EquipmentsLoadingDuration)
	prometheus.MustRegister(equipments.EquipmentsLoadingErrors)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingDuration)
	prometheus.MustRegister(freefloatings.FreeFloatingsLoadingErrors)
}

func RefreshParkings(manager *DataManager, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := utils.GetFile(uri, connectionTimeout)
	if err != nil {
		parkingsLoadingErrors.Inc()
		return err
	}

	parkingsConsumer := makeParkingLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, parkingsConsumer, loadDataOptions)
	if err != nil {
		parkingsLoadingErrors.Inc()
		return err
	}

	manager.UpdateParkings(parkingsConsumer.parkings)
	parkingsLoadingDuration.Observe(time.Since(begin).Seconds())

	return nil
}

func getCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "ISO-8859-1" {
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	}

	return nil, fmt.Errorf("Unknown Charset")
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

func LoadStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]StopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, spFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		occupanciesLoadingErrors.Inc()
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
		occupanciesLoadingErrors.Inc()
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
	resp, err := GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	navitiaRoutes := &data.NavitiaRoutes{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaRoutes)
	if err != nil {
		occupanciesLoadingErrors.Inc()
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
	resp, err = GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	navitiaRoutes = &data.NavitiaRoutes{}
	decoder = json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaRoutes)
	if err != nil {
		occupanciesLoadingErrors.Inc()
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

func LoadRoutesForAllLines(manager *DataManager, navitia_url url.URL, navitia_token string,
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
	manager.InitRouteSchedule(routeSchedules)
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
	resp, err := GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	predicts := &data.PredictionData{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(predicts)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	predictions := LoadPredictionsData(predicts, location)
	fmt.Println("*** predictions size: ", len(predictions))
	return predictions, nil
}

func CreateOccupanciesFromPredictions(manager *DataManager, predictions []Prediction) map[int]VehicleOccupancy {
	// create vehicleOccupancy with "Charge" using StopPoints and Courses in the manager for each element in Prediction
	occupanciesWithCharge := make(map[int]VehicleOccupancy)
	var vehicleJourneyId = ""
	for _, predict := range predictions {
		if predict.Order == 0 {
			firstTime, err := manager.GetCourseFirstTime(predict)
			if err != nil {
				continue
			}
			// Schedule datettime = predict.Date + firstTime
			dateTime := utils.AddDateAndTime(predict.Date, firstTime)
			vehicleJourneyId = manager.GetVehicleJourneyId(predict, dateTime)
		}

		if len(vehicleJourneyId) > 0 {
			// Fetch StopId from manager corresponding to predict.StopName and predict.Direction
			stopId := manager.GetStopId(predict.StopName, predict.Direction)
			rs := manager.GetRouteSchedule(vehicleJourneyId, stopId, predict.Direction)
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

func RefreshVehicleOccupancies(manager *DataManager, predict_url url.URL, predict_token string,
	connectionTimeout time.Duration, location *time.Location) error {
	// Continue using last loaded data if loading is deactivated
	if !manager.LoadOccupancyData() {
		return nil
	}
	begin := time.Now()
	predictions, _ := LoadPredictions(predict_url, predict_token, connectionTimeout, location)
	occupanciesWithCharge := CreateOccupanciesFromPredictions(manager, predictions)

	manager.UpdateVehicleOccupancies(occupanciesWithCharge)
	occupanciesLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func LoadAllForVehicleOccupancies(manager *DataManager, files_uri, navitia_url, predict_url url.URL, navitia_token,
	predict_token string, connectionTimeout time.Duration, location *time.Location) error {
	// Load referential Stoppoints file
	stopPoints, err := LoadStopPoints(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	manager.InitStopPoint(stopPoints)

	// Load referential course file
	courses, err := LoadCourses(files_uri, connectionTimeout)
	if err != nil {
		return err
	}
	manager.InitCourse(courses)

	// Load all RouteSchedules for line 40 and 45
	err = LoadRoutesForAllLines(manager, navitia_url, navitia_token, connectionTimeout, location)
	if err != nil {
		return err
	}

	// Load predictions for external service and update VehicleOccupancies with charge
	err = RefreshVehicleOccupancies(manager, predict_url, predict_token, connectionTimeout, location)
	if err != nil {
		return err
	}
	return nil
}
