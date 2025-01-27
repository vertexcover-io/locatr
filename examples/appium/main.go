package main

import (
	"fmt"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
)

func main() {
	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("OPENAI_MODEL"),
		os.Getenv("OPENAI_KEY"),
	)
	bLocatr := locatr.BaseLocatrOptions{
		LlmClient: llmClient,
		LogConfig: logger.LogConfig{
			Level: logger.Debug,
		},
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		"https://device.pcloudy.com/appiumcloud/wd/hub",
		"70a4b5a5-ab83-4560-9adc-96d3c1efd9ad", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	desc := "This element is a secure text field designed for user input of sensitive information, specifically a password. It is an interactive element that allows users to enter their password while concealing the text for privacy and security purposes. The field is currently enabled and visible, reflecting that it can be interacted with in the user interface. It is semantically significant as it contributes to user authentication processes, making it essential for forms requiring secure login credentials. Users can expect this type of element to be present in login screens or any scenario that necessitates password entry."
	l, err := aLocatr.GetLocatrStr(desc)
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(l)
}
