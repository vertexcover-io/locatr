package locatr // baseLocator_test.go

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

type MockPlugin struct {
	invalidLocators map[string]bool
}

func (m *MockPlugin) GetCurrentContext() string {
	return "test_url"
}

func (m *MockPlugin) IsValidLocator(locatr string) (bool, error) {
	if m.invalidLocators == nil {
		return true, nil
	}
	return !m.invalidLocators[locatr], nil
}

func (m *MockPlugin) GetMinifiedDomAndLocatorMap() (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	SelectorType,
	error,
) {
	return nil, nil, "", nil
}

type MockLlmClient struct {
	mockResponse *llm.ChatCompletionResponse
	returnError  bool
}

func (m *MockLlmClient) ChatCompletion(prompt string) (*llm.ChatCompletionResponse, error) {
	if m.returnError {
		return nil, errors.New("mock llm client error")
	}
	return m.mockResponse, nil
}

func (m *MockLlmClient) GetProvider() llm.LlmProvider {
	return "test_provider"
}

func (m *MockLlmClient) GetModel() string {
	return "test_model"
}

type MockReRankClient struct {
	mockReRankResults *[]reranker.ReRankResult
	returnError       bool
}

func (m *MockReRankClient) ReRank(request reranker.ReRankRequest) (*[]reranker.ReRankResult, error) {
	if m.returnError {
		return nil, errors.New("mock rerank client error")
	}
	return m.mockReRankResults, nil
}

func TestAddCachedLocatrs(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, LlmClient: mockLlmClient}
	baseLocatr := NewBaseLocatr(mockPlugin, options)

	tests := []struct {
		url          string
		locatrName   string
		locatrs      []string
		expected     map[string][]cachedLocatrsDto
		SelectorType SelectorType
	}{
		{
			url:        "http://example.com",
			locatrName: "testLocator",
			locatrs:    []string{"locator1"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
				},
			},
		},
		{
			url:        "http://example.com",
			locatrName: "anotherLocator",
			locatrs:    []string{"locator2"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
					{LocatrName: "anotherLocator", Locatrs: []string{"locator2"}},
				},
			},
		},
		{
			url:        "http://example.com",
			locatrName: "testLocator",
			locatrs:    []string{"locator3"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1", "locator3"}},
					{LocatrName: "anotherLocator", Locatrs: []string{"locator2"}},
				},
			},
		},
	}

	for _, tt := range tests {
		locatrOutput := &LocatrOutput{
			SelectorType: tt.SelectorType,
			Selectors:    tt.locatrs,
		}
		baseLocatr.addCachedLocatrs(tt.url, tt.locatrName, locatrOutput)
		if !reflect.DeepEqual(baseLocatr.cachedLocatrs, tt.expected) {
			t.Errorf("addCachedLocatrs() = %v, want %v", baseLocatr.cachedLocatrs, tt.expected)
		}
	}
}

func TestInitilizeState(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, CachePath: "test_cache.json", LlmClient: mockLlmClient}
	baseLocatr := NewBaseLocatr(mockPlugin, options)

	// Test when cache is successfully loaded
	err := os.WriteFile(options.CachePath, []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}
	defer os.Remove(options.CachePath)

	baseLocatr.initializeState()
	if !baseLocatr.initialized {
		t.Errorf("Expected initilized to be true, got false")
	}
}

