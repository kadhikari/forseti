package dbinternallinks

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const DbInternalLinksCsvFileName string = "arrets_chn.lar"
const dbInternalLinksCsvNumOfFields int = 4

/* -----------------------------------------------------------------------------------
// Structure and Consumer to creates DbInternalLink objects based on a line read from a CSV
----------------------------------------------------------------------------------- */
type DbInternalLink struct {
	Id             string
	StopPointId    string
	RouteId        string
	StopPointOrder int
}

func newDbInternalLink(record []string) (*DbInternalLink, error) {
	if len(record) < dbInternalLinksCsvNumOfFields {
		return nil, fmt.Errorf("missing field in DbInternalLink record")
	}

	stopPointOrder, err := strconv.Atoi(record[3])
	if err != nil {
		return nil, err
	}

	return &DbInternalLink{
		Id:             record[0],
		StopPointId:    record[1],
		RouteId:        record[2],
		StopPointOrder: stopPointOrder,
	}, nil
}

type dbInternalLinkCsvLineConsumer struct {
	dbInternalLinks map[string]DbInternalLink
}

func makeDbInternalLinkCsvLineConsumer() *dbInternalLinkCsvLineConsumer {
	return &dbInternalLinkCsvLineConsumer{
		dbInternalLinks: make(map[string]DbInternalLink),
	}
}

func (c *dbInternalLinkCsvLineConsumer) Consume(csvLine []string, _ *time.Location) error {
	dbInternalLink, err := newDbInternalLink(csvLine)
	if err != nil {
		return err
	}
	c.dbInternalLinks[dbInternalLink.Id] = *dbInternalLink
	return nil
}

func (c *dbInternalLinkCsvLineConsumer) Terminate() {
}

func LoadDbInternalLinks(uri url.URL, connectionTimeout time.Duration) (map[string]DbInternalLink, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, DbInternalLinksCsvFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      dbInternalLinksCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	dbInternalLinksConsumer := makeDbInternalLinkCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, dbInternalLinksConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}
	return dbInternalLinksConsumer.dbInternalLinks, nil
}
