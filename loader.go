package forseti

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"
	"net/http"
	"encoding/json"
	"github.com/pkg/sftp"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/charmap"
)

var (
	departureLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "departures",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	departureLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "departures",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})

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

	equipmentsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "equipments",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	equipmentsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "equipments",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})

	freeFloatingsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "forseti",
		Subsystem: "free_floatings",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	freeFloatingsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "forseti",
		Subsystem: "free_floatings",
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
	prometheus.MustRegister(departureLoadingDuration)
	prometheus.MustRegister(departureLoadingErrors)
	prometheus.MustRegister(parkingsLoadingDuration)
	prometheus.MustRegister(parkingsLoadingErrors)
	prometheus.MustRegister(equipmentsLoadingDuration)
	prometheus.MustRegister(equipmentsLoadingErrors)
	prometheus.MustRegister(freeFloatingsLoadingDuration)
	prometheus.MustRegister(freeFloatingsLoadingErrors)
}

func getFile(uri url.URL, connectionTimeout time.Duration) (io.Reader, error) {
	if uri.Scheme == "sftp" {
		return getFileWithSftp(uri, connectionTimeout)
	} else if uri.Scheme == "file" {
		return getFileWithFS(uri)
	} else {
		return nil, fmt.Errorf("Unsupported protocols %s", uri.Scheme)
	}

}

func getFileWithFS(uri url.URL) (io.Reader, error) {
	file, err := os.Open(uri.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buffer bytes.Buffer
	if _, err = buffer.ReadFrom(file); err != nil {
		return nil, err
	}
	return &buffer, nil
}

func getFileWithSftp(uri url.URL, connectionTimeout time.Duration) (io.Reader, error) {
	password, _ := uri.User.Password()
	sshConfig := &ssh.ClientConfig{
		User: uri.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         connectionTimeout,
	}

	sshClient, err := ssh.Dial("tcp", uri.Host, sshConfig)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()

	// open an SFTP session over an existing ssh connection.
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	file, err := client.Open(uri.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var buffer bytes.Buffer
	if _, err = file.WriteTo(&buffer); err != nil {
		return nil, err
	}
	return &buffer, nil

}

type LoadDataOptions struct {
	skipFirstLine bool
	delimiter     rune
	nbFields      int
}

func LoadData(file io.Reader, lineConsumer LineConsumer) error {

	return LoadDataWithOptions(file, lineConsumer, LoadDataOptions{
		delimiter:     ';',
		nbFields:      0, // do not check record size in csv.reader
		skipFirstLine: false,
	})
}

func LoadDataWithOptions(file io.Reader, lineConsumer LineConsumer, options LoadDataOptions) error {

	location, err := time.LoadLocation(location)
	if err != nil {
		return err
	}

	reader := csv.NewReader(file)
	reader.Comma = options.delimiter
	reader.FieldsPerRecord = options.nbFields

	// Loop through lines & turn into object
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if options.skipFirstLine {
			options.skipFirstLine = false
			continue
		}

		if err := lineConsumer.Consume(line, location); err != nil {
			return err
		}
	}

	lineConsumer.Terminate()
	return nil
}

// CalculateDate adds date and hour parts
func CalculateDate(info Info, location *time.Location) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", info.Date, location)
	if err != nil {
		return time.Now(), err
	}

	hour, err := time.ParseInLocation("15:04:05", info.Hour, location)
	if err != nil {
		return time.Now(), err
	}

	// Add time part to end date
	date = hour.AddDate(date.Year(), int(date.Month())-1, date.Day()-1)

	return date, nil
}

func LoadXmlData(file io.Reader) ([]EquipmentDetail, error) {

	location, err := time.LoadLocation(location)
	if err != nil {
		return nil, err
	}

	XMLdata, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(XMLdata))
	decoder.CharsetReader = getCharsetReader

	var root Root
	err = decoder.Decode(&root)
	if err != nil {
		return nil, err
	}

	equipments := make(map[string]EquipmentDetail)
	//Calculate updated_at from Info.Date and Info.Hour
	updatedAt, err := CalculateDate(root.Info, location)
	if err != nil {
		return nil, err
	}
	// for each root.Data.Lines.Stations create an object Equipment
	for _, l := range root.Data.Lines {
		for _, s := range l.Stations {
			for _, e := range s.Equipments {
				ed, err := NewEquipmentDetail(e, updatedAt, location)
				if err != nil {
					return nil, err
				}
				equipments[ed.ID] = *ed
			}
		}
	}

	// Transform the map to array of EquipmentDetail
	equipmentDetails := make([]EquipmentDetail, 0)
	for _, ed := range equipments {
		equipmentDetails = append(equipmentDetails, ed)
	}

	return equipmentDetails, nil
}

