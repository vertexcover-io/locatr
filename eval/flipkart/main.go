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
		CachePath:       ".flipkart.cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()

	flightBooking, err := playWrightLocatr.GetLocatr("Flight Bookings button")
	if err != nil {
		log.Printf("could not get locatr for flight bookings button: %v", err)
		return
	}
	err = flightBooking.First().Click()
	if err != nil {
		log.Printf("could not click on flight bookings button: %v", err)
	}
	log.Printf("clicked on flight bookings button")
	time.Sleep(5 * time.Second) // wait for page to load
	fromAndTo, err := playWrightLocatr.GetLocatr("Flight From input field.")
	if err != nil {
		log.Printf("could not get locatr for from input box: %v", err)
	}
	err = fromAndTo.First().Fill("Bangalore")
	if err != nil {
		log.Printf("could not fill from input box: %v", err)
	}
	log.Printf("filled from input box")
	time.Sleep(2 * time.Second) // wait for suggestions to load
	blrAirport, err := playWrightLocatr.GetLocatr("Bangalore (BLR) Airport option")
	if err != nil {
		log.Printf("could not get locatr for Bangalore airport option: %v", err)
		return
	}
	err = blrAirport.First().Click()
	if err != nil {
		log.Printf("could not click on Bangalore airport option: %v", err)
		return
	}
	log.Printf("clicked on Bangalore airport option")
	time.Sleep(2 * time.Second) // wait for suggestions to load

	if err := fromAndTo.Nth(1).Fill("Nepal"); err != nil {
		log.Printf("Failed to fill: %v", err)
	}
	time.Sleep(2 * time.Second)
	kathmanduAirport, err := playWrightLocatr.GetLocatr("Kathmandu, NP Airport option")
	if err != nil {
		log.Printf("could not get locatr for Kathmandu airport option: %v", err)
		return
	}
	if err := kathmanduAirport.First().Click(); err != nil {
		log.Printf("could not click on Kathmandu airport option: %v", err)
		return
	}
	time.Sleep(2 * time.Second) // wait for suggestions to load
	randomDate, err := playWrightLocatr.GetLocatr(fmt.Sprintf("There is a html table with dates, today is %s give me a random date in  the table which will point to future.", time.Now().Format("2006-01-02")))
	if err != nil {
		log.Printf("could not get locatr for background image: %v", err)
		return
	}
	if err := randomDate.First().Click(); err != nil {
		log.Printf("could not click on background image: %v", err)
		return
	}
	log.Printf("clicked on background image")
	travellersClass, err := playWrightLocatr.GetLocatr("Travellers and class input box")
	if err != nil {
		log.Printf("could not get locatr for travellers and class input field: %v", err)
		return
	}
	if err := travellersClass.First().Click(); err != nil {
		log.Printf("could not click on travellers and class input field: %v", err)
		return
	}
	log.Printf("clicked on travellers and class input field")
	businessClass, err := playWrightLocatr.GetLocatr("Business class option")
	if err != nil {
		log.Printf("could not get locatr for business class option: %v", err)
		return
	}
	if err := businessClass.First().Click(); err != nil {
		log.Printf("could not click on business class option: %v", err)
		return
	}
	searchButton, err := playWrightLocatr.GetLocatr("Search Flights button")
	if err != nil {
		log.Printf("could not get locatr for search flights button: %v", err)
		return
	}
	if err := searchButton.First().Click(); err != nil {
		log.Printf("could not click on search flights button: %v", err)
		return
	}
	time.Sleep(5 * time.Second) // wait for page to load
	playWrightLocatr.WriteResultsToFile()
	if err := pw.Stop(); err != nil {
		log.Printf("Error stopping pw: %v", err)
	}
}
