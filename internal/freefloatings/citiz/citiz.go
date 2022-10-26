package citiz

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/freefloatings"
	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

type CitizContext struct {
	connector *connectors.Connector
	auth      *utils.OAuthResponse
	user      string // user to get token
	password  string // password to get token
}

func (d *CitizContext) InitContext(
	filesURI url.URL,
	externalURI url.URL,
	loadExternalRefresh time.Duration,
	connectionTimeout time.Duration,
	userName string,
	password string,
	cityList string,
) {
	const unusedDuration time.Duration = time.Duration(-1)

	d.connector = connectors.NewConnector(
		filesURI,
		externalURI,
		"",
		loadExternalRefresh,
		unusedDuration,
		connectionTimeout,
	)
	d.auth = &utils.OAuthResponse{}
	d.user = userName
	d.password = password
	if len(cityList) > 0 {
	    d.connector.SetCityList(cityList)
	}
}

func (d *CitizContext) RefreshFreeFloatingLoop(context *freefloatings.FreeFloatingsContext) {

	context.SetPackageName(reflect.TypeOf(CitizContext{}).PkgPath())
	context.RefreshTime = d.connector.GetFilesRefreshTime()

	// Wait 10 seconds before reloading external freefloating informations
	time.Sleep(10 * time.Second)
	for {

		if d.auth.Token == "" || tokenExpired(d.auth.Expire_in) {
			auth, err := utils.GetAuthToken(d.connector.GetUrl(), d.user, d.password, d.connector.GetConnectionTimeout())
			if err != nil {
				return
			}
			auth.Expire_in = int(time.Now().Add(1 * time.Hour).Unix()) // Just for beta test
			d.auth = auth
			d.connector.SetToken(auth.Token)
		}

		err := RefreshFreeFloatings(d, context)
		if err != nil {
			logrus.Error("Error while reloading citiz freefloating data: ")
			for _, e := range err {
				logrus.Error(fmt.Sprintf("\t- %s", e))
			}
		} else {
			logrus.Debug("Free_floating data updated")
		}
		time.Sleep(d.connector.GetFilesRefreshTime())
	}
}

func RefreshFreeFloatings(citizContext *CitizContext, context *freefloatings.FreeFloatingsContext) []error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadFreeFloatingsData() {
		return nil
	}
	begin := time.Now()
	var err []error

	freeFloatings, e := LoadDatafromConnector(citizContext.connector)
	if e != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		err = e
	}

	if len(freeFloatings) > 0 {
		context.UpdateFreeFloating(freeFloatings)
	}

	freefloatings.FreeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return err
}

func LoadDatafromConnector(connector *connectors.Connector) ([]freefloatings.FreeFloating, []error) {

	urlPath := connector.GetUrl()
	token := "bearer " + connector.GetToken()
	freeFloatings := make([]freefloatings.FreeFloating, 0)
	var err []error
	var err_citiz error
	var citiz []Citiz

	// If parameter cityList exist then, get city list from the parameter instead of file
	if len(connector.GetCityList()) > 0 {
	    citiz, err_citiz = ReadCitiesFromConfig(connector.GetCityList())
	} else {
	    citiz, err_citiz = ReadCitizFile(connector.GetFilesUri())
	}

	if err_citiz != nil {
		err = append(err, err_citiz)
		return freeFloatings, err
	}

	for _, c := range citiz {

		callUrl := fmt.Sprintf("%sapi/vehicule/available/all/bycoordinates?latitude=%f&longitude=%f&radius=%d",
			urlPath.String(), c.Latitude, c.Longitude, c.Radius)
		resp, e := utils.GetHttpClient(callUrl, token, "Authorization", connector.GetConnectionTimeout())
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, e)
		}

		e = utils.CheckResponseStatus(resp)
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, fmt.Errorf("error with provider %s", c.City))
			continue
		}

		data := &CitizData{}
		decoder := json.NewDecoder(resp.Body)
		e = decoder.Decode(data)
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, e)
			continue
		}

		vehiclesCitiz := LoadVehiclesData(*data)
		freeFloatings = append(freeFloatings, vehiclesCitiz...)
		logrus.Info("Vehicles loaded from provider ", c.City)
		if len(data.KeyMessage) > 0 {
			logrus.Info("Message from provider ", data.KeyMessage)
		}
	}
	logrus.Debug("*** Size of data Citiz: ", len(freeFloatings))

	return freeFloatings, err
}