func RefreshDepartures(manager *DataManager, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := getFile(uri, connectionTimeout)
	if err != nil {
		departureLoadingErrors.Inc()
		return err
	}

	departureConsumer := makeDepartureLineConsumer()
	if err = LoadData(file, departureConsumer); err != nil {
		departureLoadingErrors.Inc()
		return err
	}
	manager.UpdateDepartures(departureConsumer.data)
	departureLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func RefreshParkings(manager *DataManager, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := getFile(uri, connectionTimeout)
	if err != nil {
		parkingsLoadingErrors.Inc()
		return err
	}

	parkingsConsumer := makeParkingLineConsumer()
	loadDataOptions := LoadDataOptions{
		delimiter:     ';',
		nbFields:      0,    // We might not have etereogenous lines
		skipFirstLine: true, // First line is a header
	}
	err = LoadDataWithOptions(file, parkingsConsumer, loadDataOptions)
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

func RefreshEquipments(manager *DataManager, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := getFile(uri, connectionTimeout)

	if err != nil {
		equipmentsLoadingErrors.Inc()
		return err
	}

	equipments, err := LoadXmlData(file)
	if err != nil {
		equipmentsLoadingErrors.Inc()
		return err
	}
	manager.UpdateEquipments(equipments)
	equipmentsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func CallHttpClient(siteHost, token string ) (*http.Response, error){
	client := &http.Client{}
	data := url.Values{}
	data.Set("query", "query($id: Int!) {area(id: $id) {vehicles{publicId, provider{name}, id, type, attributes ,latitude: lat, longitude: lng, propulsion, battery, deeplink } } }")
	// Manage area id in the query (Paris id=6)
	// TODO: load vehicles for more than one area (city)
	area_id := 6
	data.Set("variables", fmt.Sprintf("{\"id\": %d}", area_id))
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1?access_token=%s", siteHost, token), bytes.NewBufferString(data.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded; param=value")
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func LoadFreeFloatingData (data *Data) ([]FreeFloating, error) {
	// Read response body in json
	freeFloatings := make([]FreeFloating, 0)
	for i := 0; i < len(data.Data.Area.Vehicles); i++ {
		freeFloatings = append(freeFloatings, *NewFreeFloating(data.Data.Area.Vehicles[i]))
	}
	return freeFloatings, nil
}

func RefreshFreeFloatings(manager *DataManager, uri url.URL, token string, connectionTimeout time.Duration) error {
	begin := time.Now()
	resp, err := CallHttpClient(uri.String(), token)

	if err != nil {
		freeFloatingsLoadingErrors.Inc()
		return err
	}

	data := &Data{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(data)
	if err != nil {
		freeFloatingsLoadingErrors.Inc()
        return err
	}

	freeFloatings, err := LoadFreeFloatingData(data)
	if err != nil {
		freeFloatingsLoadingErrors.Inc()
		return err
	}

	manager.UpdateFreeFloating(freeFloatings)
	freeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func GetHttpClient(url, token, header string, connectionTimeout time.Duration) (*http.Response, error){
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
	file, err := getFile(uri, connectionTimeout)

	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	stopPointsConsumer := makeStopPointLineConsumer()
	loadDataOptions := LoadDataOptions{
		delimiter:     ';',
		nbFields:      0,    // We might not have etereogenous lines
		skipFirstLine: true, // First line is a header
	}
	err = LoadDataWithOptions(file, stopPointsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return stopPointsConsumer.stopPoints, nil
}

func LoadCourses(uri url.URL, connectionTimeout time.Duration) (map[string][]Course, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, courseFileName)
	file, err := getFile(uri, connectionTimeout)

	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	courseLineConsumer := makeCourseLineConsumer()
	loadDataOptions := LoadDataOptions{
		delimiter:     ';',
		nbFields:      0,    // We might not have etereogenous lines
		skipFirstLine: true, // First line is a header
	}
	err = LoadDataWithOptions(file, courseLineConsumer, loadDataOptions)
	if err != nil {
		fmt.Println("LoadCourses Error: ", err.Error())
		return nil, err
	}
	return courseLineConsumer.courses, nil
}

func LoadRouteSchedulesData(startIndex int, navitiaRoutes *NavitiaRoutes, sens int, location *time.Location) ([]RouteSchedule) {
	// Read RouteSchedule response body in json
	// Normally there is one vehiclejourney for each departure.
	routeScheduleList := make([]RouteSchedule, 0)
	for i := 0; i < len(navitiaRoutes.RouteSchedules[0].Table.Rows); i++ {
		for j := 0; j < len(navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes); j++ {
			rs, err := NewRouteSchedule(
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].StopPoint.ID,
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes[j].Links[0].Value,
				navitiaRoutes.RouteSchedules[0].Table.Rows[i].DateTimes[j].DateTime,
				sens,
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

func LoadRoutes(startIndex int, uri url.URL, token, direction string,
	connectionTimeout time.Duration, location *time.Location) ([]RouteSchedule, error) {
	dateTime := time.Now().Truncate(24 * time.Hour).Format("20060102T000000")
	callUrl := fmt.Sprintf("%s/lines/%s/route_schedules?direction_type=%s&from_datetime=%s", uri.String(), 
		"line:0:004004040:40", direction, dateTime)
	header := "Authorization"
	resp, err := GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	navitiaRoutes := &NavitiaRoutes{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(navitiaRoutes)
	if err != nil {
		occupanciesLoadingErrors.Inc()
        return nil, err
	}

	sens := 0
	if direction == "backward" { sens = 1}
	routeSchedules := LoadRouteSchedulesData(startIndex, navitiaRoutes, sens, location)
	return routeSchedules, nil
}

func LoadPredictionsData(predictionData *PredictionData, location *time.Location) ([]Prediction) {
	predictions := make([]Prediction, 0)
	for _, predict := range *predictionData {
		predictions = append(predictions, *NewPrediction(predict, location))
	}
	return predictions
}

func LoadPredictions(uri url.URL, token string, connectionTimeout time.Duration, location *time.Location) ([]Prediction, error) {
	//futuredata/getfuturedata?start_time=2021-02-15&end_time=2021-02-16
	begin := time.Now()
	start_date := begin.Format("2006-01-02")
	end_date := begin.AddDate(0,0,1).Format("2006-01-02")
	callUrl := fmt.Sprintf("%s/futuredata/getfuturedata?start_time=%s&end_time=%s", uri.String(), start_date, end_date)
	header := "Ocp-Apim-Subscription-Key"
	resp, err := GetHttpClient(callUrl, token, header, connectionTimeout)
	if err != nil {
		occupanciesLoadingErrors.Inc()
		return nil, err
	}

	predicts := &PredictionData{}
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

func RefreshVehicleOccupancies(manager *DataManager, predict_url url.URL, predict_token string,	connectionTimeout time.Duration, location *time.Location) error {
	predictions, _ := LoadPredictions(predict_url, predict_token, connectionTimeout, location)

	// create vehicleOccupancy with "Charge" using StopPoints and Courses in the manager for each element in Prediction
	occupanciesWithCharge := make(map[int]VehicleOccupancy, 0)
	var course = ""
	var vehicleJourneyId = ""
	for _, predict := range predictions {
		if course != predict.Course {course = predict.Course}
		if predict.Order == 0 {
			firstTime, err := manager.GetCourseFirstTime(predict)
			if err != nil {
				continue
			}
			// Schedule datettime = predict.Date + firstTime
			dateTime := addDateAndTime(predict.Date, firstTime)
			vehicleJourneyId = manager.GetVehicleJourneyId(predict, dateTime)
		}

		if len(vehicleJourneyId) > 0 {
			// Fetch StopId from manager corresponding to predict.StopName and predict.Sens
			stopId := manager.GetStopId(predict.StopName, predict.Sens)
			rs := manager.GetRouteSchedule(vehicleJourneyId, stopId, predict.Sens)
			if rs != nil {
				vo, err := NewVehicleOccupancy(*rs, calculateOccupancy(predict.Charge))

				if err != nil {
					continue
				}
				occupanciesWithCharge[vo.Id] = *vo
			}
		}
	}

	manager.UpdateVehicleOccupancies(occupanciesWithCharge)
	return nil
}

func LoadAllForVehicleOccupancies(manager *DataManager, files_uri, navitia_url, predict_url url.URL, navitia_token, predict_token string, 
	connectionTimeout time.Duration, location *time.Location) error {
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

	startIndex := 1
	routeSchedules, err := LoadRoutes(startIndex, navitia_url, navitia_token, "forward", connectionTimeout, location)
	if err != nil {
		return err
	}
	startIndex = len(routeSchedules) + 1
	backward, err := LoadRoutes(startIndex, navitia_url, navitia_token, "backward", connectionTimeout, location)
	if err != nil {
		return err
	}

	// Concat two arrays
	for i, _ := range backward {
		routeSchedules = append(routeSchedules, backward[i])
	}
	manager.InitRouteSchedule(routeSchedules)

	// Load predictions for external service and update VehicleOccupancies with charge
	RefreshVehicleOccupancies(manager, predict_url, predict_token, connectionTimeout, location)
	return nil
}
