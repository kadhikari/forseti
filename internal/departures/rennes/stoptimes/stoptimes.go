package stoptimes

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

type CourseId string

const StopTimesFileName string = "horaires.hor"
const stopTimesCsvNumOfFields int = 4

/* ---------------------------------------------------------------------------------------
// Structure and Consumer to creates StopTime objects based on a line read from a CSV
--------------------------------------------------------------------------------------- */
type StopTime struct {
	Id               string
	Time             time.Time
	RouteStopPointId string
	CourseId         CourseId
}

func newStopTime(record []string, loc *time.Location) (*StopTime, error) {
	if len(record) < stopTimesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in StopTime record")
	}

	stopTimeRecord := record[1]

	const timeLayout string = "15:04:05"
	stopTime, err := time.ParseInLocation(timeLayout, stopTimeRecord, loc)
	if err != nil {
		return nil,
			fmt.Errorf(
				"the stop time of the StopTime cannot be parsed from the record '%s'",
				stopTimeRecord,
			)
	}

	return &StopTime{
		Id:               record[0],
		Time:             stopTime,
		RouteStopPointId: record[2],
		CourseId:         CourseId(record[3]),
	}, nil
}

func CombineDateAndTimeInLocation(
	datePart *time.Time,
	timePart *time.Time,
	location *time.Location,
) (time.Time, error) {
	const DATE_LAYOUT string = "2006-01-02"
	const TIME_LAYOUT string = "15:04:05.999999999"
	const DATETIME_LAYOUT string = DATE_LAYOUT + "T" + TIME_LAYOUT
	datePartStr := datePart.Format(DATE_LAYOUT)
	timePartStr := timePart.Format(TIME_LAYOUT)
	datetimeStr := datePartStr + "T" + timePartStr

	return time.ParseInLocation(DATETIME_LAYOUT, datetimeStr, location)
}

func (s *StopTime) computeActualStopTime(
	dailyServiceStartTime *time.Time,
) time.Time {
	actualStopTime, _ := CombineDateAndTimeInLocation(
		dailyServiceStartTime,
		&s.Time,
		dailyServiceStartTime.Location(),
	)
	if actualStopTime.Before(*dailyServiceStartTime) {
		actualStopTime = actualStopTime.AddDate(
			0, /* year */
			0, /* month */
			1, /* day */
		)
	}
	return actualStopTime
}

type stopTimeCsvLineConsumer struct {
	stopTimes map[string]StopTime
}

func makeStopTimeCsvLineConsumer() *stopTimeCsvLineConsumer {
	return &stopTimeCsvLineConsumer{
		stopTimes: make(map[string]StopTime),
	}
}

func (c *stopTimeCsvLineConsumer) Consume(csvLine []string, loc *time.Location) error {
	stopTime, err := newStopTime(csvLine, loc)
	if err != nil {
		return err
	}
	c.stopTimes[stopTime.Id] = *stopTime
	return nil
}

func (c *stopTimeCsvLineConsumer) Terminate() {
}

func LoadStopTimes(
	uri url.URL,
	connectionTimeout time.Duration,
	dailyServiceStartTime *time.Time,
) (map[string]StopTime, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, StopTimesFileName)
	file, err := utils.GetFile(uri, connectionTimeout)

	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	return LoadStopTimesUsingReader(file, connectionTimeout, dailyServiceStartTime)

}

func LoadStopTimesUsingReader(
	reader io.Reader,
	connectionTimeout time.Duration,
	dailyServiceStartTime *time.Time,
) (map[string]StopTime, error) {

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      stopTimesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	stopTimesConsumer := makeStopTimeCsvLineConsumer()
	err := utils.LoadDataWithOptions(reader, stopTimesConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}

	// Update the data of each stop time
	for id, stopTime := range stopTimesConsumer.stopTimes {
		actualStopTime := stopTime.computeActualStopTime(
			dailyServiceStartTime,
		)
		stopTime.Time = actualStopTime
		stopTimesConsumer.stopTimes[id] = stopTime
	}
	return stopTimesConsumer.stopTimes, nil
}
