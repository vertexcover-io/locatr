package mode

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vertexcover-io/locatr/pkg/internal/constants"
	"github.com/vertexcover-io/locatr/pkg/types"
)

// Mock implementations
type MockPlugin struct {
	mock.Mock
}

func (m *MockPlugin) GetCurrentContext(ctx context.Context) (*string, error) {
	args := m.Called(ctx)
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockPlugin) GetMinifiedDOM(ctx context.Context) (*types.DOM, error) {
	args := m.Called(ctx)
	return args.Get(0).(*types.DOM), args.Error(1)
}

func (m *MockPlugin) ExtractFirstUniqueID(ctx context.Context, fragment string) (string, error) {
	args := m.Called(ctx, fragment)
	return args.String(0), args.Error(1)
}

func (m *MockPlugin) IsLocatorValid(ctx context.Context, locator string) (bool, error) {
	args := m.Called(ctx, locator)
	return args.Bool(0), args.Error(1)
}

func (m *MockPlugin) SetViewportSize(ctx context.Context, width, height int) error {
	args := m.Called(ctx, width, height)
	return args.Error(0)
}

func (m *MockPlugin) TakeScreenshot(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockPlugin) GetElementLocators(ctx context.Context, location *types.Location) ([]string, error) {
	args := m.Called(ctx, location)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPlugin) GetElementLocation(ctx context.Context, locator string) (*types.Location, error) {
	args := m.Called(ctx, locator)
	return args.Get(0).(*types.Location), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) GetProvider() types.LLMProvider {
	args := m.Called()
	return args.Get(0).(types.LLMProvider)
}

func (m *MockLLMClient) GetModel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMClient) GetJSONCompletion(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error) {
	args := m.Called(ctx, prompt, image)
	return args.Get(0).(*types.JSONCompletion), args.Error(1)
}

type MockRerankerClient struct {
	mock.Mock
}

func (m *MockRerankerClient) Rerank(ctx context.Context, request *types.RerankRequest) ([]types.RerankResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).([]types.RerankResult), args.Error(1)
}

func TestDOMAnalysisMode_ProcessRequest(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name           string
		request        string
		mockSetup      func(context.Context, *MockPlugin, *MockLLMClient, *MockRerankerClient)
		expectedError  string
		expectedResult []string
	}{
		{
			name:    "successful element identification",
			request: "find the login button",
			mockSetup: func(ctx context.Context, mp *MockPlugin, ml *MockLLMClient, mr *MockRerankerClient) {
				// Mock DOM response
				mp.On("GetMinifiedDOM", ctx).Return(&types.DOM{
					RootElement: &types.ElementSpec{
						Id: "root",
						Children: []types.ElementSpec{
							{Id: "button-1", TagName: "button", Text: "Login"},
						},
					},
					Metadata: &types.DOMMetadata{
						LocatorMap: map[string][]string{
							"elem-123": {"//*[@id='login-button']"},
						},
					},
				}, nil)

				// Mock reranker response with non-empty results
				mr.On("Rerank", ctx, mock.Anything).Return([]types.RerankResult{
					{Index: 0, Score: 0.9},
				}, nil)

				// Mock LLM response
				jsonResponse := map[string]string{
					"element_id": "elem-123",
					"error":      "",
				}
				jsonBytes, _ := json.Marshal(jsonResponse)
				ml.On("GetJSONCompletion", ctx, mock.Anything, mock.Anything).Return(&types.JSONCompletion{
					JSON: string(jsonBytes),
					LLMCompletionMeta: types.LLMCompletionMeta{
						InputTokens:  100,
						OutputTokens: 50,
					},
				}, nil)
			},
			expectedResult: []string{"//*[@id='login-button']"},
		},
		{
			name:    "no element found",
			request: "find nonexistent element",
			mockSetup: func(ctx context.Context, mp *MockPlugin, ml *MockLLMClient, mr *MockRerankerClient) {
				mp.On("GetMinifiedDOM", ctx).Return(&types.DOM{
					RootElement: &types.ElementSpec{Id: "root"},
					Metadata:    &types.DOMMetadata{},
				}, nil)

				mr.On("Rerank", ctx, mock.Anything).Return([]types.RerankResult{
					{Index: 0, Score: 0.1},
				}, nil)

				jsonResponse := map[string]string{
					"element_id": "",
					"error":      "Element not found",
				}
				jsonBytes, _ := json.Marshal(jsonResponse)
				ml.On("GetJSONCompletion", ctx, mock.Anything, mock.Anything).Return(&types.JSONCompletion{
					JSON: string(jsonBytes),
				}, nil)
			},
			expectedError: "no relevant element ID found in the DOM",
		},
		{
			name:    "empty DOM",
			request: "find button",
			mockSetup: func(ctx context.Context, mp *MockPlugin, ml *MockLLMClient, mr *MockRerankerClient) {
				mp.On("GetMinifiedDOM", ctx).Return(&types.DOM{
					RootElement: &types.ElementSpec{},
					Metadata:    &types.DOMMetadata{},
				}, nil)

				mr.On("Rerank", ctx, mock.Anything).Return([]types.RerankResult{}, nil)

				ml.On("GetJSONCompletion", ctx, mock.Anything, mock.Anything).Return(&types.JSONCompletion{
					JSON: `{"element_id": "", "error": "No DOM content available"}`,
				}, nil)
			},
			expectedError: "no relevant element ID found in the DOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize mocks
			mockPlugin := new(MockPlugin)
			mockLLM := new(MockLLMClient)
			mockReranker := new(MockRerankerClient)

			// Setup mocks with context
			tt.mockSetup(ctx, mockPlugin, mockLLM, mockReranker)

			// Create mode instance
			mode := &DOMAnalysisMode{
				ChunkSize:        1000,
				MaxAttempts:      3,
				ChunksPerAttempt: 2,
			}

			// Create completion object
			completion := &types.LocatrCompletion{}

			// Execute with context
			err := mode.ProcessRequest(
				ctx,
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

func TestDOMAnalysisMode_applyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		mode     DOMAnalysisMode
		expected DOMAnalysisMode
	}{
		{
			name: "empty values",
			mode: DOMAnalysisMode{},
			expected: DOMAnalysisMode{
				ChunkSize:        constants.DEFAULT_CHUNK_SIZE,
				MaxAttempts:      constants.DEFAULT_MAX_ATTEMPTS,
				ChunksPerAttempt: constants.DEFAULT_CHUNKS_PER_ATTEMPT,
			},
		},
		{
			name: "custom values",
			mode: DOMAnalysisMode{
				ChunkSize:        500,
				MaxAttempts:      5,
				ChunksPerAttempt: 3,
			},
			expected: DOMAnalysisMode{
				ChunkSize:        500,
				MaxAttempts:      5,
				ChunksPerAttempt: 3,
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
