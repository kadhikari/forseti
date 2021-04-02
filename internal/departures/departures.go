package departures

import (
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/CanalTP/forseti/internal/utils"
)

func RefreshDeparturesLoop(context *DeparturesContext,
	departuresURI url.URL,
	departuresRefresh, connectionTimeout time.Duration) {
	if len(departuresURI.String()) == 0 || departuresRefresh.Seconds() <= 0 {
		logrus.Debug("Departures data refreshing is disabled")
		return
	}
	for {
		err := RefreshDepartures(context, departuresURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading departures data: ", err)
		}
		logrus.Debug("Departures data updated")
		time.Sleep(departuresRefresh)
	}
}

func RefreshDepartures(context *DeparturesContext, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := utils.GetFile(uri, connectionTimeout)
	if err != nil {
		DepartureLoadingErrors.Inc()
		return err
	}

	departureConsumer := makeDepartureLineConsumer()
	if err = utils.LoadData(file, departureConsumer); err != nil {
		DepartureLoadingErrors.Inc()
		return err
	}
	context.UpdateDepartures(departureConsumer.data)
	DepartureLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}
