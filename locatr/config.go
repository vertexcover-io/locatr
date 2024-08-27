package locatr

type LlmConfig struct {
	ApiKey   string `validate:"regexp=sk-*"`
	Provider string `validate:"regexp=(openai|anthropic)"`
	Model    string `validate:"min=2,max=50"`
}

type LocatrConfig struct {
	CachePath string    `json:"cache_path" validate:"required"`
	LlmConfig LlmConfig `json:"llm_config" validate:"required"`
}
