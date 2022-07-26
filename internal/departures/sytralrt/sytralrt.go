package sytralrt

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

type SytralRTContext struct {
	connector *connectors.Connector
}

func (d *SytralRTContext) InitContext(
	externalURI url.URL,
	loadExternalRefresh,
	connectionTimeout time.Duration,
) {
	const unusedDuration time.Duration = time.Duration(-1)
	if len(externalURI.String()) == 0 || loadExternalRefresh.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}
	d.connector = connectors.NewConnector(
		externalURI,
		externalURI,
		"",
		loadExternalRefresh,
		unusedDuration,
		connectionTimeout,
	)
}

func (d *SytralRTContext) RefreshDeparturesLoop(context *departures.DeparturesContext) {
	context.SetPackageName(reflect.TypeOf(SytralRTContext{}).PkgPath())
	context.SetFilesRefeshTime(d.connector.GetFilesRefreshTime())

	// Wait 10 seconds before reloading external departures informations
	time.Sleep(10 * time.Second)
	for {
		err := RefreshDepartures(d, context)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		} else {
			logrus.Debug("Departures data updated")
		}
		time.Sleep(d.connector.GetFilesRefreshTime())
	}
}

func RefreshDepartures(sytralContext *SytralRTContext, context *departures.DeparturesContext) error {
	begin := time.Now()

	file, err := utils.GetFile(sytralContext.connector.GetFilesUri(), sytralContext.connector.GetConnectionTimeout())
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return err
	}

	departureConsumer := makeDepartureLineConsumer()
	if err = utils.LoadData(file, departureConsumer); err != nil {
		departures.DepartureLoadingErrors.Inc()
		return err
	}
	context.UpdateDepartures(departureConsumer.Data)
	departures.DepartureLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

// departureLineConsumer constructs a departure from a slice of strings
type departureLineConsumer struct {
	Data map[string][]departures.Departure
}

func newDeparture(record []string, location *time.Location) (departures.Departure, error) {
	if len(record) < 7 {
		return departures.Departure{}, fmt.Errorf("missing field in record")
	}
	dt, err := time.ParseInLocation("2006-01-02 15:04:05", record[5], location)
	if err != nil {
		return departures.Departure{}, err
	}
	var directionType departures.DirectionType
	if len(record) >= 10 {
		directionType = departures.ParseDirectionType(record[9])
	}

	return departures.Departure{
		Stop:          record[0],
		Line:          record[1],
		Type:          record[4],
		Datetime:      dt,
		Direction:     record[6],
		DirectionName: record[2],
		DirectionType: directionType,
	}, nil
}

func makeDepartureLineConsumer() *departureLineConsumer {
	return &departureLineConsumer{make(map[string][]departures.Departure)}
}

func (p *departureLineConsumer) Consume(line []string, loc *time.Location) error {

	departure, err := newDeparture(line, loc)
	if err != nil {
		return err
	}

	p.Data[departure.Stop] = append(p.Data[departure.Stop], departure)
	return nil
}

func (p *departureLineConsumer) Terminate() {
	//sort the departures
	for _, v := range p.Data {
		sort.Slice(v, func(i, j int) bool {
			return v[i].Datetime.Before(v[j].Datetime)
		})
	}
}
