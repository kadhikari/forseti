package freefloatings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/CanalTP/forseti/internal/data"
	"github.com/sirupsen/logrus"
)

func LoadFreeFloatingsData(data *data.Data) ([]FreeFloating, error) {
	// Read response body in json
	freeFloatings := make([]FreeFloating, 0)
	for i := 0; i < len(data.Data.Area.Vehicles); i++ {
		freeFloatings = append(freeFloatings, *NewFreeFloating(data.Data.Area.Vehicles[i]))
	}
	return freeFloatings, nil
}

func ManagefreeFloatingActivation(context *FreeFloatingsContext, freeFloatingsActive bool) {
	context.ManageFreeFloatingsStatus(freeFloatingsActive)
}

func RefreshFreeFloatingLoop(context *FreeFloatingsContext,
	freeFloatingsURI url.URL,
	freeFloatingsToken string,
	freeFloatingsRefresh,
	connectionTimeout time.Duration) {
	if len(freeFloatingsURI.String()) == 0 || freeFloatingsRefresh.Seconds() <= 0 {
		logrus.Debug("FreeFloating data refreshing is disabled")
		return
	}

	// Wait 10 seconds before reloading external freefloating informations
	time.Sleep(10 * time.Second)
	for {
		err := RefreshFreeFloatings(context, freeFloatingsURI, freeFloatingsToken, connectionTimeout)
		if err != nil {
			logrus.Error("Error while reloading freefloating data: ", err)
		}
		logrus.Debug("Free_floating data updated")
		time.Sleep(freeFloatingsRefresh)
	}
}

func RefreshFreeFloatings(context *FreeFloatingsContext, uri url.URL, token string, connectionTimeout time.Duration) error {
	// Continue using last loaded data if loading is deactivated
	if !context.LoadFreeFloatingsData() {
		return nil
	}
	begin := time.Now()
	resp, err := CallHttpClient(uri.String(), token)

	if err != nil {
		FreeFloatingsLoadingErrors.Inc()
		return err
	}

	data := &data.Data{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(data)
	if err != nil {
		FreeFloatingsLoadingErrors.Inc()
		return err
	}

	freeFloatings, err := LoadFreeFloatingsData(data)
	if err != nil {
		FreeFloatingsLoadingErrors.Inc()
		return err
	}

	context.UpdateFreeFloating(freeFloatings)
	FreeFloatingsLoadingDuration.Observe(time.Since(begin).Seconds())
	return nil
}

func CallHttpClient(siteHost, token string) (*http.Response, error) {
	client := &http.Client{}
	data := url.Values{}
	data.Set("query", "query($id: Int!) {area(id: $id) {vehicles{publicId, provider{name}, id, type, attributes ,"+
		"latitude: lat, longitude: lng, propulsion, battery, deeplink } } }")
	// Manage area id in the query (Paris id=6)
	// TODO: load vehicles for more than one area (city)
	area_id := 6
	data.Set("variables", fmt.Sprintf("{\"id\": %d}", area_id))
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
