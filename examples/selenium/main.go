package main

import (
	"context"
	"fmt"
	"log"
	"time"

	locatr "github.com/vertexcover-io/locatr/pkg"
	"github.com/vertexcover-io/locatr/pkg/plugins"
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

	// wait for page to load
	time.Sleep(3 * time.Second)

	wd, err := selenium.ConnectRemote("http://localhost:4444/wd/hub", driver.SessionID())
	if err != nil {
		log.Fatal(err)
	}
	plugin, err := plugins.NewSeleniumPlugin(&wd)
	if err != nil {
		log.Fatal(err)
	}
	locatr, err := locatr.NewLocatr(plugin)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
		return
	}
	completion, err := locatr.Locate(context.Background(), "First news link in the site.")
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(completion)
}
