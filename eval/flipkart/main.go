// nolint
package main

import (
	"fmt"
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
	if _, err := page.Goto("https://www.flipkart.com/"); err != nil {
		log.Fatalf("could not navigate to flipkart.com: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "flipkart_locatr_results.json",
		CachePath:       ".flipkart_cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	playWrightLocatr.WriteResultsToFile()

	flightBooking, err := playWrightLocatr.GetLocatr("Flight Bookings button")
	if err != nil {
		log.Fatalf("could not get locatr for flight bookings button: %v", err)
		return
	}
	err = flightBooking.Nth(3).Click()
	if err != nil {
		log.Fatalf("could not click on flight bookings button: %v", err)
	}
	log.Printf("clicked on flight bookings button")
	time.Sleep(5 * time.Second) // wait for page to load
	fromAndTo, err := playWrightLocatr.GetLocatr("Flight From input field.")
	if err != nil {
		log.Fatalf("could not get locatr for from input box: %v", err)
	}
	err = fromAndTo.First().Fill("Bangalore")
	if err != nil {
		log.Fatalf("could not fill from input box: %v", err)
	}
	log.Printf("filled from input box")
	time.Sleep(2 * time.Second) // wait for suggestions to load
	blrAirport, err := playWrightLocatr.GetLocatr("Bangalore (BLR) Airport option")
	if err != nil {
		log.Fatalf("could not get locatr for Bangalore airport option: %v", err)
		return
	}
	err = blrAirport.First().Click()
	if err != nil {
		log.Fatalf("could not click on Bangalore airport option: %v", err)
		return
	}
	log.Printf("clicked on Bangalore airport option")
	time.Sleep(2 * time.Second) // wait for suggestions to load

	err = fromAndTo.Nth(1).Fill("Nepal")
	time.Sleep(2 * time.Second)                                                         // wait for suggestions to load
	kathmanduAirport, err := playWrightLocatr.GetLocatr("Kathmandu, NP Airport option") // The element is not present in the minified DOM
	if err != nil {
		log.Fatalf("could not get locatr for Kathmandu airport option: %v", err)
		return
	}
	if err := kathmanduAirport.First().Click(); err != nil {
		log.Fatalf("could not click on Kathmandu airport option: %v", err)
		return
	}
	randomDate, err := playWrightLocatr.GetLocatr(fmt.Sprintf("Random departure date in the future, Today is %s", time.Now().Format("2006-01-02")))
	if err != nil {
		log.Fatalf("could not get locatr for random departure date: %v", err)
		return
	}
	if err := randomDate.First().Click(); err != nil {
		log.Fatalf("could not click on random departure date: %v", err)
		return
	}
	time.Sleep(2 * time.Second) // wait for suggestions to load
	searchButton, err := playWrightLocatr.GetLocatr("Search Flights button")
	if err != nil {
		log.Fatalf("could not get locatr for search flights button: %v", err)
		return
	}
	if err := searchButton.First().Click(); err != nil {
		log.Fatalf("could not click on search flights button: %v", err)
		return
	}
	time.Sleep(5 * time.Second) // wait for page to load
	playWrightLocatr.WriteResultsToFile()
}
