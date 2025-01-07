package main

import (
	"fmt"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

func main() {
	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("LLM_MODEL"),
		os.Getenv("LLM_API_KEY"),
	)
	reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))
	bLocatr := locatr.BaseLocatrOptions{
		ReRankClient: reRankClient,
		LlmClient:    llmClient,
		LogConfig: logger.LogConfig{
			Level: logger.Debug,
		},
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		"https://staging.pcloudy.com/appiumcloud/wd/hub",
		"e4432b14-cc80-4f69-88fa-0096cfe9374f", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	l, err := aLocatr.GetLocatrStr("Airplane Mode id")
	if err != nil {
		fmt.Println("erorr getting locatr", err)
	}
	fmt.Println(l)
}
