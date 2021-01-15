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

	location, err := time.LoadLocation("Europe/Paris")
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

	location, err := time.LoadLocation("Europe/Paris")
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
	data.Set("query", "query($id: Int!) {area(id: $id) {vehicles{public_id: publicId, provider{name}, id, type, attributes ,latitude: lat, longitude: lng, propulsion, battery, deeplink } } }")
	data.Set("variables", `{"id": 6}`)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1?access_token=%s", siteHost, token), bytes.NewBufferString(data.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded; param=value")
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func LoadFreeFloatingData (resp *http.Response) ([]FreeFloating, error) {
	// Read response body in json
	fmt.Println("************** Reading response body in json  ***********")
	data := &Data{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(data)
	if err != nil {
        return nil, err
	}

	freeFloatings := make([]FreeFloating, 0)
	for i := 0; i < len(data.Data.Area.Vehicles); i++ {
		ff, err := NewFreeFloating(data.Data.Area.Vehicles[i])
		if err != nil {
			return nil, err
		}
		freeFloatings = append(freeFloatings, *ff)
	}
	return freeFloatings, nil
}

func RefreshFreeFloatings(manager *DataManager, uri url.URL, token string, connectionTimeout time.Duration) error {
	begin := time.Now()
	fmt.Println("*** RefreshFreeFloatings ***")
	resp, err := CallHttpClient(uri.String(), token)

	if err != nil {
		freeFloatingsLoadingErrors.Inc()
		return err
	}

	freeFloatings, err := LoadFreeFloatingData(resp)
	if err != nil {
		freeFloatingsLoadingErrors.Inc()
		return err
	}
	fmt.Println("freeFloatings size: ", len(freeFloatings))

	manager.UpdateFreeFloating(freeFloatings)
	freeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}
