package sytralrt

import (
	"net/url"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

type SytralRTContext struct {
	connector *connectors.Connector
}

func (d *SytralRTContext) InitContext(externalURI url.URL, loadExternalRefresh,
	connectionTimeout time.Duration) {

	if len(externalURI.String()) == 0 || loadExternalRefresh.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}
	d.connector = connectors.NewConnector(externalURI, externalURI, "", loadExternalRefresh, connectionTimeout)
}

func (d *SytralRTContext) RefreshDeparturesLoop(context *departures.DeparturesContext) {
	context.SetPackageName(reflect.TypeOf(SytralRTContext{}).PkgPath())
	context.RefreshTime = d.connector.GetRefreshTime()

	// Wait 10 seconds before reloading external departures informations
	time.Sleep(10 * time.Second)
	for {
		err := RefreshDepartures(d, context)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		} else {
			logrus.Debug("Departures data updated")
		}
		time.Sleep(d.connector.GetRefreshTime())
	}
}

func RefreshDepartures(sytralContext *SytralRTContext, context *departures.DeparturesContext) error {
	begin := time.Now()

	file, err := utils.GetFile(sytralContext.connector.GetFilesUri(), sytralContext.connector.GetConnectionTimeout())
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return err
	}

	departureConsumer := departures.MakeDepartureLineConsumer()
	if err = utils.LoadData(file, departureConsumer); err != nil {
		departures.DepartureLoadingErrors.Inc()
		return err
	}
	context.UpdateDepartures(departureConsumer.Data)
	departures.DepartureLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}
