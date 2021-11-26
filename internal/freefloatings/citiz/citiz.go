package citiz

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

type CitizContext struct {
	connector *connectors.Connector
	providers []string // list of providers to get vehicles free-floatings
	auth      *utils.OAuthResponse
	user      string // user to get token
	password  string // password to get token
}

func (d *CitizContext) InitContext(externalURI url.URL, loadExternalRefresh time.Duration, providers []string,
	connectionTimeout time.Duration, reloadActive bool, userName, password string) {

	d.connector = connectors.NewConnector(externalURI, externalURI, "", loadExternalRefresh, connectionTimeout)
	d.providers = providers
	d.auth = &utils.OAuthResponse{}
	d.user = userName
	d.password = password
}

func ManagefreeFloatingActivation(context *freefloatings.FreeFloatingsContext, freeFloatingsActive bool) {
	context.ManageFreeFloatingsStatus(freeFloatingsActive)
}

func (d *CitizContext) RefreshFreeFloatingLoop(context *freefloatings.FreeFloatingsContext) {

	context.SetPackageName(reflect.TypeOf(CitizContext{}).PkgPath())
	context.RefreshTime = d.connector.GetRefreshTime()

	if d.auth.Token == "" || tokenExpired() {
		auth, err := utils.GetAuthToken(d.connector.GetUrl(), d.user, d.password, d.connector.GetConnectionTimeout())
		if err != nil {
			return
		}
		d.auth = auth
	}

	// Wait 10 seconds before reloading external freefloating informations
	time.Sleep(10 * time.Second)
	for {
		err := RefreshFreeFloatings(d, context)
		if err != nil {
			logrus.Error("Error while reloading freefloating data: ")
			for _, e := range err {
				logrus.Error(fmt.Sprintf("\t- %s", e))
			}
		} else {
			logrus.Debug("Free_floating data updated")
		}
		time.Sleep(d.connector.GetRefreshTime())
	}
}

func RefreshFreeFloatings(citizContext *CitizContext, context *freefloatings.FreeFloatingsContext) []error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadFreeFloatingsData() {
		return nil
	}
	begin := time.Now()
	var err []error

	freeFloatings, e := LoadDatafromConnector(citizContext.connector, citizContext.providers)
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

func LoadDatafromConnector(connector *connectors.Connector,
	providers []string) ([]freefloatings.FreeFloating, []error) {

	urlPath := connector.GetUrl()
	token := "Bearer " + connector.GetToken()
	freeFloatings := make([]freefloatings.FreeFloating, 0)
	var err []error

	for _, provider := range providers {

		callUrl := fmt.Sprintf("%s/api/provider/%s/extended-vehicles?refresh=true", urlPath.String(), provider)
		resp, e := utils.GetHttpClient(callUrl, token, "Authorization", connector.GetConnectionTimeout())
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, e)
			continue
		}

		e = utils.CheckResponseStatus(resp)
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, fmt.Errorf("error with provider %s", provider))
			continue
		}

		vehicles := &CitizData{}
		decoder := json.NewDecoder(resp.Body)
		e = decoder.Decode(vehicles)
		if e != nil {
			freefloatings.FreeFloatingsLoadingErrors.Inc()
			err = append(err, e)
			continue
		}

		vehiclesCitiz := LoadVehiclesData(*vehicles)
		freeFloatings = append(freeFloatings, vehiclesCitiz...)
		logrus.Info("Vehicles loaded from provider ", provider)
	}

	logrus.Debug("*** Size of data Citiz: ", len(freeFloatings))

	return freeFloatings, err
}

func LoadVehiclesData(vehiclesData CitizData) []freefloatings.FreeFloating {
	vehicles := make([]freefloatings.FreeFloating, 0)
	for _, ve := range vehiclesData {
		vehicles = append(vehicles, *NewCitiz(ve))
	}
	return vehicles
}

func tokenExpired() bool {
	// TODO: verify expire_in from authentication
	return false
}

// NewCitiz creates a new FreeFloating object from the object Vehicle
func NewCitiz(ve Vehicles) *freefloatings.FreeFloating {
	return &freefloatings.FreeFloating{
		PublicId:     ve.LicencePlate,
		ProviderName: "Citiz",
		Id:           strconv.Itoa(int(ve.VehiculeId)),
		Type:         "CAR",
		Coord:        freefloatings.Coord{Lat: ve.GpsLatitude, Lon: ve.GpsLongitude},
		Propulsion:   getPropulsion(ve.ElectricEngine),
		Attributes:   []string{getPropulsion((ve.ElectricEngine))},
		Distance:     float64(ve.KmTotal),
		Parameters: map[string]string{
			"category":       ve.Category,
			"externalRemark": ve.ExternalRemark,
		},
	}
}

func getPropulsion(isElectric bool) string {
	if isElectric {
		return "ELECTRIC"
	} else {
		return "COMBUSTION"
	}
}

// Structure used to load data from Citiz
type CitizData []Vehicles
type Vehicles struct {
	LocationId       int32   `json:"locationId,omitempty"`
	LocationCityId   int32   `json:"locationCityId,omitempty"`
	CategoryId       int32   `json:"categoryId,omitempty"`
	KmTotal          int32   `json:"kmTotal,omitempty"`
	VehiculeId       int32   `json:"vehiculeId,omitempty"`
	ProviderId       int32   `json:"providerId,omitempty"`
	StationId        int32   `json:"stationId,omitempty"`
	Engine           int32   `json:"engine,omitempty"`
	FuelLevel        int32   `json:"fuelLevel,omitempty"`
	IsFreeFloating   bool    `json:"isFreeFloating,omitempty"`
	ElectricEngine   bool    `json:"electricEngine,omitempty"`
	IsCharging       bool    `json:"isCharging,omitempty"`
	LocationName     string  `json:"locationName,omitempty"`
	LocationCity     string  `json:"locationCity,omitempty"`
	InternalRemark   string  `json:"internalRemark,omitempty"`
	AffectationStart string  `json:"affectationStart,omitempty"`
	AffectationEnd   string  `json:"affectationEnd,omitempty"`
	Voltage          string  `json:"voltage,omitempty"`
	Name             string  `json:"name,omitempty"`
	LicencePlate     string  `json:"licencePlate,omitempty"`
	ExternalRemark   string  `json:"externalRemark,omitempty"`
	Category         string  `json:"category,omitempty"`
	GpsLatitude      float64 `json:"gpsLatitude,omitempty"`
	GpsLongitude     float64 `json:"gpsLongitude,omitempty"`
}
