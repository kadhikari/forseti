package utils

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/hove-io/forseti/internal/data"
)

var location = "Europe/Paris"

func GetFile(uri url.URL, connectionTimeout time.Duration) (io.Reader, error) {
	if uri.Scheme == "sftp" {
		return GetFileWithSftp(uri, connectionTimeout)
	} else if uri.Scheme == "file" {
		return GetFileWithFS(uri)
	} else {
		return nil, fmt.Errorf("Unsupported protocols %s", uri.Scheme)
	}

}

func GetFileWithFS(uri url.URL) (io.Reader, error) {
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

func GetFileWithSftp(uri url.URL, connectionTimeout time.Duration) (io.Reader, error) {
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
	SkipFirstLine bool
	Delimiter     rune
	NbFields      int
}

func LoadData(file io.Reader, lineConsumer data.LineConsumer) error {

	return LoadDataWithOptions(file, lineConsumer, LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0, // do not check record size in csv.reader
		SkipFirstLine: false,
	})
}

func LoadDataWithOptions(file io.Reader, lineConsumer data.LineConsumer, options LoadDataOptions) error {

	location, err := time.LoadLocation(location)
	if err != nil {
		return err
	}

	reader := csv.NewReader(file)
	reader.Comma = options.Delimiter
	reader.FieldsPerRecord = options.NbFields

	// Loop through lines & turn into object
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if options.SkipFirstLine {
			options.SkipFirstLine = false
			continue
		}

		if err := lineConsumer.Consume(line, location); err != nil {
			return err
		}
	}

	lineConsumer.Terminate()
	return nil
}
