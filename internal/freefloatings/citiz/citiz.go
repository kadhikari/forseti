package citiz

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/CanalTP/forseti/internal/connectors"
	"github.com/CanalTP/forseti/internal/freefloatings"
	"github.com/CanalTP/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

type CitizContext struct {
	connector *connectors.Connector
	auth      *utils.OAuthResponse
	user      string
	password  string
}

func (d *CitizContext) InitContext(externalURI url.URL, loadExternalRefresh,
	connectionTimeout time.Duration, reloadActive bool, userName, password string) {

	d.connector = connectors.NewConnector(externalURI, externalURI,
		"kxJNxs3BsPXr6irjLWyeJkoZs8Z7HdKDPeZwlNWugsUue2MJCg7NmjBts-Q8_U-nncBX9VYXDGDXqzX2EtJ7rwZWrKRw6aZENAC2"+
			"rqa7FPLYfw1Zop5R2-dY4I8fKnleZILU65ClRibSx3qzx2FgnHNaAXJDKcStqqLG8MAOchGFCkm6TbqJtph_PLkcgA7Hxp9iqhOt"+
			"ko4QVfBLPblJPQ", loadExternalRefresh, connectionTimeout)
	d.auth = &utils.OAuthResponse{}
	d.user = userName
	d.password = password
}

func ManagefreeFloatingActivation(context *freefloatings.FreeFloatingsContext, freeFloatingsActive bool) {
	context.ManageFreeFloatingsStatus(freeFloatingsActive)
}

func (d *CitizContext) RefreshFreeFloatingLoop(context *freefloatings.FreeFloatingsContext) {

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
			logrus.Error("Error while reloading freefloating data: ", err)
		} else {
			logrus.Debug("Free_floating data updated")
		}
		time.Sleep(d.connector.GetRefreshTime())
	}
}

func RefreshFreeFloatings(citizContext *CitizContext, context *freefloatings.FreeFloatingsContext) error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadFreeFloatingsData() {
		return nil
	}
	begin := time.Now()

	freeFloatings, err := LoadDatafromConnector(citizContext.connector)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return err
	}

	context.UpdateFreeFloating(freeFloatings)
	freefloatings.FreeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func LoadDatafromConnector(connector *connectors.Connector) ([]freefloatings.FreeFloating, error) {
	urlPath := connector.GetUrl()
	token := "Bearer " + connector.GetToken()
	providerId := 19
	callUrl := fmt.Sprintf("%s/api/provider/%d/extended-vehicles?refresh=true", urlPath.String(), providerId)
	resp, err := utils.GetHttpClient(callUrl, token, "Authorization", connector.GetConnectionTimeout())
	if err != nil {
		return nil, err
	}

	err = utils.CheckResponseStatus(resp)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return nil, err
	}

	vehicles := &CitizData{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(vehicles)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return nil, err
	}

	vehiclesCitiz := LoadVehiclesData(*vehicles)
	logrus.Debug("*** Size of data Citiz: ", len(vehiclesCitiz))

	return vehiclesCitiz, nil
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

// NewFreeFloating creates a new FreeFloating object from the object Vehicle
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
