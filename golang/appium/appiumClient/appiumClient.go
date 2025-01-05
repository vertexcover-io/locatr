package appiumClient

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

var ErrSessionNotActive = errors.New("session not active")

var ErrFailedConnectingToAppiumServer = errors.New("failed connecting to appium server")

type appiumPageSourceResponse struct {
	SessionId string `json:"sessionId"`
	Value     string `json:"value"`
}
type appiumGetElementResponse struct {
	Value struct {
		Error   string `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"value"`
}

type AppiumClient struct {
	httpClient *resty.Client
	sessionId  string
}
type capabilities struct {
	PlatformName   string      `json:"platformName"`
	AutomationName string      `json:"automationName"`
	DeviceName     string      `json:"deviceName"`
	AppPackage     string      `json:"appPackage"`
	AppActivity    string      `json:"appActivity"`
	Language       string      `json:"language"`
	Locale         string      `json:"locale"`
	LastScrollData interface{} `json:"lastScrollData"`
}
type sessionResponse struct {
	Value struct {
		Capabilities *capabilities `json:"capabilities,omitempty"`
		Error        string        `json:"error,omitempty"`
		Message      string        `json:"message,omitempty"`
		Stacktrace   string        `json:"stacktrace,omitempty"`
	} `json:"value"`
}

type findElementRequest struct {
	Value string `json:"value"`
	Using string `json:"using"`
}

type getActivityResponse struct {
	Value string `json:"value"`
}

func CreateNewHttpClient(baseUrl string) *resty.Client {
	client := resty.New()
	client.SetBaseURL(baseUrl)
	client.SetHeader("Accept", "application/json")
	client.SetTimeout(2 * time.Second)
	return client
}

func NewAppiumClient(serverUrl string, sessionId string) (*AppiumClient, error) {
	baseUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	joinedUrl := baseUrl.JoinPath("session").JoinPath(sessionId)
	client := CreateNewHttpClient(fmt.Sprintf("%s", joinedUrl))
	resp, err := client.R().Get("")
	if err != nil {
		return nil, fmt.Errorf("%v : %v", ErrFailedConnectingToAppiumServer, err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%v : %s", ErrSessionNotActive, sessionId)
	}
	return &AppiumClient{
		httpClient: client,
		sessionId:  sessionId,
	}, nil
}

func (ac *AppiumClient) GetPageSource() (string, error) {
	response, err := ac.httpClient.R().SetResult(&appiumPageSourceResponse{}).Get("source")
	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%v : %s ", ErrSessionNotActive, ac.sessionId)
	}
	r := response.Result().(*appiumPageSourceResponse)
	return r.Value, nil
}

func (ac *AppiumClient) FindElement(xpath string) error {
	requestBody := findElementRequest{
		Using: "xpath",
		Value: xpath,
	}
	response, err := ac.httpClient.R().
		SetBody(requestBody).
		SetResult(&appiumGetElementResponse{}).
		Post("element")

	if err != nil {
		return fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		r := response.Result().(*appiumGetElementResponse)
		return fmt.Errorf("%s : %s", r.Value.Error, r.Value.Message)
	}
	return nil
}

func (ac *AppiumClient) GetCapabilites() (*capabilities, error) {
	response, err := ac.httpClient.R().SetResult(&sessionResponse{}).Post("element")
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	r := response.Result().(*sessionResponse)
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("%s : %s", r.Value.Error, r.Value.Message)
	}
	return r.Value.Capabilities, nil
}
func (ac *AppiumClient) GetCurrentActivity() (string, error) {
	response, err := ac.httpClient.R().SetResult(&getActivityResponse{}).Post("appium/device/current_activity")
	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	r := response.Result().(*getActivityResponse)
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%s : %s", ErrSessionNotActive, ac.sessionId)
	}
	return r.Value, nil
}
