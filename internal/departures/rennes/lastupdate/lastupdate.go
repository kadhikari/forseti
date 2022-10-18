package lastupdate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/departures"
	"github.com/hove-io/forseti/internal/utils"
)

const lastUpdateFileName string = "last-update"

const LastUpdateLayout string = "2006-01-02 15:04:05"

func LoadLastUpdate(
	uri url.URL,
	header string,
	token string,
	connectionTimeout time.Duration,
	location *time.Location,
) (*time.Time, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, lastUpdateFileName)
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

	csvString, err := extractCsvStringFromJson(inputBytes)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	lastUpdate, err := parseLastUpdateInLocation(csvString, location)
	if err != nil {
		departures.DepartureLoadingErrors.Inc()
		return nil, err
	}

	return &lastUpdate, nil
}

func parseLastUpdateInLocation(lastUpdateStr string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(LastUpdateLayout, lastUpdateStr, loc)
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

	var propertyLastUpdate string
	{
		const PROPERTY_NAME string = "lastUpdate"
		var m map[string]interface{} = propertyData.(map[string]interface{})
		var ok bool
		if propertyLastUpdate, ok = m[PROPERTY_NAME].(string); !ok {
			return "", fmt.Errorf("an unexpected error occupred while the reading of the property '%s'", PROPERTY_NAME)
		}
	}
	return propertyLastUpdate, nil
}
