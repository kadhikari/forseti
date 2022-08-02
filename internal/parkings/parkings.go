package parkings

import (
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

func RefreshParkingsLoop(context *ParkingsContext,
	parkingsURI url.URL,
	parkingsRefresh, connectionTimeout time.Duration) {
	if len(parkingsURI.String()) == 0 || parkingsRefresh.Seconds() <= 0 {
		logrus.Debug("Parking data refreshing is disabled")
		return
	}
	for {
		err := RefreshParkings(context, parkingsURI, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading parking data: ", err)
		} else {
			logrus.Debug("Parking data updated")
		}
		time.Sleep(parkingsRefresh)
	}
}

func RefreshParkings(context *ParkingsContext, uri url.URL, connectionTimeout time.Duration) error {
	begin := time.Now()
	file, err := utils.GetFile(uri, connectionTimeout)
	if err != nil {
		ParkingsLoadingErrors.Inc()
		return err
	}

	parkingsConsumer := makeParkingLineConsumer()
	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      0,    // We might not have etereogenous lines
		SkipFirstLine: true, // First line is a header
	}
	err = utils.LoadDataWithOptions(file, parkingsConsumer, loadDataOptions)
	if err != nil {
		ParkingsLoadingErrors.Inc()
		return err
	}

	context.UpdateParkings(parkingsConsumer.parkings)
	ParkingsLoadingDuration.Observe(time.Since(begin).Seconds())

	return nil
}
