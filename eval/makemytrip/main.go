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
	if _, err := page.Goto("https://www.makemytrip.com/"); err != nil {
		log.Fatalf("could not navigate to makemytrip.com: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "makemytrip.json",
		CachePath:       ".makemytrip_cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()
	popUpClose, err := playWrightLocatr.GetLocatr("Login popup close button.")
	if err != nil {
		log.Println("could not get locatr for login popup close button: %v", err)
	}
	err = popUpClose.First().Click()
	if err != nil {
		log.Println("could not click on login popup close button: %v", err)
	}

	fromCity, err := playWrightLocatr.GetLocatr("From City button.")
	if err != nil {
		log.Println("could not get locatr for flight bookings button: %v", err)
		return
	}
	err = fromCity.First().Click()
	if err != nil {
		log.Println("could not click on from city button: %v", err)
		return
	}
	log.Println("Clicked on from city button")
	fromCityInput, err := playWrightLocatr.GetLocatr("Second From city input. It is the only writable input. Do not select the readonly input.")
	if err != nil {
		log.Println("could not get locatr for from city input: %v", err)
		return
	}
	err = fromCityInput.First().Fill("Delhi")
	if err != nil {
		log.Println("could not fill from city input: %v", err)
		return
	}
	log.Println("Filled from city input")
	time.Sleep(3 * time.Second)
	hindonAirport, err := playWrightLocatr.GetLocatr("Hindon Airport option.")
	if err != nil {
		log.Println("could not get locatr for hindon airport option: %v", err)
		return
	}
	err = hindonAirport.First().Click()
	if err != nil {
		log.Println("could not click on hindon airport option: %v", err)
		return
	}
	log.Println("Clicked on hindon airport option")
	toCity, err := playWrightLocatr.GetLocatr("To City button.")
	if err != nil {
		log.Println("could not get locatr for to city button: %v", err)
		return
	}
	err = toCity.First().Click()
	if err != nil {
		log.Println("could not click on to city button: %v", err)
		return
	}
	log.Println("Clicked on to city button")
	toCityInput, err := playWrightLocatr.GetLocatr("Writable To city input. Do not select the readonly input.")
	if err != nil {
		log.Println("could not get locatr for to city input: %v", err)
		return
	}
	err = toCityInput.First().Fill("Mumbai")
	if err != nil {
		log.Println("could not fill to city input: %v", err)
		return
	}
	log.Println("Filled to city input")
	time.Sleep(3 * time.Second)
	puneAirport, err := playWrightLocatr.GetLocatr("Pune Airport option.")
	if err != nil {
		log.Println("could not get locatr for pune airport option: %v", err)
		return
	}
	err = puneAirport.First().Click()
	if err != nil {
		log.Println("could not click on pune airport option: %v", err)
		return
	}
	log.Println("Clicked on pune airport option")
	time.Sleep(5 * time.Second)
}
