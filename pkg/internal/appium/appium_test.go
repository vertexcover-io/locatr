package appium

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AppiumTestSuite struct {
	suite.Suite
	server *httptest.Server
	client *Client
}

func (s *AppiumTestSuite) SetupTest() {
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"value": map[string]interface{}{
				"platformName":   "Android",
				"automationName": "UiAutomator2",
				"deviceName":     "test-device",
				"appPackage":     "com.test.app",
				"appActivity":    "MainActivity",
			},
		})
		s.Require().NoError(err)
	}))

	client, err := NewClient(s.server.URL, "test-session-id")
	s.Require().NoError(err)
	s.client = client
}

func (s *AppiumTestSuite) TearDownTest() {
	s.server.Close()
}

func TestAppiumSuite(t *testing.T) {
	suite.Run(t, new(AppiumTestSuite))
}

func (s *AppiumTestSuite) TestNewClient_Success() {
	assert.NotNil(s.T(), s.client)
	assert.Equal(s.T(), "test-session-id", s.client.sessionId)
	caps := s.client.Capabilities
	assert.Equal(s.T(), "Android", caps.PlatformName)
	assert.Equal(s.T(), "UiAutomator2", caps.AutomationName)
	assert.Equal(s.T(), "test-device", caps.DeviceName)
}

func (s *AppiumTestSuite) TestNewClient_InvalidURL() {
	client, err := NewClient("invalid-url", "test-session-id")
	assert.Error(s.T(), err)
	assert.Nil(s.T(), client)
}

func (s *AppiumTestSuite) TestExecuteScript() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(s.T(), "/session/test-session-id/execute/sync", r.URL.Path)
		assert.Equal(s.T(), http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		s.Require().NoError(err)
		assert.Equal(s.T(), "test script", requestBody["script"])

		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": "script result",
		})
		s.Require().NoError(err)
	})

	result, err := s.client.ExecuteScript(context.Background(), "test script", []any{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "script result", result)
}

func (s *AppiumTestSuite) TestGetCurrentViewContext() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(s.T(), "/session/test-session-id/context", r.URL.Path)
		assert.Equal(s.T(), http.MethodGet, r.Method)

		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"value": "WEBVIEW_1",
		})
		s.Require().NoError(err)
	})

	context, err := s.client.GetCurrentViewContext(context.Background())
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "WEBVIEW_1", context)
}

func (s *AppiumTestSuite) TestIsWebView() {
	testCases := []struct {
		name           string
		contextValue   string
		expectedResult bool
	}{
		{"WebView Context", "WEBVIEW_1", true},
		{"Chromium Context", "CHROMIUM", true},
		{"Native Context", "NATIVE_APP", false},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := json.NewEncoder(w).Encode(map[string]interface{}{
					"value": tc.contextValue,
				})
				s.Require().NoError(err)
			})

			result := s.client.IsWebView(context.Background())
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func (s *AppiumTestSuite) TestGetPageSource() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(s.T(), "/session/test-session-id/source/", r.URL.Path)
		assert.Equal(s.T(), http.MethodGet, r.Method)

		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"value": "<xml>test source</xml>",
		})
		s.Require().NoError(err)
	})

	source, err := s.client.GetPageSource(context.Background())
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "<xml>test source</xml>", source)
}

func (s *AppiumTestSuite) TestFindElement() {
	testCases := []struct {
		name          string
		contextValue  string
		locator       string
		expectedError bool
		serverStatus  int
		serverResp    map[string]interface{}
	}{
		{
			name:          "WebView Success",
			contextValue:  "WEBVIEW_1",
			locator:       "#element",
			expectedError: false,
			serverStatus:  http.StatusOK,
			serverResp:    map[string]interface{}{"value": map[string]interface{}{"element-6066-11e4-a52e-4f735466cecf": "123"}},
		},
		{
			name:          "Native Success",
			contextValue:  "NATIVE_APP",
			locator:       "//android.view.View",
			expectedError: false,
			serverStatus:  http.StatusOK,
			serverResp:    map[string]interface{}{"value": map[string]interface{}{"element-6066-11e4-a52e-4f735466cecf": "123"}},
		},
		{
			name:          "Element Not Found",
			contextValue:  "NATIVE_APP",
			locator:       "//invalid",
			expectedError: true,
			serverStatus:  http.StatusNotFound,
			serverResp: map[string]interface{}{
				"value": map[string]interface{}{
					"error":   "no such element",
					"message": "Cannot find element",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/session/test-session-id/context" {
					err := json.NewEncoder(w).Encode(map[string]interface{}{
						"value": tc.contextValue,
					})
					s.Require().NoError(err)
					return
				}

				if r.URL.Path == "/session/test-session-id/element" {
					w.WriteHeader(tc.serverStatus)
					err := json.NewEncoder(w).Encode(tc.serverResp)
					s.Require().NoError(err)
				}
			})

			_, err := s.client.FindElement(context.Background(), "", tc.locator)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *AppiumTestSuite) TestExecuteScript_ServerError() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	result, err := s.client.ExecuteScript(context.Background(), "test script", []any{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), ErrEvaulatingScriptFailed.Error())
	assert.Nil(s.T(), result)
}

func (s *AppiumTestSuite) TestGetCurrentViewContext_ServerError() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	context, err := s.client.GetCurrentViewContext(context.Background())
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), ErrSessionNotActive.Error())
	assert.Empty(s.T(), context)
}

func (s *AppiumTestSuite) TestGetCurrentViewContext_UnmarshalError() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"value": invalid_json}`))
		s.Require().NoError(err)
	})

	context, err := s.client.GetCurrentViewContext(context.Background())
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to unmarshal response")
	assert.Empty(s.T(), context)
}

func (s *AppiumTestSuite) TestGetPageSource_ServerError() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	source, err := s.client.GetPageSource(context.Background())
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), ErrSessionNotActive.Error())
	assert.Empty(s.T(), source)
}

func (s *AppiumTestSuite) TestGetPageSource_UnmarshalError() {
	s.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"value": invalid_json}`))
		s.Require().NoError(err)
	})

	source, err := s.client.GetPageSource(context.Background())
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to unmarshal response")
	assert.Empty(s.T(), source)
}

func (s *AppiumTestSuite) TestExecuteScript_ConnectionError() {
	// Close the server to simulate connection error
	s.server.Close()

	result, err := s.client.ExecuteScript(context.Background(), "test script", []any{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), ErrFailedConnectingToAppiumServer.Error())
	assert.Nil(s.T(), result)

	// Recreate the server for other tests
	s.SetupTest()
}
