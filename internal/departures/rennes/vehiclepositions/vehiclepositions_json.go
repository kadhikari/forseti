package vehiclepositions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
	// forseti_vp "github.com/hove-io/forseti/internal/vehiclepositions"
)

const vehiclePositionsFileName string = "vehicules"

func LoadVehiclePositions(
	uri url.URL,
	header string,
	token string,
	connectionTimeout time.Duration,
) ([]VehiclePosition, error) {
	uri.Path = fmt.Sprintf("%s/%s", uri.Path, vehiclePositionsFileName)
	logrus.Infof("send a GET request %s", uri.String())
	response, err := utils.GetHttpClient(uri.String(), token, header, connectionTimeout)

	if err != nil {
		// forseti_vp.VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}

	inputBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		// forseti_vp.VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}

	return parseVehiclePositionsFromFileBytes(inputBytes)
}

func parseVehiclePositionsFromFileBytes(
	fileBytes []byte,
) ([]VehiclePosition, error) {

	csvString, err := extractCsvStringFromJson(fileBytes)
	if err != nil {
		// forseti_vp.VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}

	reader := bytes.NewReader([]byte(csvString))
	var vehiclesPositions []VehiclePosition
	vehiclesPositions, err = loadVehiclePositionsUsingReader(reader)
	if err != nil {
		// forseti_vp.VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}
	return vehiclesPositions, nil
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
	return propertyFile, nil
}
