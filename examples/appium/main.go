package main

import (
	"fmt"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
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
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		"http://172.30.192.1:4723",
		"477d6d25-1c0a-49a4-a640-2b96ea7e9b93", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	l, err := aLocatr.GetLocatrStr("Network and internet id")
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(l)
}
