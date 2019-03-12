package sytralrt

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

	"github.com/pkg/sftp"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/charmap"
)

var (
	departureLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "sytralrt",
		Subsystem: "departures",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	departureLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "sytralrt",
		Subsystem: "departures",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})

	parkingsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "sytralrt",
		Subsystem: "parkings",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	parkingsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "sytralrt",
		Subsystem: "parkings",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	})

	equipmentsLoadingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "sytralrt",
		Subsystem: "equipments",
		Name:      "load_durations_seconds",
		Help:      "http request latency distributions.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 1.5, 15),
	})

	equipmentsLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "sytralrt",
		Subsystem: "equipments",
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
}

func getFile(uri url.URL) (io.Reader, error) {
	if uri.Scheme == "sftp" {
		return getFileWithSftp(uri)
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

func getFileWithSftp(uri url.URL) (io.Reader, error) {
	password, _ := uri.User.Password()
	sshConfig := &ssh.ClientConfig{
		User: uri.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
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
		nbFields:      8,
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

func LoadXmlData(file io.Reader, lineConsumer LineConsumer) error {

	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return err
	}

	XMLdata, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(bytes.NewReader(XMLdata))
	decoder.CharsetReader = getCharsetReader

	var root Root
	err = decoder.Decode(&root)
	if err != nil {
		return err
	}

	// for each root.Data.Lines.Stations create an object Equipment
	for _, l := range root.Data.Lines {
		for _, s := range l.Stations {
			for _, e := range s.Equipments {
				line := []string{e.ID, e.Name, e.Type, e.Cause, e.Effect, e.Start, e.End, e.Hour}
				if err := lineConsumer.Consume(line, location); err != nil {
					return err
				}
			}
		}
	}

	lineConsumer.Terminate()
	return nil
}

func RefreshDepartures(manager *DataManager, uri url.URL) error {
	begin := time.Now()
	file, err := getFile(uri)
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

func RefreshParkings(manager *DataManager, uri url.URL) error {
	begin := time.Now()
	file, err := getFile(uri)
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

func RefreshEquipments(manager *DataManager, uri url.URL) error {
	begin := time.Now()
	file, err := getFile(uri)
	if err != nil {
		equipmentsLoadingErrors.Inc()
		return err
	}

	equipmentConsumer := makeEquipmentLineConsumer()

	err = LoadXmlData(file, equipmentConsumer)
	if err != nil {
		equipmentsLoadingErrors.Inc()
		return err
	}
	manager.UpdateEquipments(equipmentConsumer.equipments)
	equipmentsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}
