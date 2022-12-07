package estimatedstoptimes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/departures/rennes/stoptimes"
	"github.com/hove-io/forseti/internal/utils"
)

// type aliases
type CourseId = stoptimes.CourseId

const estimatedStopTimesFileName string = "horaires"
const estimatedStopTimesCsvNumOfFields int = 5

type Fiability int

const (
	// Theorique
	Fiability_T Fiability = 0
	// Fiable
	Fiability_F Fiability = 1
)

func parseFiabilityFromString(fiabilityStr string) (Fiability, error) {
	var fiabilitiesMap = map[string]Fiability{
		"T": Fiability_T,
		"":  Fiability_T, /* Empty value is considered as a *theoretical* */
		"F": Fiability_F,
	}

	parsedFiability, ok := fiabilitiesMap[fiabilityStr]
	var err error = nil
	if !ok {
		parsedFiability = Fiability_T
		err = fmt.Errorf("estimated stop time fiability cannot be parsed, unexpected value %s", fiabilityStr)
	}
	return parsedFiability, err
}

/* -------------------------------------------------------------------------------------------
// Structure and Consumer to creates EstimatedStopTime objects based on a line read from a CSV
---------------------------------------------------------------------------------------------- */
type EstimatedStopTime struct {
	Id               string
	Time             time.Time
	RouteStopPointId string
	CourseId         CourseId
	Fiability        Fiability
}

func newEstimatedStopTime(record []string, loc *time.Location) (*EstimatedStopTime, error) {
	if len(record) != estimatedStopTimesCsvNumOfFields {
		return nil, fmt.Errorf("missing field in EstimatedStopTime record")
	}

	estimatedStopTimeRecord := record[1]

	const timeLayout string = "15:04:05"
	estimatedStopTime, err := time.ParseInLocation(timeLayout, estimatedStopTimeRecord, loc)
	if err != nil {
		return nil,
			fmt.Errorf(
				"the stop time of the EstimatedStopTime cannot be parsed from the record '%s'",
				estimatedStopTimeRecord,
			)
	}

	fiabilityStr := record[4]
	fiability, err := parseFiabilityFromString(fiabilityStr)
	if err != nil {
		return nil,
			fmt.Errorf(
				"the stop time of the EstimatedStopTime cannot be parsed from the record '%s': %s",
				fiabilityStr,
				err.Error(),
			)
	}

	return &EstimatedStopTime{
		Id:               record[0],
		Time:             estimatedStopTime,
		RouteStopPointId: record[2],
		CourseId:         CourseId(record[3]),
		Fiability:        fiability,
	}, nil
}

type estimatedStopTimeCsvLineConsumer struct {
	estimatedStopTimes map[string]EstimatedStopTime
}

func makeEstimatedStopTimeCsvLineConsumer() *estimatedStopTimeCsvLineConsumer {
	return &estimatedStopTimeCsvLineConsumer{
		estimatedStopTimes: make(map[string]EstimatedStopTime),
	}
}

func (c *estimatedStopTimeCsvLineConsumer) Consume(csvLine []string, loc *time.Location) error {
	estimatedStopTimes, err := newEstimatedStopTime(csvLine, loc)
	if err != nil {
		return err
	}
	c.estimatedStopTimes[estimatedStopTimes.Id] = *estimatedStopTimes
	return nil
}

func (c *estimatedStopTimeCsvLineConsumer) Terminate() {
}

func LoadEstimatedStopTimes(
	uri url.URL,
	header string,
	token string,
	connectionTimeout time.Duration,
	localDailyServiceStartTime *time.Time,
) (map[string]EstimatedStopTime, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, estimatedStopTimesFileName)
	response, err := utils.GetHttpClient(uri.String(), token, header, connectionTimeout)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	inputBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}
	return parseEstimatedStopTimesFromFile(inputBytes, localDailyServiceStartTime)
}

func parseEstimatedStopTimesFromFile(
	fileBytes []byte,
	dailyServiceStartTime *time.Time,
) (map[string]EstimatedStopTime, error) {

	propertyFile, err := extractCsvStringFromJson(fileBytes)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	reader := bytes.NewReader([]byte(propertyFile))
	var estimatedStopTimes map[string]EstimatedStopTime
	estimatedStopTimes, err = loadEstimatedStopTimesUsingReader(reader, dailyServiceStartTime)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}
	return estimatedStopTimes, nil
}

func extractCsvStringFromJson(b []byte) (string, error) {

	var err error
	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		return "", err
	}

	var propertyResponse interface{}
	{
		const PROPERTY_NAME string = "response"
		var m map[string]interface{} = f.(map[string]interface{})
		var ok bool
		if propertyResponse, ok = m[PROPERTY_NAME]; !ok {
			return "", fmt.Errorf("an unexpected error occupred while the reading of the property '%s'", PROPERTY_NAME)
		}
	}

	var propertyData interface{}
	{
		const PROPERTY_NAME string = "data"
		var m map[string]interface{} = propertyResponse.(map[string]interface{})
		var ok bool
		if propertyData, ok = m[PROPERTY_NAME]; !ok {
			return "", fmt.Errorf("an unexpected error occupred while the reading of the property '%s'", PROPERTY_NAME)
		}
	}

	var propertyFile string
	{
		const PROPERTY_NAME string = "file"
		var m map[string]interface{} = propertyData.(map[string]interface{})
		var ok bool
		if propertyFile, ok = m[PROPERTY_NAME].(string); !ok {
			return "", fmt.Errorf("an unexpected error occupred while the reading of the property '%s'", PROPERTY_NAME)
		}
	}

	// Substitute the substrings "\n" into end of line characters
	propertyFile = strings.ReplaceAll(propertyFile, "\\n", "\n")
	return propertyFile, nil
}

func loadEstimatedStopTimesUsingReader(
	reader io.Reader,
	dailyServiceStartTime *time.Time,
) (map[string]EstimatedStopTime, error) {

	loadDataOptions := utils.LoadDataOptions{
		Delimiter:     ';',
		NbFields:      estimatedStopTimesCsvNumOfFields,
		SkipFirstLine: true, // First line is a header
	}

	estimatedStopTimesConsumer := makeEstimatedStopTimeCsvLineConsumer()
	err := utils.LoadDataWithOptions(reader, estimatedStopTimesConsumer, loadDataOptions)
	if err != nil {
		return nil, err
	}

	// Update the data of each stop time
	for id, stopTime := range estimatedStopTimesConsumer.estimatedStopTimes {
		actualStopTime := stoptimes.ComputeActualStopTime(
			stopTime.Time,
			dailyServiceStartTime,
		)
		stopTime.Time = actualStopTime
		estimatedStopTimesConsumer.estimatedStopTimes[id] = stopTime
	}
	return estimatedStopTimesConsumer.estimatedStopTimes, nil
}
