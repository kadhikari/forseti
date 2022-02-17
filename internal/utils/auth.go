package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// *************************
// Getting API access tokens
// *************************

type OAuthResponse struct {
	Token      string `json:"access_token"`
	Token_type string `json:"token_type"`
	Expire_in  int    `json:"expires_in"`
}

// SendOAuthRequest - Sends a POST to authenticate
func SendOAuthRequest(urlConnection string, userName string, password string,
	connectionTimeout time.Duration) (*OAuthResponse, error) {

	var err error
	var decodedResponse OAuthResponse

	hc := http.Client{Timeout: 10 * connectionTimeout}
	form := url.Values{}

	// Build form to POST
	form.Add("grant_type", "password")
	form.Add("userName", userName)
	form.Add("password", password)

	req, err := http.NewRequest("POST", urlConnection, strings.NewReader(form.Encode()))
	req.PostForm = form
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error submitting request authentication")
	}

	err = json.NewDecoder(resp.Body).Decode(&decodedResponse)
	if err != nil {
		return nil, fmt.Errorf("Error decoding OAuthResponse")
	}

	return &decodedResponse, err
}

func GetAuthToken(urlPath url.URL, userName string, password string,
	connectionTimeout time.Duration) (*OAuthResponse, error) {
	callUrl := fmt.Sprintf("%s/authentication", urlPath.String())
	authenticate, err := SendOAuthRequest(callUrl, userName, password, connectionTimeout)
	if err != nil {
		return nil, err
	}
	return authenticate, err
}
