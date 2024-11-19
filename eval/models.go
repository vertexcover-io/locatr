package main

type evalConfigYaml struct {
	Name   string `yaml:"name"`
	Url    string `yaml:"url"`
	Config struct {
		UseCache         bool   `yaml:"useCache"`
		CachePath        string `yaml:"cachePath"`
		ResulstsFilePath string `yaml:"resultsFilePath"`
		UseReRank        bool   `yaml:"useReRank"`
		PageLoadTimeout  int    `yaml:"pageLoadTimeout"`
	} `yaml:"config"`
	Steps []struct {
		Name            string   `yaml:"name"`
		UserRequest     string   `yaml:"userRequest"`
		ExpectedLocatrs []string `yaml:"expectedLocatrs"`
		Timeout         int      `yaml:"timeout"`
		Action          string   `yaml:"action"`
		FillText        string   `yaml:"fillText"`
		ElementNo       int      `yaml:"elementNo"`
		Key             string   `yaml:"key"`
	} `yaml:"steps"`
}

type evalResult struct {
	Url              string   `json:"url"`
	UserRequest      string   `json:"userRequest"`
	Passed           bool     `json:"passed"`
	GeneratedLocatrs []string `json:"generatedLocatrs"`
	ExpectedLocatrs  []string `json:"expectedLocatrs"`
	Error            string   `json:"error"`
}
