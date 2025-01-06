package main

import (
	"fmt"
	"os"

	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"

	"github.com/vertexcover-io/locatr/golang/baseLocatr"
)

func main() {
	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("LLM_MODEL"),
		os.Getenv("LLM_API_KEY"),
	)
	reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	bLocatr := baseLocatr.BaseLocatrOptions{
		ReRankClient: reRankClient,
		LlmClient:    llmClient,
		LogConfig: logger.LogConfig{
			Level: logger.Debug,
		},
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr("http://172.30.192.1:4723/", "a7e5b9d7-60a8-4f1c-bdb3-93a0956e4ef1", bLocatr)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
	}
	l, err := aLocatr.GetLocatrStr("Give me locatr that shows the current time.")
	if err != nil {
		fmt.Println("erorr getting locatr", err)
	}
	fmt.Println(l)
}
