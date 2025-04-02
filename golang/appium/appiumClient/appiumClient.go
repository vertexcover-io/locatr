package appiumClient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/attribute"
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

type SessionResponse struct {
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

func NewAppiumClient(ctx context.Context, serverUrl string, sessionId string) (*AppiumClient, error) {
	_, span := tracing.StartSpan(ctx, "NewAppiumClient")
	defer span.End()

	defer logger.GetTimeLogger("Creating appium client")()

	baseUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	// should be in milliseconds
	joinedUrl := baseUrl.JoinPath("session").JoinPath(sessionId)
	client := CreateNewHttpClient(joinedUrl.String())

	span.SetAttributes(attribute.String("client-url", joinedUrl.String()))

	// added to test session still exists.
	// TODO: consider a parameter to skip the test when interacting from python side
	// todo: create a cached session, Make a stateful session.

	span.AddEvent("fetching /context")
	resp, err := client.R().Get("/context")
	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	span.AddEvent("received /context")
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

func (ac *AppiumClient) ExecuteScript(ctx context.Context, script string, args ...any) (any, error) {
	defer logger.GetTimeLogger("Appium: ExecuteScript")()

	_, span := tracing.StartSpan(ctx, "ExecuteScript")
	defer span.End()

	body := map[string]any{
		"script": script,
		"args":   args,
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	span.AddEvent("request /execute/sync")

	response, err := ac.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyJson).
		Post("/execute/sync")

	span.AddEvent("response /execute/sync")

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

	span.AddEvent("reading response")
	var respBody resp
	err = json.Unmarshal(response.Body(), &respBody)
	if err != nil {
		return response.Body(), nil
	}
	return respBody.Value, nil
}

func (ac *AppiumClient) GetCurrentViewContext(ctx context.Context) (string, error) {
	defer logger.GetTimeLogger("Appium: GetCurrentViewContext")()

	_, span := tracing.StartSpan(ctx, "GetCurrentViewContext")
	defer span.End()

	span.AddEvent("request /context")
	response, err := ac.httpClient.R().Get("/context")
	span.AddEvent("response /context")

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

	span.AddEvent("reading response")

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

func (ac *AppiumClient) IsWebView(ctx context.Context) bool {
	ctx, span := tracing.StartSpan(ctx, "IsWebView")
	defer span.End()

	view, err := ac.GetCurrentViewContext(ctx)
	if err != nil {
		return false
	}
	view = strings.ToLower(view)
	return strings.Contains(view, "webview") || strings.Contains(view, "chromium")
}

func (ac *AppiumClient) GetPageSource(ctx context.Context) (string, error) {
	defer logger.GetTimeLogger("Appium: GetPageSource")()

	_, span := tracing.StartSpan(ctx, "GetPageSource")
	defer span.End()

	span.AddEvent("request /source/")
	response, err := ac.httpClient.R().Get("source/")
	span.AddEvent("response /source/")

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

	span.AddEvent("read response")

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

func (ac *AppiumClient) FindElement(ctx context.Context, locator, locator_type string) error {
	defer logger.GetTimeLogger("Appium: FindElement")()

	_, span := tracing.StartSpan(ctx, "FindElement")
	defer span.End()

	requestBody := findElementRequest{
		Using: locator_type,
		Value: locator,
	}
	span.SetAttributes(
		attribute.String("locator-type", locator_type),
		attribute.String("locator", locator),
	)

	span.AddEvent("request /element")
	response, err := ac.httpClient.R().
		SetBody(requestBody).
		SetResult(&appiumGetElementResponse{}).
		Post("element")
	span.AddEvent("response /element")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}
	if response.StatusCode() != 200 {
		span.AddEvent("read response")

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

func (ac *AppiumClient) GetCapabilities(ctx context.Context) (*SessionResponse, error) {
	defer logger.GetTimeLogger("Appium: GetCapabilities")()

	_, span := tracing.StartSpan(ctx, "GetCapabilities")
	defer span.End()

	span.AddEvent("request /")
	response, err := ac.httpClient.R().SetResult(&SessionResponse{}).Get("/")
	span.AddEvent("response /")

	logger.Logger.Debug(
		"Request sent to Appium server",
		slog.String("url", response.Request.URL),
		slog.String("method", response.Request.Method),
	)

	if err != nil {
		return nil, fmt.Errorf("%w : %w", ErrFailedConnectingToAppiumServer, err)
	}

	span.AddEvent("read response")

	var result SessionResponse
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

func (ac *AppiumClient) GetCurrentActivity(ctx context.Context) (string, error) {
	defer logger.GetTimeLogger("Appium: GetCurrentActivity")()

	_, span := tracing.StartSpan(ctx, "GetCurrentActivity")
	defer span.End()

	span.AddEvent("request /appium/device/current_activity")
	response, err := ac.httpClient.R().SetResult(&getActivityResponse{}).Get("appium/device/current_activity")
	span.AddEvent("response /appium/device/current_activity")

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
