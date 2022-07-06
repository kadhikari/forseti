package stoppoints

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const StopPointsCsvFileName string = "arrets.art"
const stopPointsCsvNumOfFields int = 4

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates StopPoint objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type StopPoint struct {
	InternalId string // Internal ID of the stop point
	ExternalId string // External ID of the stop point, namely the same ID used by Navitia)
}

func newStopPoint(record []string) (*StopPoint, error) {
	if len(record) < stopPointsCsvNumOfFields {
		return nil, fmt.Errorf("missing field in StopPoint record")
	}

	return &StopPoint{
		InternalId: record[0],
		ExternalId: record[2],
	}, nil
}

type stopPointCsvLineConsumer struct {
	stopPoints map[string]StopPoint
}

func makeStopPointCsvLineConsumer() *stopPointCsvLineConsumer {
	return &stopPointCsvLineConsumer{
		stopPoints: make(map[string]StopPoint),
	}
}

func (c *stopPointCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	stopPoint, err := newStopPoint(csvLine)
	if err != nil {
		return err
	}
	c.stopPoints[stopPoint.ExternalId] = *stopPoint
	return nil
}

func (c *stopPointCsvLineConsumer) Terminate() {
}

func LoadStopPoints(uri url.URL, connectionTimeout time.Duration) (map[string]StopPoint, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, StopPointsCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      stopPointsCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	stopPointsConsumer := makeStopPointCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, stopPointsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return stopPointsConsumer.stopPoints, nil
}
