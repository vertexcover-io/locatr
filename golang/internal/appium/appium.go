package appium

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/vertexcover-io/locatr/golang/logging"
)

type Client struct {
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
var ErrCreatingSessionFailed = errors.New("failed creating session")
var ErrEvaulatingScriptFailed = errors.New("failed evaulating script")

var ErrFailedConnectingToAppiumServer = errors.New("failed connecting to appium server")

func createNewHttpClient(baseUrl string) *resty.Client {
	client := resty.New()
	client.SetBaseURL(baseUrl)
	client.SetHeader("Accept", "application/json")
	client.SetTimeout(300 * time.Second)
	return client
}

func NewSession(serverUrl string, capabilities map[string]any) (string, error) {
	defer logging.CreateTopic("Appium: NewSession", logging.DefaultLogger)()

	alwaysMatch := make(map[string]any)
	for k, v := range capabilities {
		if k != "platformName" && !strings.HasPrefix(k, "appium:") {
			alwaysMatch[fmt.Sprintf("appium:%s", k)] = v
		} else {
			alwaysMatch[k] = v
		}
	}

	body := map[string]any{
		"capabilities": map[string]any{
			"alwaysMatch": alwaysMatch,
			"firstMatch":  []any{[]map[string]any{}},
		},
	}

	client := createNewHttpClient(strings.TrimSuffix(serverUrl, "/"))

	resp, err := client.R().SetBody(body).Post("/session")
	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("%w : %s", ErrCreatingSessionFailed, resp.String())
	}

	var responseBody map[string]any
	err = json.Unmarshal(resp.Body(), &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	sessionData, ok := responseBody["value"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("failed to get session value from response: %w", ErrSessionNotActive)
	}

	sessionId, ok := sessionData["sessionId"].(string)
	if !ok || sessionId == "" {
		return "", fmt.Errorf("failed to get sessionId from response: %w", ErrSessionNotActive)
	}
	return sessionId, nil
}

func NewClient(serverUrl string, sessionId string) (*Client, error) {
	defer logging.CreateTopic("Appium: NewClient", logging.DefaultLogger)()

	baseUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}

	joinedUrl := baseUrl.JoinPath("session", sessionId)
	client := createNewHttpClient(joinedUrl.String())

	resp, err := client.R().Get("/context")
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%w : %s", ErrSessionNotActive, sessionId)
	}

	return &Client{
		httpClient: client,
		sessionId:  sessionId,
	}, nil
}

type resp struct {
	Value any `json:"value"`
}

func (c *Client) ExecuteScript(script string, args []any) (any, error) {
	defer logging.CreateTopic("Appium: ExecuteScript", logging.DefaultLogger)()

	bodyJson, err := json.Marshal(map[string]any{"script": script, "args": args})
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyJson).
		Post("/execute/sync")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.IsError() {
		return nil, fmt.Errorf("%w: %s", ErrEvaulatingScriptFailed, response.Error())
	}

	var respBody resp
	err = json.Unmarshal(response.Body(), &respBody)
	if err != nil {
		return response.Body(), nil
	}
	return respBody.Value, nil
}

func (c *Client) GetCurrentViewContext() (string, error) {
	defer logging.CreateTopic("Appium: GetCurrentViewContext", logging.DefaultLogger)()

	response, err := c.httpClient.R().Get("/context")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w: %s", ErrSessionNotActive, c.sessionId)
	}
	var responseBody appiumGetCurrentContextResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return responseBody.Value, nil
}

func (c *Client) IsWebView() bool {
	view, err := c.GetCurrentViewContext()
	if err != nil {
		return false
	}

	view = strings.ToLower(view)
	return strings.Contains(view, "webview") || strings.Contains(view, "chromium")
}

func (c *Client) GetPageSource() (string, error) {
	defer logging.CreateTopic("Appium: GetPageSource", logging.DefaultLogger)()

	response, err := c.httpClient.R().Get("source/")
	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w : %s ", ErrSessionNotActive, c.sessionId)
	}

	var responseBody appiumPageSourceResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return responseBody.Value, nil
}

func (c *Client) FindElement(using, value string) (*string, error) {
	defer logging.CreateTopic("Appium: FindElement", logging.DefaultLogger)()

	requestBody := findElementRequest{
		Using: using,
		Value: value,
	}
	response, err := c.httpClient.R().
		SetBody(requestBody).
		SetResult(&appiumGetElementResponse{}).
		Post("element")

	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}

	if response.StatusCode() != 200 {
		var result appiumGetElementResponse
		err = json.Unmarshal(response.Body(), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return nil, fmt.Errorf("%s : %s", result.Value.Error, result.Value.Message)
	}

	var res map[string]map[string]string
	err = json.Unmarshal(response.Body(), &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	elementId := res["value"]["ELEMENT"]
	return &elementId, nil
}

func (c *Client) GetCapabilities() (*sessionResponse, error) {
	defer logging.CreateTopic("Appium: GetCapabilities", logging.DefaultLogger)()

	response, err := c.httpClient.R().SetResult(&sessionResponse{}).Get("")
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}

	var result sessionResponse
	err = json.Unmarshal(response.Body(), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("%s : %s", result.Value.Error, result.Value.Message)
	}
	return &result, nil
}

func (c *Client) GetCurrentActivity() (string, error) {
	defer logging.CreateTopic("Appium: GetCurrentActivity", logging.DefaultLogger)()

	response, err := c.httpClient.R().SetResult(&getActivityResponse{}).Get("appium/device/current_activity")
	if err != nil {
		return "", fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}

	r := response.Result().(*getActivityResponse)
	if response.StatusCode() != 200 {
		return "", fmt.Errorf("%w : %s", ErrSessionNotActive, c.sessionId)
	}
	return r.Value, nil
}
