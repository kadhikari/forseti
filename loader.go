package sytralrt

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
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
	},
	)

	departureLoadingErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "sytralrt",
		Subsystem: "departures",
		Name:      "loading_errors",
		Help:      "current number of http request being served",
	},
	)
)

func init() {
	prometheus.MustRegister(departureLoadingDuration)
	prometheus.MustRegister(departureLoadingErrors)
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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

func LoadData(file io.Reader) (map[string][]Departure, error) {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return nil, err
	}

	data := make(map[string][]Departure)

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = 8

	// Loop through lines & turn into object
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		departure, err := NewDeparture(line, location)
		if err != nil {
			return nil, err
		}
		data[departure.Stop] = append(data[departure.Stop], departure)
	}

	//sort the departure
	for _, v := range data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
	return data, nil
}

func RefreshDepartures(manager *DataManager, uri url.URL) error {
	begin := time.Now()
	file, err := getFile(uri)
	if err != nil {
		departureLoadingErrors.Inc()
		return err
	}
	d, err := LoadData(file)
	if err != nil {
		departureLoadingErrors.Inc()
		return err
	}
	manager.UpdateDepartures(d)
	departureLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}
