package connectors

import (
	"net/url"
	"sync"
	"time"
)

type ConnectorType string

const (
	Connector_GRFS_RT ConnectorType = "gtfsrt"
	Connector_ODITI   ConnectorType = "oditi"
	Connector_FLUCTUO ConnectorType = "fluctuo"
	Connector_CITIZ   ConnectorType = "citiz"
)

type Connector struct {
	filesUri          url.URL
	url               url.URL
	token             string
	header            string
	refreshTime       time.Duration
	connectionTimeout time.Duration
	mutex             sync.Mutex
}

func (d *Connector) GetFilesUri() url.URL {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.filesUri
}

func (d *Connector) GetUrl() url.URL {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.url
}

func (d *Connector) GetToken() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.token
}

func (d *Connector) SetToken(token string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.token = token
}

func (d *Connector) GetHeader() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.header
}

func (d *Connector) GetConnectionTimeout() time.Duration {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.connectionTimeout
}

func (d *Connector) GetRefreshTime() time.Duration {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.refreshTime
}

func NewConnector(filesURI, url url.URL, token string, refresh,
	connectionTimeout time.Duration) *Connector {
	return &Connector{
		filesUri:          filesURI,
		url:               url,
		token:             token,
		refreshTime:       refresh,
		connectionTimeout: connectionTimeout,
	}
}
