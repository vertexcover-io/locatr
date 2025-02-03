package main

import (
	"fmt"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
)

func main() {
	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("OPENAI_MODEL"),
		os.Getenv("OPENAI_KEY"),
	)
	bLocatr := locatr.BaseLocatrOptions{
		LlmClient: llmClient,
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		"https://device.pcloudy.com/appiumcloud/wd/hub",
		"89ead025-4cf4-4c44-b723-feff1c3aa28f", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	desc := "This element is a textbox designed for user input, specifically for search queries. It is labeled for accessibility as \"Google Search\" and allows users to enter text. The textbox supports autocomplete features, enhancing the user experience by suggesting possible queries as the user types. It is configured to ignore capitalization and offers spell check capabilities. Additionally, it has a maximum length for input, ensuring submissions remain manageable. The role of the element is defined as a \"textbox,\" indicating its primary purpose for text entry."
	l, err := aLocatr.GetLocatrStr(desc)
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(l)
}
