package lines

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const LinesCsvFileName string = "lignes.lig"
const linesCsvNumOfFields int = 3

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates Line objects based on a line read from a CSV
-------------------------------------------------------------------------------------- */
type Line struct {
	InternalId string // Internal ID of the line
	ExternalId string // External ID of the line, namely the same ID used by Navitia)
}

func newLine(record []string) (*Line, error) {
	if len(record) < linesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in Line record")
	}

	return &Line{
		InternalId: record[0],
		ExternalId: record[1],
	}, nil
}

type lineCsvLineConsumer struct {
	lines map[string]Line
}

func makeLineCsvLineConsumer() *lineCsvLineConsumer {
	return &lineCsvLineConsumer{
		lines: make(map[string]Line),
	}
}

func (c *lineCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	line, err := newLine(csvLine)
	if err != nil {
		return err
	}
	c.lines[line.InternalId] = *line
	return nil
}

func (c *lineCsvLineConsumer) Terminate() {
}

func LoadLines(uri url.URL, connectionTimeout time.Duration) (map[string]Line, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, LinesCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      linesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	lineConsumer := makeLineCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, lineConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return lineConsumer.lines, nil
}
