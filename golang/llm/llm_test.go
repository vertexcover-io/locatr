package llm

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLlmClient(t *testing.T) {
	tests := []struct {
		name        string
		provider    LlmProvider
		model       string
		apiKey      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid OpenAI Configuration",
			provider:    OpenAI,
			model:       "gpt-4",
			apiKey:      "test-key",
			expectError: false,
		},
		{
			name:        "Invalid Provider",
			provider:    "invalid",
			model:       "gpt-4",
			apiKey:      "test-key",
			expectError: true,
			errorMsg:    "invalid provider for llm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLlmClient(tt.provider, tt.model, tt.apiKey)
			fmt.Println("err", err, "client", client)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, client)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, tt.provider, client.GetProvider())
			assert.Equal(t, tt.model, client.GetModel())
		})
	}
}

func TestCreateLlmClientFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Environment Variables",
			envVars: map[string]string{
				"LLM_PROVIDER": string(OpenAI),
				"LLM_MODEL":    "gpt-4",
				"LLM_API_KEY":  "test-key",
			},
			expectError: false,
		},
		{
			name: "Missing API Key",
			envVars: map[string]string{
				"LLM_PROVIDER": string(OpenAI),
				"LLM_MODEL":    "gpt-4",
			},
			expectError: true,
			errorMsg:    "API key is required",
		},

		{
			name: "Missing Model",
			envVars: map[string]string{
				"LLM_PROVIDER": string(OpenAI),
				"LLM_API_KEY":  "test-key",
			},
			expectError: true,
			errorMsg:    "model name must be between 2 and 50 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			client, err := CreateLlmClientFromEnv()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Nil(t, client)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}
}
