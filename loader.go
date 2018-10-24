package main

import (
	"bytes"
	"encoding/csv"
	"io"
	"sort"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpFileConfig struct {
	Host     string
	User     string
	Password string
	File     string
}

func getFileWithSftp(config SftpFileConfig) (io.Reader, error) {

	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClient, err := ssh.Dial("tcp", config.Host, sshConfig)
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

	file, err := client.Open(config.File)
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

func LoadData(file io.Reader) map[string][]Departure {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}

	data := make(map[string][]Departure)

	// Read File
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = 8

	// Loop through lines & turn into object
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		dt, err := time.ParseInLocation("2006-01-02 15:04:05", line[5], location) // aaaa-mm-jjhh:mi:ss
		if err != nil {
			panic(err)
		}

		departure := Departure{
			Stop:          line[0],
			Line:          line[1],
			Type:          line[4],
			Datetime:      dt,
			Direction:     line[6],
			DirectionName: line[2],
		}
		data[departure.Stop] = append(data[departure.Stop], departure)
	}

	//sort the departure
	for _, v := range data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
	return data
}
