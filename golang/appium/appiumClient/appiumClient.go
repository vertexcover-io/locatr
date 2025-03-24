package appiumClient

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/vertexcover-io/locatr/golang/logger"
)

type AppiumClient struct {
	httpClient *resty.Client
	sessionId  string
}

type appiumPageSourceResponse struct {
	SessionId string `json:"sessionId,omitempty"`
	Value     string `json:"value"`
}

type appiumGetCurrentContextResponse struct {
	Value string `json:"value"`
}

type appiumGetElementResponse struct {
	Value struct {
		Error      string `json:"error"`
		Message    string `json:"message"`
		Stacktrace string `json:"stacktrace"`
	} `json:"value"`
}

type Capabilities struct {
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
		Capabilities
		Cap        Capabilities `json:"capabilities"`
		Error      string       `json:"error"`
		Message    string       `json:"message"`
		Stacktrace string       `json:"stacktrace"`
	} `json:"value"`
}

type findElementRequest struct {
	Value string `json:"value"`
	Using string `json:"using"`
}

type getActivityResponse struct {
	Value string `json:"value"`
}

var ErrSessionNotActive = errors.New("session not active")

var ErrFailedConnectingToAppiumServer = errors.New("failed connecting to appium server")

// TODO: remove json.Unmarshal

func CreateNewHttpClient(baseUrl string) *resty.Client {
	client := resty.New()
	client.SetBaseURL(baseUrl)
	client.SetHeader("Accept", "application/json")
	client.SetTimeout(300 * time.Second)
	return client
}

func NewAppiumClient(serverUrl string, sessionId string) (*AppiumClient, error) {
	defer logger.GetTimeLogger("Creating appium client")()
	baseUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	// should be in milliseconds
	joinedUrl := baseUrl.JoinPath("session").JoinPath(sessionId)
	client := CreateNewHttpClient(joinedUrl.String())

	// added to test session still exists.
	// TODO: consider a parameter to skip the test when interacting from python side
	// todo: create a cached session, Make a stateful session.
	resp, err := client.R().Get("/context")
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%w : %s", ErrSessionNotActive, sessionId)
	}
	return &AppiumClient{
		httpClient: client,
		sessionId:  sessionId,
	}, nil
}

type resp struct {
	Value any `json:"value"`
}

func (ac *AppiumClient) ExecuteScript(script string, args []any) (any, error) {
	defer logger.GetTimeLogger("Appium: ExecuteScript")()

	body := map[string]any{
		"script": script,
		"args":   []string{},
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response, err := ac.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyJson).
		Post("/execute/sync")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotActive, ac.sessionId)
	}
	var respBody resp
	err = json.Unmarshal(response.Body(), &respBody)
	if err != nil {
		return response.Body(), nil
	}
	return respBody.Value, nil
}

func (ac *AppiumClient) GetCurrentViewContext() (string, error) {
	defer logger.GetTimeLogger("Appium: GetCurrentViewContext")()

	response, err := ac.httpClient.R().Get("/context")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w: %s", ErrSessionNotActive, ac.sessionId)
	}
	var responseBody appiumGetCurrentContextResponse
	body := response.Body()
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return "", fmt.Errorf(
			"failed to unmarshal response: %w, expected json, received: %s",
			err,
			body,
		)
	}
	return responseBody.Value, nil
}

func (ac *AppiumClient) IsWebView() bool {
	view, err := ac.GetCurrentViewContext()
	if err != nil {
		return false
	}
	view = strings.ToLower(view)
	return strings.Contains(view, "webview") || strings.Contains(view, "chromium")
}

func (ac *AppiumClient) GetPageSource() (string, error) {
	defer logger.GetTimeLogger("Appium: GetPageSource")()

	response, err := ac.httpClient.R().Get("source/")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w : %s ", ErrSessionNotActive, ac.sessionId)
	}
	var responseBody appiumPageSourceResponse
	body := response.Body()
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return "", fmt.Errorf(
			"failed to unmarshal response: %w, expected json, received: %s",
			err,
			body,
		)
	}
	return responseBody.Value, nil
}

func (ac *AppiumClient) FindElement(locator, locator_type string) error {
	defer logger.GetTimeLogger("Appium: FindElement")()

	requestBody := findElementRequest{
		Using: locator_type,
		Value: locator,
	}
	response, err := ac.httpClient.R().
		SetBody(requestBody).
		SetResult(&appiumGetElementResponse{}).
		Post("element")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		var result appiumGetElementResponse
		body := response.Body()
		err = json.Unmarshal(body, &result)
		if err != nil {
			return fmt.Errorf(
				"failed to unmarshal response: %w, expected json, received: %s",
				err,
				body,
			)
		}
		return fmt.Errorf("%s : %s", result.Value.Error, result.Value.Message)
	}
	return nil
}

func (ac *AppiumClient) GetCapabilities() (*sessionResponse, error) {
	defer logger.GetTimeLogger("Appium: GetCapabilities")()

	response, err := ac.httpClient.R().SetResult(&sessionResponse{}).Get("/")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	var result sessionResponse
	body := response.Body()
	err = json.Unmarshal(body, &result)
	if response.StatusCode() != 200 {
		if err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal response: %w, expected json, received: %s",
				err,
				body,
			)
		}
		return nil, fmt.Errorf("%s : %s", result.Value.Error, result.Value.Message)
	}
	return &result, nil
}

func (ac *AppiumClient) GetCurrentActivity() (string, error) {
	defer logger.GetTimeLogger("Appium: GetCurrentActivity")()

	response, err := ac.httpClient.R().SetResult(&getActivityResponse{}).Get("appium/device/current_activity")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	r := response.Result().(*getActivityResponse)
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w : %s", ErrSessionNotActive, ac.sessionId)
	}
	return r.Value, nil
}
