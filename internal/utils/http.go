package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func GetHttpClient_(url, token, header string, connectionTimeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * connectionTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" && header != "" {
		req.Header.Set("content-type", "application/x-www-form-urlencoded; param=value")
		req.Header.Set(header, token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, CheckResponseStatus_(resp)
}

func CheckResponseStatus_(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusInternalServerError {
			mes := getMessageError_(string(bodyBytes))
			return fmt.Errorf("ERROR %d: %s", resp.StatusCode, mes)

		} else {
			return fmt.Errorf("ERROR %d: no details for this error", resp.StatusCode)
		}
	}
	return nil
}

func Split_(r rune) bool {
	return r == '{' || r == '}' || r == ':' || r == ','
}

func getMessageError_(bodyString string) string {
	bodySplit := strings.FieldsFunc(bodyString, Split_)
	for idx, field := range bodySplit {
		if strings.Contains(field, "message") {
			return strings.Trim(strings.TrimSpace(bodySplit[idx+1]), "\"")
		}
	}
	return "no details for this error"
}
