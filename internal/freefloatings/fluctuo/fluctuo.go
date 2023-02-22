package fluctuo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/hove-io/forseti/internal/connectors"
	"github.com/hove-io/forseti/internal/data"
	"github.com/hove-io/forseti/internal/freefloatings"
	"github.com/hove-io/forseti/internal/utils"
	"github.com/sirupsen/logrus"
)

type FluctuoContext struct {
	connector *connectors.Connector
}

func (d *FluctuoContext) InitContext(
	externalURI url.URL,
	loadExternalRefresh time.Duration,
	token string,
	connectionTimeout time.Duration,
	areaId int,
) {

	const unusedDuration time.Duration = time.Duration(-1)

	d.connector = connectors.NewConnector(
		externalURI,
		externalURI,
		token,
		loadExternalRefresh,
		unusedDuration,
		connectionTimeout,
	)
	d.connector.SetAreaId(areaId)
}

func LoadFreeFloatingsData(data *data.Data) ([]freefloatings.FreeFloating, error) {
	// Read response body in json
	freeFloatings := make([]freefloatings.FreeFloating, 0)
	for i := 0; i < len(data.Data.Area.Vehicles); i++ {
		freeFloatings = append(freeFloatings, *NewFluctuo(data.Data.Area.Vehicles[i]))
	}
	return freeFloatings, nil
}

func (d *FluctuoContext) RefreshFreeFloatingLoop(context *freefloatings.FreeFloatingsContext) {

	context.SetPackageName(reflect.TypeOf(FluctuoContext{}).PkgPath())
	context.RefreshTime = d.connector.GetFilesRefreshTime()

	// Wait 10 seconds before reloading external freefloating informations
	time.Sleep(10 * time.Second)
	for {
		err := RefreshFreeFloatings(d, context)
		if err != nil {
			logrus.Error("Error while reloading freefloating data: ", err)
		} else {
			logrus.Debug("Free_floating data updated")
		}
		time.Sleep(d.connector.GetFilesRefreshTime())
	}
}

func RefreshFreeFloatings(fluctuoContext *FluctuoContext, context *freefloatings.FreeFloatingsContext) error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadFreeFloatingsData() {
		return nil
	}
	begin := time.Now()
	urlPath := fluctuoContext.connector.GetUrl()
	areaId := fluctuoContext.connector.GetAreaId()
	resp, err := CallHttpClient(urlPath.String(), fluctuoContext.connector.GetToken(), areaId)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return err
	}

	err = utils.CheckResponseStatus(resp)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return err
	}

	data := &data.Data{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(data)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return err
	}

	freeFloatings, err := LoadFreeFloatingsData(data)
	if err != nil {
		freefloatings.FreeFloatingsLoadingErrors.Inc()
		return err
	}

	context.UpdateFreeFloating(freeFloatings)
	logrus.Debug("*** Size of data Fluctuo: ", len(freeFloatings))
	freefloatings.FreeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func CallHttpClient(siteHost, token string, areaId int) (*http.Response, error) {
	client := &http.Client{}
	data := url.Values{}
	data.Set("query", "query($id: Int!) {area(id: $id) {vehicles{publicId, provider{name}, id, type, attributes ,"+
		"latitude: lat, longitude: lng, propulsion, battery, deeplink } } }")
	data.Set("variables", fmt.Sprintf("{\"id\": %d}", areaId))
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1?access_token=%s", siteHost, token),
		bytes.NewBufferString(data.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded; param=value")
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// NewFluctuo creates a new FreeFloating object from the object Vehicle
func NewFluctuo(ve data.Vehicle) *freefloatings.FreeFloating {
	return &freefloatings.FreeFloating{
		PublicId:     ve.PublicId,
		ProviderName: ve.Provider.Name,
		Id:           ve.Id,
		Type:         ve.Type,
		Coord:        freefloatings.Coord{Lat: ve.Latitude, Lon: ve.Longitude},
		Propulsion:   ve.Propulsion,
		Battery:      ve.Battery,
		Deeplink:     ve.Deeplink,
		Attributes:   ve.Attributes,
	}
}
