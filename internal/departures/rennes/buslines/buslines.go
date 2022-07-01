package buslines

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const BusLinesCsvFileName string = "lignes.lig"
const busLinesCsvNumOfFields int = 3

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates BusLine objects based on a line read from a CSV
-------------------------------------------------------------------------------------- */
type BusLine struct {
	InternalId string // Internal ID of the line
	ExternalId string // External ID of the line, namely the same ID used by Navitia)
}

func newBusLine(record []string) (*BusLine, error) {
	if len(record) < busLinesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in Line record")
	}

	return &BusLine{
		InternalId: record[0],
		ExternalId: record[1],
	}, nil
}

type busLineCsvLineConsumer struct {
	busLines map[string]BusLine
}

func makeBusLineCsvLineConsumer() *busLineCsvLineConsumer {
	return &busLineCsvLineConsumer{
		busLines: make(map[string]BusLine),
	}
}

func (c *busLineCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	busLine, err := newBusLine(csvLine)
	if err != nil {
		return err
	}
	c.busLines[busLine.InternalId] = *busLine
	return nil
}

func (c *busLineCsvLineConsumer) Terminate() {
}

func LoadBusLines(uri url.URL, connectionTimeout time.Duration) (map[string]BusLine, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, BusLinesCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      busLinesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	busLineConsumer := makeBusLineCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, busLineConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return busLineConsumer.busLines, nil
}
