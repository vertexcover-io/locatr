package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vertexcover-io/locatr"
	"github.com/vertexcover-io/selenium"
	"github.com/vertexcover-io/selenium/chrome"
)

func main() {
	service, err := selenium.NewChromeDriverService("./chromedriver-linux64/chromedriver", 4444)
	if err != nil {
		log.Fatal(err)
		return
	}
	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: []string{}})

	defer service.Stop()
	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		log.Fatal(err)
		return
	}
	driver.Get("https://news.ycombinator.com/")

	reRankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		UseCache:     true,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.

	// wait for page to load
	time.Sleep(3 * time.Second)

	seleniumLocatr, err := locatr.NewRemoteConnSeleniumLocatr(
		"http://localhost:4444/wd/hub", driver.SessionID(), options) // the path must end with /wd/hub

	/*
		or: directly pass the driver
		seleniumLocatr, err := locatr.NewSeleniumLocatr(driver, options)
	*/
	if err != nil {
		log.Fatal(err)
		return
	}
	newsLocatr, err := seleniumLocatr.GetLocatrStr("First news link in the site..")
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(newsLocatr)
}
