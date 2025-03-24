package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
)

func main() {
	ctx := context.Background()

	logger.Level.Set(slog.LevelDebug)

	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("OPENAI_MODEL"),
		os.Getenv("OPENAI_KEY"),
	)
	bLocatr := locatr.BaseLocatrOptions{
		LlmClient: llmClient,
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		ctx,
		"http://localhost:4723",
		"640daa1b-afdc-45a3-83fd-d0c37cffb3de", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	desc := "This input element is designed for password entry, indicated by its type attribute set to \"password,\" which obscures the text entered for privacy. It requires user input, as denoted by the \"required\" attribute, ensuring that users do not submit the form without filling out this field. The placeholder text prompts users to \"Enter your password,\" guiding them on the expected input. This input is commonly used within forms where sensitive data is collected, such as registration or login forms."
	l, err := aLocatr.GetLocatrStr(ctx, desc)
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(l)
}
