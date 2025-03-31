package mode

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/types"
)

// Reuse the mock implementations from dom-analysis_test.go
// MockPlugin, MockLLMClient, and MockRerankerClient implementations...

func TestVisualAnalysisMode_ProcessRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        string
		resolution     *types.Resolution
		mockSetup      func(*MockPlugin, *MockLLMClient, *MockRerankerClient)
		expectedError  string
		expectedResult []string
	}{
		{
			name:    "successful element identification",
			request: "click the submit button",
			resolution: &types.Resolution{
				Width:  1280,
				Height: 800,
			},
			mockSetup: func(mp *MockPlugin, ml *MockLLMClient, mr *MockRerankerClient) {
				// Add this mock for ExtractFirstUniqueID
				mp.On("ExtractFirstUniqueID", mock.AnythingOfType("string")).Return("button-123", nil)

				// Mock DOM response
				mp.On("GetMinifiedDOM").Return(&types.DOM{
					RootElement: &types.ElementSpec{
						Id: "root",
						Children: []types.ElementSpec{
							{Id: "button-123", TagName: "button", Text: "Submit"},
						},
					},
					Metadata: &types.DOMMetadata{
						LocatorMap: map[string][]string{
							"button-123": {"//*[@id='submit-button']"},
						},
					},
				}, nil)

				// Mock reranker response
				mr.On("Rerank", mock.Anything).Return([]types.RerankResult{
					{Index: 0, Score: 0.9},
				}, nil)

				// Mock viewport setup
				mp.On("SetViewportSize", 1280, 800).Return(nil)

				// Mock element location
				mp.On("GetElementLocation", "//*[@id='submit-button']").Return(&types.Location{
					Point:          types.Point{X: 100, Y: 100},
					ScrollPosition: types.Point{X: 0, Y: 0},
				}, nil)

				// Mock screenshot
				mp.On("TakeScreenshot").Return([]byte("mock-screenshot"), nil)

				// Mock LLM response
				jsonResponse := map[string]string{
					"element_point": "150,200",
					"error":         "",
				}
				jsonBytes, _ := json.Marshal(jsonResponse)
				ml.On("GetJSONCompletion", mock.Anything, mock.Anything).Return(&types.JSONCompletion{
					JSON: string(jsonBytes),
					LLMCompletionMeta: types.LLMCompletionMeta{
						InputTokens:  100,
						OutputTokens: 50,
					},
				}, nil)

				// Mock element locators from point
				mp.On("GetElementLocators", mock.MatchedBy(func(loc *types.Location) bool {
					return loc.Point.X == 150 && loc.Point.Y == 200
				})).Return([]string{"//*[@id='submit-button']"}, nil)
			},
			expectedResult: []string{"//*[@id='submit-button']"},
		},
		{
			name:    "invalid coordinates from LLM",
			request: "click the button",
			resolution: &types.Resolution{
				Width:  1280,
				Height: 800,
			},
			mockSetup: func(mp *MockPlugin, ml *MockLLMClient, mr *MockRerankerClient) {
				// Add this mock for ExtractFirstUniqueID
				mp.On("ExtractFirstUniqueID", mock.AnythingOfType("string")).Return("button-123", nil)

				// Mock DOM response with valid ID that matches the chunk
				mp.On("GetMinifiedDOM").Return(&types.DOM{
					RootElement: &types.ElementSpec{
						Id: "root",
						Children: []types.ElementSpec{
							{Id: "button-123", TagName: "button", Text: "Click Me"},
						},
					},
					Metadata: &types.DOMMetadata{
						LocatorMap: map[string][]string{
							"button-123": {"//*[@id='button']"},
						},
					},
				}, nil)

				// Mock reranker to return chunk containing the button-123 ID
				mr.On("Rerank", mock.Anything).Return([]types.RerankResult{
					{Index: 0, Score: 0.8},
				}, nil)

				// These calls will now be made because we have a valid ID
				mp.On("SetViewportSize", 1280, 800).Return(nil)
				mp.On("GetElementLocation", "//*[@id='button']").Return(&types.Location{
					Point:          types.Point{X: 100, Y: 100},
					ScrollPosition: types.Point{X: 0, Y: 0},
				}, nil)
				mp.On("TakeScreenshot").Return([]byte("mock-screenshot"), nil)

				jsonResponse := map[string]string{
					"element_point": "invalid,coords",
					"error":         "",
				}
				jsonBytes, _ := json.Marshal(jsonResponse)
				ml.On("GetJSONCompletion", mock.Anything, mock.Anything).Return(&types.JSONCompletion{
					JSON: string(jsonBytes),
				}, nil)
			},
			expectedError: "no relevant element point found in the DOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize mocks
			mockPlugin := new(MockPlugin)
			mockLLM := new(MockLLMClient)
			mockReranker := new(MockRerankerClient)

			// Setup mocks
			tt.mockSetup(mockPlugin, mockLLM, mockReranker)

			// Create mode instance
			mode := &VisualAnalysisMode{
				Resolution:  tt.resolution,
				MaxAttempts: 3,
			}

			// Create completion object
			completion := &types.LocatrCompletion{}

			// Execute
			err := mode.ProcessRequest(
				tt.request,
				mockPlugin,
				mockLLM,
				mockReranker,
				slog.Default(),
				completion,
			)

			// Assert
			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, completion.Locators)
			}

			// Verify all mocks
			mockPlugin.AssertExpectations(t)
			mockLLM.AssertExpectations(t)
			mockReranker.AssertExpectations(t)
		})
	}
}

func TestVisualAnalysisMode_applyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		mode     VisualAnalysisMode
		expected VisualAnalysisMode
	}{
		{
			name: "empty values",
			mode: VisualAnalysisMode{},
			expected: VisualAnalysisMode{
				Resolution: &types.Resolution{
					Width:  1280,
					Height: 800,
				},
				MaxAttempts: constants.DEFAULT_TOP_N,
			},
		},
		{
			name: "custom values",
			mode: VisualAnalysisMode{
				Resolution: &types.Resolution{
					Width:  1920,
					Height: 1080,
				},
				MaxAttempts: 5,
			},
			expected: VisualAnalysisMode{
				Resolution: &types.Resolution{
					Width:  1920,
					Height: 1080,
				},
				MaxAttempts: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mode.applyDefaults()
			assert.Equal(t, tt.expected, tt.mode)
		})
	}
}
