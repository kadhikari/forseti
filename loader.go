package sytralrt

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/pkg/sftp"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ssh"
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
)

func init() {
	prometheus.MustRegister(departureLoadingDuration)
	prometheus.MustRegister(departureLoadingErrors)
	prometheus.MustRegister(parkingsLoadingDuration)
	prometheus.MustRegister(parkingsLoadingErrors)
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

func LoadData(file io.Reader, lineConsumer lineConsumer) error {

	return LoadDataWithOptions(file, lineConsumer, LoadDataOptions{
		delimiter:     ';',
		nbFields:      8,
		skipFirstLine: false,
	})
}

func LoadDataWithOptions(file io.Reader, lineConsumer lineConsumer, options LoadDataOptions) error {

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

		if err := lineConsumer.consume(line, location); err != nil {
			return err
		}
	}

	lineConsumer.terminate()
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