func TestLoadLocatorsCache(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, CachePath: "test_cache.json", LlmClient: mockLlmClient}
	baseLocatr := NewBaseLocatr(mockPlugin, options)

	// Test loading a valid cache file
	err := os.WriteFile(options.CachePath, []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}
	defer os.Remove(options.CachePath)

	err = baseLocatr.loadLocatorsCache(options.CachePath)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := map[string][]cachedLocatrsDto{
		"http://example.com": {
			{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
		},
	}
	if !reflect.DeepEqual(baseLocatr.cachedLocatrs, expected) {
		t.Errorf("Expected %v, got %v", expected, baseLocatr.cachedLocatrs)
	}

	// Test loading an invalid cache file
	err = os.WriteFile(options.CachePath, []byte(`invalid json`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}

	err = baseLocatr.loadLocatorsCache(options.CachePath)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Test loading a non-existent cache file
	err = baseLocatr.loadLocatorsCache("non_existent_cache.json")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
func TestWriteLocatorsToCache(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache.json")

	// Test writing to a valid cache path
	cacheData := []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`)
	err := writeLocatorsToCache(cachePath, cacheData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file content
	content, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}
	if !reflect.DeepEqual(content, cacheData) {
		t.Errorf("Expected %s, got %s", cacheData, content)
	}

	// Test writing to a non-existent directory
	nonExistentDir := filepath.Join(tempDir, "non_existent_dir", "test_cache.json")
	err = writeLocatorsToCache(nonExistentDir, cacheData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file content
	content, err = os.ReadFile(nonExistentDir)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}
	if !reflect.DeepEqual(content, cacheData) {
		t.Errorf("Expected %s, got %s", cacheData, content)
	}

	// Test handling errors when creating directories or files
	invalidPath := string([]byte{0})
	err = writeLocatorsToCache(invalidPath, cacheData)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetLocatrsFromState(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, LlmClient: mockLlmClient}
	baseLocatr := NewBaseLocatr(mockPlugin, options)

	testUrl := "https://example.com"
	baseLocatr.cachedLocatrs = map[string][]cachedLocatrsDto{
		testUrl: {
			{
				LocatrName: "test_key",
				Locatrs:    []string{"loc1", "loc2", "loc3"},
			},
			{
				LocatrName: "another_key",
				Locatrs:    []string{"loc4", "loc5"},
			},
		},
		"https://another-example.com": {
			{
				LocatrName: "test_key",
				Locatrs:    []string{"loc6", "loc7"},
			},
		},
	}

	locatrs, _, err := baseLocatr.getLocatrsFromState("test_key", testUrl)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := []string{"loc1", "loc2", "loc3"}
	if len(locatrs) != len(expected) {
		t.Errorf("Expected %v locatrs, got %v", len(expected), len(locatrs))
	}
	for i, v := range expected {
		if locatrs[i] != v {
			t.Errorf("Expected locatr %v at position %v, got %v", v, i, locatrs[i])
		}
	}

	locatrs, _, err = baseLocatr.getLocatrsFromState("non_existing_key", testUrl)
	if err == nil {
		t.Error("Expected error for non-existing key, got nil")
	}
	if err.Error() != fmt.Sprintf("key `%s` not found in cache", "non_existing_key") {
		t.Errorf("Expected 'key not found' error, got %v", err.Error())
	}
	if locatrs != nil {
		t.Errorf("Expected nil locatrs for non-existing key, got %v", locatrs)
	}

	locatrs, _, err = baseLocatr.getLocatrsFromState("test_key", "https://non-existing-url.com")
	if err == nil {
		t.Error("Expected error for non-existing URL, got nil")
	}
	if err.Error() != fmt.Sprintf("key `%s` not found in cache", "test_key") {
		t.Errorf("Expected 'key not found' error, got %v", err.Error())
	}
	if locatrs != nil {
		t.Errorf("Expected nil locatrs for non-existing URL, got %v", locatrs)
	}

	locatrs, _, err = baseLocatr.getLocatrsFromState("test_key", "https://another-example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected = []string{"loc6", "loc7"}
	if len(locatrs) != len(expected) {
		t.Errorf("Expected %v locatrs, got %v", len(expected), len(locatrs))
	}
	for i, v := range expected {
		if locatrs[i] != v {
			t.Errorf("Expected locatr %v at position %v, got %v", v, i, locatrs[i])
		}
	}
}

func TestNewBaseLocatrNoLLmClient(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "openai")
	os.Setenv("LLM_MODEL", "test_model")
	os.Setenv("LLM_API_KEY", "test_key")
	mockPlugin := &MockPlugin{}
	options := BaseLocatrOptions{UseCache: true}
	baseLocatr := NewBaseLocatr(mockPlugin, options)
	if baseLocatr.llmClient == nil {
		t.Errorf("Expected llmClient, got %v", baseLocatr.llmClient)
	}
	assert.Equal(t, baseLocatr.llmClient.GetProvider(), llm.OpenAI)
	assert.Equal(t, baseLocatr.llmClient.GetModel(), "test_model")
}

func TestGetUniqueStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "All duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "Empty array",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "Long string duplicates",
			input:    []string{"test1", "test2", "test1", "test3", "test2"},
			expected: []string{"test1", "test2", "test3"},
		},
		{
			name:     "Special characters",
			input:    []string{"#id", ".class", "#id", "[attr]", ".class"},
			expected: []string{"#id", ".class", "[attr]"},
		},
		{
			name:     "Mixed case duplicates",
			input:    []string{"Test", "test", "TEST", "TeSt"},
			expected: []string{"Test", "test", "TEST", "TeSt"},
		},
		{
			name:     "Numeric strings",
			input:    []string{"1", "2", "1", "3", "2"},
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "Empty strings",
			input:    []string{"", "", ""},
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueStringArray(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLocatrResultMarshalJSON(t *testing.T) {
	// Helper function to create a fixed time for testing
	parseTime := func(timeStr string) time.Time {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			panic(err)
		}
		return t
	}

	testCases := []struct {
		name     string
		input    LocatrResult
		expected string
		wantErr  bool
	}{
		{
			name: "basic marshaling with all fields",
			input: LocatrResult{
				LocatrDescription:        "Find login button",
				Url:                      "https://example.com",
				CacheHit:                 true,
				InputTokens:              100,
				OutputTokens:             50,
				TotalTokens:              150,
				LlmErrorMessage:          "",
				ChatCompletionTimeTaken:  2000,
				AttemptNo:                1,
				LocatrRequestInitiatedAt: parseTime("2024-01-01T10:00:00Z"),
				LocatrRequestCompletedAt: parseTime("2024-01-01T10:00:01Z"),
				AllLocatrs:               []string{"#login-btn", ".login-button"},
				SelectorType:             "css",
			},
			expected: `{
				"locatr_description": "Find login button",
				"url": "https://example.com",
				"cache_hit": true,
				"input_tokens": 100,
				"output_tokens": 50,
				"total_tokens": 150,
				"llm_error_message": "",
				"llm_locatr_generation_time_taken": 2000,
				"attempt_no": 1,
				"request_initiated_at": "2024-01-01T10:00:00Z",
				"request_completed_at": "2024-01-01T10:00:01Z",
				"all_locatrs": ["#login-btn", ".login-button"],
				"selector_type": "css"
			}`,
			wantErr: false,
		},
		{
			name: "empty fields",
			input: LocatrResult{
				LocatrRequestInitiatedAt: parseTime("2024-01-01T10:00:00Z"),
				LocatrRequestCompletedAt: parseTime("2024-01-01T10:00:01Z"),
			},
			expected: `{
				"locatr_description": "",
				"url": "",
				"cache_hit": false,
				"input_tokens": 0,
				"output_tokens": 0,
				"total_tokens": 0,
				"llm_error_message": "",
				"llm_locatr_generation_time_taken": 0,
				"attempt_no": 0,
				"request_initiated_at": "2024-01-01T10:00:00Z",
				"request_completed_at": "2024-01-01T10:00:01Z",
				"all_locatrs": null,
				"selector_type": ""
			}`,
			wantErr: false,
		},
		{
			name: "with error message",
			input: LocatrResult{
				LocatrDescription:        "Find non-existent element",
				LlmErrorMessage:          "Element not found",
				LocatrRequestInitiatedAt: parseTime("2024-01-01T10:00:00Z"),
				LocatrRequestCompletedAt: parseTime("2024-01-01T10:00:01Z"),
			},
			expected: `{
				"locatr_description": "Find non-existent element",
				"url": "",
				"cache_hit": false,
				"input_tokens": 0,
				"output_tokens": 0,
				"total_tokens": 0,
				"llm_error_message": "Element not found",
				"llm_locatr_generation_time_taken": 0,
				"attempt_no": 0,
				"request_initiated_at": "2024-01-01T10:00:00Z",
				"request_completed_at": "2024-01-01T10:00:01Z",
				"all_locatrs": null,
				"selector_type": ""
			}`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the input
			got, err := tc.input.MarshalJSON()
			if (err != nil) != tc.wantErr {
				t.Errorf("LocatrResult.MarshalJSON() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			// If we don't expect an error, compare the JSON
			if !tc.wantErr {
				// Create a normalized version of expected JSON for comparison
				var expectedMap map[string]interface{}
				if err := json.Unmarshal([]byte(tc.expected), &expectedMap); err != nil {
					t.Fatalf("Failed to unmarshal expected JSON: %v", err)
				}

				var gotMap map[string]interface{}
				if err := json.Unmarshal(got, &gotMap); err != nil {
					t.Fatalf("Failed to unmarshal actual JSON: %v", err)
				}

				// Compare the maps
				for key, expectedValue := range expectedMap {
					gotValue, exists := gotMap[key]
					if !exists {
						t.Errorf("Missing key in result: %s", key)
						continue
					}

					// Convert both values to JSON for deep comparison
					expectedJSON, _ := json.Marshal(expectedValue)
					gotJSON, _ := json.Marshal(gotValue)
					if string(expectedJSON) != string(gotJSON) {
						t.Errorf("For key %s, got %v, want %v", key, gotValue, expectedValue)
					}
				}
			}
		})
	}
}

func TestCreateLocatrResultFromOutput(t *testing.T) {
	// Helper function to compare LocatrResults while handling time comparisons
	compareLocatrResults := func(t *testing.T, got, want LocatrResult) {
		t.Helper()

		if got.LocatrDescription != want.LocatrDescription {
			t.Errorf("LocatrDescription = %v, want %v", got.LocatrDescription, want.LocatrDescription)
		}
		if got.Url != want.Url {
			t.Errorf("Url = %v, want %v", got.Url, want.Url)
		}
		if got.CacheHit != want.CacheHit {
			t.Errorf("CacheHit = %v, want %v", got.CacheHit, want.CacheHit)
		}
		if got.InputTokens != want.InputTokens {
			t.Errorf("InputTokens = %v, want %v", got.InputTokens, want.InputTokens)
		}
		if got.OutputTokens != want.OutputTokens {
			t.Errorf("OutputTokens = %v, want %v", got.OutputTokens, want.OutputTokens)
		}
		if got.TotalTokens != want.TotalTokens {
			t.Errorf("TotalTokens = %v, want %v", got.TotalTokens, want.TotalTokens)
		}
		if got.LlmErrorMessage != want.LlmErrorMessage {
			t.Errorf("LlmErrorMessage = %v, want %v", got.LlmErrorMessage, want.LlmErrorMessage)
		}
		if got.ChatCompletionTimeTaken != want.ChatCompletionTimeTaken {
			t.Errorf("ChatCompletionTimeTaken = %v, want %v", got.ChatCompletionTimeTaken, want.ChatCompletionTimeTaken)
		}
		if got.AttemptNo != want.AttemptNo {
			t.Errorf("AttemptNo = %v, want %v", got.AttemptNo, want.AttemptNo)
		}
		if !got.LocatrRequestInitiatedAt.Equal(want.LocatrRequestInitiatedAt) {
			t.Errorf("LocatrRequestInitiatedAt = %v, want %v", got.LocatrRequestInitiatedAt, want.LocatrRequestInitiatedAt)
		}
		if !got.LocatrRequestCompletedAt.Equal(want.LocatrRequestCompletedAt) {
			t.Errorf("LocatrRequestCompletedAt = %v, want %v", got.LocatrRequestCompletedAt, want.LocatrRequestCompletedAt)
		}
		if len(got.AllLocatrs) != len(want.AllLocatrs) {
			t.Errorf("AllLocatrs length = %v, want %v", len(got.AllLocatrs), len(want.AllLocatrs))
		} else {
			for i := range got.AllLocatrs {
				if got.AllLocatrs[i] != want.AllLocatrs[i] {
					t.Errorf("AllLocatrs[%d] = %v, want %v", i, got.AllLocatrs[i], want.AllLocatrs[i])
				}
			}
		}
		if got.SelectorType != want.SelectorType {
			t.Errorf("SelectorType = %v, want %v", got.SelectorType, want.SelectorType)
		}
	}

	testCases := []struct {
		name         string
		userReq      string
		currentUrl   string
		allLocatrs   []string
		output       []locatrOutputDto
		selectorType SelectorType
		want         []LocatrResult
	}{
		{
			name:       "successful single output",
			userReq:    "Find login button",
			currentUrl: "https://example.com",
			allLocatrs: []string{"#login", ".login-btn"},
			output: []locatrOutputDto{
				{
					llmLocatorOutputDto: llmLocatorOutputDto{
						LocatorID: "login-1",
						completionResponse: llm.ChatCompletionResponse{
							InputTokens:  100,
							OutputTokens: 50,
							TotalTokens:  150,
							TimeTaken:    1000,
						},
						Error: "",
					},
					AttemptNo:                1,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
				},
			},
			selectorType: "css",
			want: []LocatrResult{
				{
					LocatrDescription:        "Find login button",
					Url:                      "https://example.com",
					CacheHit:                 false,
					InputTokens:              100,
					OutputTokens:             50,
					TotalTokens:              150,
					LlmErrorMessage:          "",
					ChatCompletionTimeTaken:  1000,
					AttemptNo:                1,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
					AllLocatrs:               []string{"#login", ".login-btn"},
					SelectorType:             "css",
				},
			},
		},
		{
			name:       "multiple outputs with error",
			userReq:    "Find non-existent element",
			currentUrl: "https://example.com",
			allLocatrs: []string{},
			output: []locatrOutputDto{
				{
					llmLocatorOutputDto: llmLocatorOutputDto{
						LocatorID: "",
						completionResponse: llm.ChatCompletionResponse{
							InputTokens:  80,
							OutputTokens: 30,
							TotalTokens:  110,
							TimeTaken:    800,
						},
						Error: "Element not found in first attempt",
					},
					AttemptNo:                1,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
				},
				{
					llmLocatorOutputDto: llmLocatorOutputDto{
						LocatorID: "",
						completionResponse: llm.ChatCompletionResponse{
							InputTokens:  90,
							OutputTokens: 40,
							TotalTokens:  130,
							TimeTaken:    900,
						},
						Error: "Element not found in second attempt",
					},
					AttemptNo:                2,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 2, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 3, 0, time.UTC),
				},
			},
			selectorType: "",
			want: []LocatrResult{
				{
					LocatrDescription:        "Find non-existent element",
					Url:                      "https://example.com",
					CacheHit:                 false,
					InputTokens:              80,
					OutputTokens:             30,
					TotalTokens:              110,
					LlmErrorMessage:          "Element not found in first attempt",
					ChatCompletionTimeTaken:  800,
					AttemptNo:                1,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
					AllLocatrs:               []string{},
					SelectorType:             "",
				},
				{
					LocatrDescription:        "Find non-existent element",
					Url:                      "https://example.com",
					CacheHit:                 false,
					InputTokens:              90,
					OutputTokens:             40,
					TotalTokens:              130,
					LlmErrorMessage:          "Element not found in second attempt",
					ChatCompletionTimeTaken:  900,
					AttemptNo:                2,
					LocatrRequestInitiatedAt: time.Date(2024, 1, 1, 10, 0, 2, 0, time.UTC),
					LocatrRequestCompletedAt: time.Date(2024, 1, 1, 10, 0, 3, 0, time.UTC),
					AllLocatrs:               []string{},
					SelectorType:             "",
				},
			},
		},
		{
			name:       "empty output",
			userReq:    "Find button",
			currentUrl: "https://example.com",
			allLocatrs: []string{},
			output:     []locatrOutputDto{},
			want:       []LocatrResult{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := createLocatrResultFromOutput(
				tc.userReq,
				tc.currentUrl,
				tc.allLocatrs,
				tc.output,
				tc.selectorType,
			)

			if len(got) != len(tc.want) {
				t.Fatalf("got %d results, want %d", len(got), len(tc.want))
			}

			for i := range got {
				compareLocatrResults(t, got[i], tc.want[i])
			}
		})
	}
}

func TestSortRerankChunks(t *testing.T) {
	testCases := []struct {
		name          string
		chunks        []string
		reRankResults []reranker.ReRankResult
		expected      []string
	}{
		{
			name:   "basic sorting",
			chunks: []string{"first", "second", "third", "fourth"},
			reRankResults: []reranker.ReRankResult{
				{Index: 2, Score: 0.9}, // "third"
				{Index: 0, Score: 0.8}, // "first"
				{Index: 3, Score: 0.7}, // "fourth"
				{Index: 1, Score: 0.6}, // "second"
			},
			expected: []string{"third", "first", "fourth", "second"},
		},
		{
			name:          "empty chunks and results",
			chunks:        []string{},
			reRankResults: []reranker.ReRankResult{},
			expected:      []string{},
		},
		{
			name:   "single chunk",
			chunks: []string{"only"},
			reRankResults: []reranker.ReRankResult{
				{Index: 0, Score: 1.0},
			},
			expected: []string{"only"},
		},
		{
			name:   "subset of chunks",
			chunks: []string{"first", "second", "third", "fourth"},
			reRankResults: []reranker.ReRankResult{
				{Index: 1, Score: 0.9}, // "second"
				{Index: 3, Score: 0.8}, // "fourth"
			},
			expected: []string{"second", "fourth"},
		},
		{
			name:   "same scores different order",
			chunks: []string{"first", "second", "third"},
			reRankResults: []reranker.ReRankResult{
				{Index: 2, Score: 0.8}, // "third"
				{Index: 1, Score: 0.8}, // "second"
				{Index: 0, Score: 0.8}, // "first"
			},
			expected: []string{"third", "second", "first"},
		},
		{
			name: "html chunks",
			chunks: []string{
				"<div>Header</div>",
				"<div>Content</div>",
				"<div>Footer</div>",
			},
			reRankResults: []reranker.ReRankResult{
				{Index: 1, Score: 0.95}, // Content
				{Index: 0, Score: 0.85}, // Header
				{Index: 2, Score: 0.75}, // Footer
			},
			expected: []string{
				"<div>Content</div>",
				"<div>Header</div>",
				"<div>Footer</div>",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sortRerankChunks(tc.chunks, tc.reRankResults)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("sortRerankChunks() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestFixLLmJson(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No formatting",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Surrounded by backticks",
			input:    "```{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Prefixed with json and surrounded by backticks",
			input:    "```json{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Prefixed with json without backticks",
			input:    "json{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Backticks only at start",
			input:    "```{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Backticks only at end",
			input:    "{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Json prefix without backticks",
			input:    "json{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only backticks",
			input:    "```",
			expected: "",
		},
		{
			name:     "Only json prefix",
			input:    "json",
			expected: "",
		},
		{
			name:     "Json with escaped characters",
			input:    "```json{\"key\": \"value\\n\"}```",
			expected: `{"key": "value\n"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fixLLmJson(test.input)
			if result != test.expected {
				t.Errorf("got %q, want %q", result, test.expected)
			}
		})
	}
}

func TestGetValidLocator(t *testing.T) {
	mockPlugin := &MockPlugin{}
	baseLocatr := &BaseLocatr{
		plugin: mockPlugin,
		logger: logger.NewLogger(logger.LogConfig{}),
	}

	// Test case 1: All locators are valid
	locators := []string{"locator1", "locator2", "locator3"}
	validLocators, err := baseLocatr.getValidLocator(locators)
	assert.NoError(t, err)
	assert.Equal(t, locators, validLocators)

	// Test case 2: Some locators are invalid
	mockPluginInvalid := &MockPlugin{
		invalidLocators: map[string]bool{
			"locator2": true,
		},
	}
	baseLocatrInvalid := &BaseLocatr{
		plugin: mockPluginInvalid,
		logger: logger.NewLogger(logger.LogConfig{}),
	}

	locatorsWithInvalid := []string{"locator1", "locator2", "locator3"}
	validLocators, err = baseLocatrInvalid.getValidLocator(locatorsWithInvalid)
	assert.NoError(t, err)
	assert.Equal(t, []string{"locator1", "locator3"}, validLocators)

	// Test case 3: No valid locators
	mockPluginNoValid := &MockPlugin{
		invalidLocators: map[string]bool{
			"locator1": true,
			"locator2": true,
			"locator3": true,
		},
	}
	baseLocatrNoValid := &BaseLocatr{
		plugin: mockPluginNoValid,
		logger: logger.NewLogger(logger.LogConfig{}),
	}

	validLocators, err = baseLocatrNoValid.getValidLocator(locators)
	assert.Error(t, err)
	assert.Nil(t, validLocators)
}

// TestLlmGetElementId tests the llmGetElementId method
func TestLlmGetElementId(t *testing.T) {
	// Mock LLM client that returns a successful response
	mockLlmClient := &MockLlmClient{
		mockResponse: &llm.ChatCompletionResponse{
			Completion: `{"locator_id": "test_id"}`,
		},
	}

	baseLocatr := &BaseLocatr{
		llmClient: mockLlmClient,
		logger:    logger.NewLogger(logger.LogConfig{}),
	}

	// Test case 1: Successful LLM response
	htmlDom := "<html>test dom</html>"
	userReq := "find element"

	result, err := baseLocatr.llmGetElementId(htmlDom, userReq)
	assert.NoError(t, err)
	assert.Equal(t, "test_id", result.LocatorID)

	// Test case 2: LLM client returns error
	mockLlmClientError := &MockLlmClient{
		returnError: true,
	}
	baseLocatrError := &BaseLocatr{
		llmClient: mockLlmClientError,
		logger:    logger.NewLogger(logger.LogConfig{}),
	}

	_, err = baseLocatrError.llmGetElementId(htmlDom, userReq)
	assert.Error(t, err)
}

// TestGetLocatrOutput tests the getLocatrOutput method
func TestGetLocatrOutput(t *testing.T) {
	// Mock LLM client that returns a successful response
	mockLlmClient := &MockLlmClient{
		mockResponse: &llm.ChatCompletionResponse{
			Completion: `{"locator_id": "test_id"}`,
		},
	}

	baseLocatr := &BaseLocatr{
		llmClient: mockLlmClient,
		logger:    logger.NewLogger(logger.LogConfig{}),
	}

	// Test case 1: Successful locatr output
	htmlDom := "<html>test dom</html>"
	userReq := "find element"

	result, err := baseLocatr.getLocatrOutput(htmlDom, userReq)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test_id", result.LocatorID)

	// Test case 2: LLM returns error
	mockLlmClientError := &MockLlmClient{
		mockResponse: &llm.ChatCompletionResponse{
			Completion: `{"error": "test error"}`,
		},
	}
	baseLocatrError := &BaseLocatr{
		llmClient: mockLlmClientError,
		logger:    logger.NewLogger(logger.LogConfig{}),
	}

	_, err = baseLocatrError.getLocatrOutput(htmlDom, userReq)
	assert.Error(t, err)
}
