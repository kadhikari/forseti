package destinations

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const DestionationsCsvFileName string = "destinations.dst"
const destionationsCsvNumOfFields int = 2

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates Destination objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type Destination struct {
	Id   string
	Name string
}

func newDestination(record []string) (*Destination, error) {
	if len(record) < destionationsCsvNumOfFields {
		return nil, fmt.Errorf("missing field in Destination record")
	}

	return &Destination{
		Id:   record[0],
		Name: record[1],
	}, nil
}

type destinationCsvLineConsumer struct {
	destinations map[string]Destination
}

func makeDestinationCsvLineConsumer() *destinationCsvLineConsumer {
	return &destinationCsvLineConsumer{
		destinations: make(map[string]Destination),
	}
}

func (c *destinationCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	destination, err := newDestination(csvLine)
	if err != nil {
		return err
	}
	c.destinations[destination.Id] = *destination
	return nil
}

func (c *destinationCsvLineConsumer) Terminate() {
}

func LoadDestinations(uri url.URL, connectionTimeout time.Duration) (map[string]Destination, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, DestionationsCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      destionationsCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	destinationsConsumer := makeDestinationCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, destinationsConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return destinationsConsumer.destinations, nil
}
