package appiumLocatr

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
)

var SessionNotActive = errors.New("session not active")

var FailedConnectingToAppiumServer = errors.New("failed connecting to appium server")

type appiumPageSourceResponse struct {
	SessionId string `json:"sessionId"`
	Value     string `json:"value"`
}

type appiumClinet struct {
	httpClient *resty.Client
	sessionId  string
}

func newAppiumClient(serverUrl string, sessionId string) (*appiumClinet, error) {
	baseUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	joinedUrl := baseUrl.JoinPath("session").JoinPath(sessionId)
	client := CreateNewHttpClient(fmt.Sprintf("%s", joinedUrl))
	resp, err := client.R().Get("")
	if err != nil {
		return nil, fmt.Errorf("%v : %v", FailedConnectingToAppiumServer, err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%v : %s", SessionNotActive, sessionId)
	}
	return &appiumClinet{
		httpClient: client,
		sessionId:  sessionId,
	}, nil
}

func (ac *appiumClinet) getPageSource() (string, error) {
	response, err := ac.httpClient.R().SetResult(&appiumPageSourceResponse{}).Get("source")
	if err != nil {
		return "", FailedConnectingToAppiumServer
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%v : %s ", SessionNotActive, ac.sessionId)
	}
	r := response.Result().(*appiumPageSourceResponse)
	return r.Value, nil
}
