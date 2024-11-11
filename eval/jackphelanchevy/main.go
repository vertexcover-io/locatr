// nolint
package main

import (
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
			IgnoreDefaultArgs: []string{
				"--enable-automation",
			},
			Args: []string{
				"--disable-blink-features=AutomationControlled",
			},
		},
	)
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err := page.Goto("https://www.jackphelanchevy.info/new-vehicles/"); err != nil {
		log.Fatalf("could not navigate to jackphelanchevy.info: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "jackphelanchevy.json",
		CachePath:       ".jackphelanchevy_cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()

	firstCar, err := playWrightLocatr.GetLocatr("Link to First Chevrolet car on the page")
	if err != nil {
		log.Fatalf("could not get locatr: %v", err)
		return
	}
	if err = firstCar.First().Click(); err != nil {
		log.Fatalf("could not click on first car: %v", err)
		return
	}
	log.Printf("clicked on first car")
	time.Sleep(3 * time.Second)
	scheduleTestDrive, err := playWrightLocatr.GetLocatr("Schedule a test drive link")
	if err != nil {
		log.Fatalf("could not get locatr: %v", err)
		return
	}
	if err = scheduleTestDrive.First().Click(); err != nil {
		log.Fatalf("could not click on schedule test drive link: %v", err)
		return
	}
	log.Printf("clicked on schedule test drive link")
	firstName, err := playWrightLocatr.GetLocatr("First name input")
	if err != nil {
		log.Fatalf("could not get locatr: %v", err)
		return
	}
	if err = firstName.First().Fill("John"); err != nil {
		log.Fatalf("could not fill first name: %v", err)
		return
	}
	lastName, err := playWrightLocatr.GetLocatr("Last name input")
	if err != nil {
		log.Fatalf("could not get locatr: %v", err)
		return
	}
	if err = lastName.First().Fill("Doe"); err != nil {
		log.Fatalf("could not fill last name: %v", err)
		return
	}
	time.Sleep(10 * time.Second)
	playWrightLocatr.WriteResultsToFile()
}
