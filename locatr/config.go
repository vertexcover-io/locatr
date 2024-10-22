package locatr

type LlmConfig struct {
	ApiKey   string `json:"api_key" validate:"regexp=sk-*"`
	Provider string `json:"provider" validate:"regexp=(openai|anthropic)"`
	Model    string `json:"model" validate:"min=2,max=50"`
}

type LocatrConfig struct {
	CachePath string    `json:"cache_path" validate:"required"`
	LlmConfig LlmConfig `json:"llm_config" validate:"required"`
}
