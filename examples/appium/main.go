package main

import (
	"fmt"
	"log"

	locatr "github.com/vertexcover-io/locatr/pkg"
	"github.com/vertexcover-io/locatr/pkg/plugins"
)

func main() {
	plugin, err := plugins.NewAppiumPlugin(
		"http://localhost:4723",
		"640daa1b-afdc-45a3-83fd-d0c37cffb3de",
	)
	if err != nil {
		log.Fatal("failed creating appium plugin", err)
	}
	locatr, err := locatr.NewLocatr(plugin)
	if err != nil {
		log.Fatal("failed creating appium locatr locatr", err)
	}
	desc := "This input element is designed for password entry, indicated by its type attribute set to \"password,\" which obscures the text entered for privacy. It requires user input, as denoted by the \"required\" attribute, ensuring that users do not submit the form without filling out this field. The placeholder text prompts users to \"Enter your password,\" guiding them on the expected input. This input is commonly used within forms where sensitive data is collected, such as registration or login forms."
	completion, err := locatr.Locate(desc)
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(completion)
}
