package appiumClient

import (
	"context"
	"errors"
	"fmt"

	"github.com/vertexcover-io/locatr/golang/tracing"
)

var ErrNotSupported = errors.New("action is not supported")
var ErrAppiumCommandFailed = errors.New("appium command failed")

type appiumCacheClient struct {
	sessionId    string
	serverUrl    string
	appiumClient *AppiumClient

	cache map[string]any
}

func NewAppiumCacheClient(
	ctx context.Context,
	serverUrl string,
	sessionId string,
) (*appiumCacheClient, error) {
	ctx, span := tracing.StartSpan(ctx, "NewAppiumCacheClient")
	defer span.End()

	appiumClient, err := NewAppiumClient(ctx, serverUrl, sessionId)
	if err != nil {
		return nil, err
	}

	cache := appiumCacheClient{
		sessionId:    sessionId,
		serverUrl:    serverUrl,
		appiumClient: appiumClient,
		cache:        make(map[string]any),
	}
	return &cache, nil
}

func (a *appiumCacheClient) ExecuteScript(
	ctx context.Context,
	script string,
	args ...any,
) (any, error) {
	if !a.IsWebView(ctx) {
		return nil, fmt.Errorf("can't run script in native view: %w", ErrNotSupported)
	}

	return a.appiumClient.ExecuteScript(ctx, script, args)
}

func (a *appiumCacheClient) GetCurrentViewContext(ctx context.Context) (string, error) {
	cacheKey := "getCurrentViewContext"

	if value, ok := a.cache[cacheKey]; ok {
		return value.(string), nil
	}

	viewCtx, err := a.appiumClient.GetCurrentViewContext(ctx)
	if err != nil {
		return "", err
	}
	a.cache[cacheKey] = viewCtx
	return viewCtx, nil
}

func (a *appiumCacheClient) IsWebView(ctx context.Context) bool {
	cacheKey := "isWebView"

	if value, ok := a.cache[cacheKey]; ok {
		return value.(bool)
	}

	webView := a.appiumClient.IsWebView(ctx)
	a.cache[cacheKey] = webView
	return webView
}

func (a *appiumCacheClient) GetPageSource(ctx context.Context) (string, error) {
	cacheKey := "getPageSource"
	if value, ok := a.cache[cacheKey]; ok {
		return value.(string), nil
	}

	source, err := a.appiumClient.GetPageSource(ctx)
	if err != nil {
		return "", err
	}
	a.cache[cacheKey] = source
	return source, nil
}

func (a *appiumCacheClient) FindElement(
	ctx context.Context,
	locator, locator_type string,
) error {
	return a.appiumClient.FindElement(ctx, locator, locator_type)
}

func (a *appiumCacheClient) GetCapabilities(ctx context.Context) (*SessionResponse, error) {
	cacheKey := "getCapabilities"
	if value, ok := a.cache[cacheKey]; ok {
		return value.(*SessionResponse), nil
	}

	cap, err := a.appiumClient.GetCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	a.cache[cacheKey] = cap
	return cap, nil
}

func (a *appiumCacheClient) GetCurrentActivity(ctx context.Context) (string, error) {
	cacheKey := "getCurrentActivity"
	if value, ok := a.cache[cacheKey]; ok {
		return value.(string), nil
	}

	act, err := a.appiumClient.GetCurrentActivity(ctx)
	if err != nil {
		return "", err
	}
	a.cache[cacheKey] = act
	return act, nil
}