func LoadVehiclesData(vehiclesData CitizData) []freefloatings.FreeFloating {
	vehicles := make([]freefloatings.FreeFloating, 0)
	for _, ve := range vehiclesData.Vehicles {
		vehicles = append(vehicles, *NewCitiz(ve))
	}
	return vehicles
}

func ReadCitizFile(path url.URL) ([]Citiz, error) {
	logrus.Debug("*** Using city list from file ***")
	reader, err := utils.GetFileWithFS(path)
	if err != nil {
		return nil, err
	}

	jsonData, err := ioutil.ReadAll(reader)
	if err != nil {
	    logrus.Error("Error while reading city file:", err)
		return nil, err
	}

	data := &AllCitiz{}
	err = json.Unmarshal(jsonData, data)
	if err != nil {
	logrus.Error("Error while reading cities in the file:", err)
		return nil, err
	}

	return *data, nil
}

func ReadCitiesFromConfig(strCitiz string) ([]Citiz, error) {
    logrus.Debug("*** Using city list parameter ***")
    data := &AllCitiz{}
	if err := json.Unmarshal([]byte(strCitiz), &data); err != nil {
        logrus.Error("Error converting city list parameter to object list:", err)
        return nil, err
    }
    return *data, nil
}

func tokenExpired(expireIn int) bool {
	return int(time.Now().Unix()) > expireIn
}

// NewCitiz creates a new FreeFloating object from the object Vehicle
func NewCitiz(ve Vehicle) *freefloatings.FreeFloating {
	return &freefloatings.FreeFloating{
		PublicId:     ve.LicencePlate,
		ProviderName: getProviderName(ve.IsFreeFloating),
		Id:           strconv.Itoa(int(ve.VehiculeId)),
		Type:         "CAR",
		Coord:        freefloatings.Coord{Lat: ve.GpsLatitude, Lon: ve.GpsLongitude},
		Propulsion:   getPropulsion(ve.ElectricEngine),
		Attributes:   []string{getPropulsion((ve.ElectricEngine))},
		Properties: map[string]string{
			"category":       ve.Category,
			"externalRemark": ve.ExternalRemark,
			"stationId":      strconv.Itoa(int(ve.StationId)),
		},
	}
}

func getProviderName(isFreeFloating bool) string {
	if isFreeFloating {
		return "Yea"
	} else {
		return "Citiz"
	}
}

func getPropulsion(isElectric bool) string {
	if isElectric {
		return "ELECTRIC"
	} else {
		return "COMBUSTION"
	}
}

type Vehicle struct {
	VehiculeId     int32   `json:"vehiculeId,omitempty"`
	ProviderId     int32   `json:"providerId,omitempty"`
	FuelLevel      int32   `json:"fuelLevel,omitempty"`
	StationId      int32   `json:"stationId,omitempty"`
	Engine         int32   `json:"engine,omitempty"`
	IsFreeFloating bool    `json:"isFreeFloating,omitempty"`
	ElectricEngine bool    `json:"electricEngine,omitempty"`
	IsCharging     bool    `json:"isCharging,omitempty"`
	Name           string  `json:"name,omitempty"`
	LicencePlate   string  `json:"licencePlate,omitempty"`
	ExternalRemark string  `json:"externalRemark,omitempty"`
	Category       string  `json:"category,omitempty"`
	Icon           string  `json:"icon,omitempty"`
	GpsLatitude    float64 `json:"gpsLatitude,omitempty"`
	GpsLongitude   float64 `json:"gpsLongitude,omitempty"`
}

type CitizData struct {
	Vehicles     []Vehicle `json:"results,omitempty"`
	Status       int32     `json:"status,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Message      string    `json:"message,omitempty"`
	KeyMessage   string    `json:"keyMessage,omitempty"`
}

type AllCitiz []Citiz
type Citiz struct {
	City      string  `json:"city"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Radius    int32   `json:"radius"`
}
