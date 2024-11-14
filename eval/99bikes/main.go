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

	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
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

	if _, err := page.Goto("https://www.99bikes.com.au/"); err != nil {
		log.Fatalf("could not navigate to 99bikes.com: %v", err)
	}
	time.Sleep(10 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "99bikes.json",
		CachePath:       ".99bikes.cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()
	popupClose, err := playWrightLocatr.GetLocatr("popup close button")
	if err != nil {
		log.Printf("could not get locatr: %v", err)
	} else {
		if err := popupClose.First().Click(); err != nil {
			log.Fatalf("could not click on popup close button: %v", err)
			return
		}
		log.Printf("clicked on popup close button")
		time.Sleep(2 * time.Second)
	}
	electricBikes, err := playWrightLocatr.GetLocatr("electric bikes option in the navbar.")
	if err != nil {
		log.Printf("could not get locatr: %v", err)
		return
	}
	if err := electricBikes.First().Click(); err != nil {
		log.Fatalf("could not click on electric bikes: %v", err)
		return
	}
	log.Printf("clicked on electric bikes")
	electricMountainBikes, err := playWrightLocatr.GetLocatr("Link to Electric Mountain Bikes only.")
	if err != nil {
		log.Printf("could not get locatr: %v", err)
		return
	}
	if err := electricMountainBikes.First().Click(); err != nil {
		log.Fatalf("could not click on electric mountain bikes: %v", err)
		return
	}
	log.Printf("clicked on electric mountain bikes")
	firstBike, err := playWrightLocatr.GetLocatr("First Bike with price range 1k-6k")
	if err != nil {
		log.Printf("could not get locatr: %v", err)
		return
	}
	if err := firstBike.First().Click(); err != nil {
		log.Fatalf("could not click on first electric bike: %v", err)
		return
	}
	log.Printf("clicked on first electric bike")
	time.Sleep(10 * time.Second)
}
