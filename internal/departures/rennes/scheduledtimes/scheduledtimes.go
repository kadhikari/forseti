package scheduledtimes

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const ScheduledTimesFileName string = "horaires.hor"
const scheduledTimesCsvNumOfFields int = 4

/* ---------------------------------------------------------------------------------------
// Structure and Consumer to creates ScheduledTime objects based on a line read from a CSV
--------------------------------------------------------------------------------------- */
type ScheduledTime struct {
	Id               string
	Time             time.Time
	DbInternalLinkId string
}

func newScheduledTime(record []string, loc *time.Location) (*ScheduledTime, error) {
	if len(record) < scheduledTimesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in ScheduledTime record")
	}

	scheduledTimeRecord := record[1]

	const timeLayout string = "15:04:05"
	scheduledTime, err := time.ParseInLocation(timeLayout, scheduledTimeRecord, loc)
	if err != nil {
		return nil,
			fmt.Errorf(
				"the scheduled time of the ScheduledTime cannot be parsed from the record '%s'",
				scheduledTimeRecord,
			)
	}

	return &ScheduledTime{
		Id:               record[0],
		Time:             scheduledTime,
		DbInternalLinkId: record[2],
	}, nil
}

type scheduledTimeCsvLineConsumer struct {
	scheduledTimes map[string]ScheduledTime
}

func makeScheduledTimeCsvLineConsumer() *scheduledTimeCsvLineConsumer {
	return &scheduledTimeCsvLineConsumer{
		scheduledTimes: make(map[string]ScheduledTime),
	}
}

func (c *scheduledTimeCsvLineConsumer) Consume(csvLine []string, loc *time.Location) error {
	scheduledTime, err := newScheduledTime(csvLine, loc)
	if err != nil {
		return err
	}
	c.scheduledTimes[scheduledTime.Id] = *scheduledTime
	return nil
}

func (c *scheduledTimeCsvLineConsumer) Terminate() {
}

func LoadScheduledTimes(
	uri url.URL,
	connectionTimeout time.Duration,
	processingDate *time.Time,
) (map[string]ScheduledTime, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, ScheduledTimesFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      scheduledTimesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	scheduledTimesConsumer := makeScheduledTimeCsvLineConsumer()
	err = utils.LoadDataWithOptions(file, scheduledTimesConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}

	// Update the data of each stop time
	for id, scheduledTime := range scheduledTimesConsumer.scheduledTimes {
		t := time.Date(
			processingDate.Year(),
			processingDate.Month(),
			processingDate.Day(),
			scheduledTime.Time.Hour(),
			scheduledTime.Time.Minute(),
			scheduledTime.Time.Second(),
			scheduledTime.Time.Nanosecond(),
			processingDate.Location(),
		)
		scheduledTime.Time = t
		scheduledTimesConsumer.scheduledTimes[id] = scheduledTime
	}
	return scheduledTimesConsumer.scheduledTimes, nil
}
