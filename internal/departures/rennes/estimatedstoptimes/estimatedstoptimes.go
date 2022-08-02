package estimatedstoptimes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/departures/rennes/stoptimes"
	"github.com/hove-io/forseti/internal/utils"
)

const estimatedStopTimesFileName string = "horaires"

func LoadEstimatedStopTimes(
	uri url.URL,
	header string,
	token string,
	connectionTimeout time.Duration,
	localDailyServiceStartTime *time.Time,
) (map[string]stoptimes.StopTime, error) {
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
	return parseEstimatedStopTimesFromFile(inputBytes, connectionTimeout, localDailyServiceStartTime)
}

func parseEstimatedStopTimesFromFile(
	fileBytes []byte,
	connectionTimeout time.Duration,
	dailyServiceStartTime *time.Time,
) (map[string]stoptimes.StopTime, error) {

	propertyFile, err := extractCsvStringFromJson(fileBytes)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	reader := bytes.NewReader([]byte(propertyFile))
	var stopTimes map[string]stoptimes.StopTime
	stopTimes, err = stoptimes.LoadStopTimesUsingReader(reader, connectionTimeout, dailyServiceStartTime)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}
	return stopTimes, nil
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
