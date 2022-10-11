package vehiclepositions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	forseti_utils "github.com/hove-io/forseti/internal/utils"
)

const (
	URL_GET_LAST_LOAD              = "%s/status?"
	URL_GET_VEHICLES_JOURNEYS_LINE = "%s/v1/coverage/%s/lines/%s/vehicle_positions"
	LINE_CODE                      = "line:SAR:0100"
)

// This method call Navitia api with specific url and return a request response
func callNavitia(callUrl string, token string, connectionTimeout time.Duration) (*http.Response, error) {
	resp, err := forseti_utils.GetHttpClient(callUrl, token, "Authorization", connectionTimeout)
	if err != nil {
		VehiclePositionsLoadingErrors.Inc()
		return nil, err
	}
	return resp, nil
}

type NavitiaVehiclePositions struct {
	VehiclePositions []NavitiaVehiclePosition `json:"vehicle_positions"`
}

func (vp *NavitiaVehiclePositions) getNumberOfVehicleJourneyPositions() int {
	numberOfVehicleJourneys := 0
	for _, vehiculePosition := range vp.VehiclePositions {
		numberOfVehicleJourneys += len(vehiculePosition.VehicleJourneyPositions)
	}
	return numberOfVehicleJourneys
}

func (vp *NavitiaVehiclePositions) getVehicleJourneyCode() (string, error) {
	numOfVehiclePositions := len(vp.VehiclePositions)
	if numOfVehiclePositions == 0 {
		return "", fmt.Errorf("no vehicle position has been found in the navita response")
	}
	if numOfVehiclePositions > 1 {
		return "", fmt.Errorf(
			"at most 1 vehicle position is expected in the navitia , get %d",
			numOfVehiclePositions,
		)
	}
	return vp.VehiclePositions[0].getVehicleJourneyCode()
}

type NavitiaVehiclePosition struct {
	Line                    NavitiaLine                     `json:"line"`
	VehicleJourneyPositions []NavitiaVehicleJourneyPosition `json:"vehicle_journey_positions"`
}

func (vp *NavitiaVehiclePosition) getVehicleJourneyCode() (string, error) {
	numOfVehicleJourneyPositions := len(vp.VehicleJourneyPositions)
	if numOfVehicleJourneyPositions == 0 {
		return "", fmt.Errorf("no vehicle journey has been found")
	}
	if numOfVehicleJourneyPositions > 1 {
		return "", fmt.Errorf(
			"at most 1 vehicle journey is expected, get %d",
			numOfVehicleJourneyPositions,
		)
	}
	return vp.VehicleJourneyPositions[0].getVehicleJourneyCode()
}

type NavitiaLine struct {
	Id string `json:"id"`
}

type NavitiaVehicleJourneyPosition struct {
	VehicleJourney NavitiaVehicleJourney `json:"vehicle_journey"`
}

func (vjp *NavitiaVehicleJourneyPosition) getVehicleJourneyCode() (string, error) {
	return vjp.VehicleJourney.getSourceCode()
}

type NavitiaVehicleJourney struct {
	Id    string        `json:"id"`
	Codes []NavitiaCode `json:"codes"`
}

func (vj *NavitiaVehicleJourney) getSourceCode() (string, error) {
	const VALUE_FROM_TYPE string = "source"
	findSourceCodes := make([]string, 0)
	for _, code := range vj.Codes {
		if code.Type == VALUE_FROM_TYPE {
			findSourceCodes = append(findSourceCodes, code.Value)
		}
	}
	numberOfFindSourceCodes := len(findSourceCodes)
	if numberOfFindSourceCodes == 0 {
		return "", fmt.Errorf("no source code has been found")
	} else if numberOfFindSourceCodes > 1 {
		return "", fmt.Errorf(
			"at most 1 vehicle journey is expected, get %d",
			numberOfFindSourceCodes,
		)
	}
	return findSourceCodes[0], nil
}

type NavitiaCode struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// GetVehicleJourney get object vehicle journey from Navitia compared to ODITI vehicle id.
func GetVehicleJourneyCodeFromNavitia(
	token string,
	connectionTimeout time.Duration,
	location *time.Location,
	uri url.URL,
	coverageName string,
	utcCurrentDatetime *time.Time,
) (string, error) {
	// Format the URL og the navitia GET request
	callUrl := fmt.Sprintf(
		URL_GET_VEHICLES_JOURNEYS_LINE,
		uri.String(),
		coverageName,
		LINE_CODE,
	)
	if utcCurrentDatetime != nil {
		// Check the llocation of the current datetime
		{
			currentDatetimeLocation := utcCurrentDatetime.Location()
			if currentDatetimeLocation != time.UTC {
				return "", fmt.Errorf(
					"the current datetime must be localized at UTC: %v",
					currentDatetimeLocation,
				)
			}
		}
		callUrl += fmt.Sprintf(
			"?_current_datetime=%s",
			utcCurrentDatetime.Format("20060102T150405"),
		)
	}
	logrus.Debugf(
		"send request to navitia: %s",
		callUrl,
	)
	resp, err := callNavitia(callUrl, token, connectionTimeout)
	if err != nil {
		return "", err
	}

	// Check the response status code: only 200 and 404 are allowed
	respStatusCode := resp.StatusCode
	if !(respStatusCode == http.StatusOK || respStatusCode == http.StatusNotFound) {
		return "",
			fmt.Errorf(
				"an unexpected error occurred, navitia returned a response with the staus code %d",
				resp.StatusCode,
			)
	}
	if respStatusCode == http.StatusNotFound {
		return "", nil
	}

	navitiaVP := NavitiaVehiclePositions{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&navitiaVP)
	if err != nil {
		VehiclePositionsLoadingErrors.Inc()
		return "", err
	}
	numberOfVehicleJourneyPositions := navitiaVP.getNumberOfVehicleJourneyPositions()
	if numberOfVehicleJourneyPositions == 0 {
		return "", fmt.Errorf("no vehicle journey position has been returned by navitia")
	}
	if numberOfVehicleJourneyPositions > 1 {
		return "", fmt.Errorf(
			"at most 1 vehicle journey position is expected from navitia, get %d",
			numberOfVehicleJourneyPositions,
		)
	}
	return navitiaVP.getVehicleJourneyCode()
}
