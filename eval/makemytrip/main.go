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

	if _, err := page.Goto("https://www.makemytrip.com/"); err != nil {
		log.Fatalf("could not navigate to makemytrip.com: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "makemytrip.json",
		CachePath:       ".makemytrip.cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()

	popUpClose, err := playWrightLocatr.GetLocatr("Login popup close button.")

	if err != nil {
		log.Println("could not get locatr for login popup close button", err)

		log.Println("Clicked on login popup close button")
	} else {
		err = popUpClose.First().Click()
		if err != nil {
			log.Println("could not click on login popup close buttonv", err)
		}
	}

	fromCity, err := playWrightLocatr.GetLocatr("From City button.")
	if err != nil {
		log.Println("could not get locatr for flight bookings button", err)
		return
	}
	err = fromCity.First().Click()
	if err != nil {
		log.Println("could not click on from city button", err)
		return
	}
	log.Println("Clicked on from city button")
	fromCityInput, err := playWrightLocatr.GetLocatr("Inputable From city input field.")
	if err != nil {
		log.Println("could not get locatr for from city input", err)
		return
	}
	err = fromCityInput.First().Fill("Delhi")
	if err != nil {
		log.Println("could not fill from city input", err)
		return
	}
	log.Println("Filled from city input")
	time.Sleep(3 * time.Second)
	hindonAirport, err := playWrightLocatr.GetLocatr("Hindon Airport option.")
	if err != nil {
		log.Println("could not get locatr for hindon airport option", err)
		return
	}
	err = hindonAirport.First().Click()
	if err != nil {
		log.Println("could not click on hindon airport option", err)
		return
	}
	log.Println("Clicked on hindon airport option")
	toCity, err := playWrightLocatr.GetLocatr("To City button.")
	if err != nil {
		log.Println("could not get locatr for to city button", err)
		return
	}
	err = toCity.First().Click()
	if err != nil {
		log.Println("could not click on to city button", err)
		return
	}
	log.Println("Clicked on to city button")
	toCityInput, err := playWrightLocatr.GetLocatr("Inputable To city input field.")
	if err != nil {
		log.Println("could not get locatr for to city input: ", err)
		return
	}
	err = toCityInput.First().Fill("Mumbai")
	if err != nil {
		log.Println("could not fill to city input: ", err)
		return
	}
	log.Println("Filled to city input")
	time.Sleep(3 * time.Second)
	puneAirport, err := playWrightLocatr.GetLocatr("Pune Airport option.")
	if err != nil {
		log.Println("could not get locatr for pune airport option: ", err)
		return
	}
	err = puneAirport.First().Click()
	if err != nil {
		log.Println("could not click on pune airport option: ", err)
		return
	}
	log.Println("Clicked on pune airport option")
	time.Sleep(5 * time.Second)
	depattureDate, err := playWrightLocatr.GetLocatr(fmt.Sprintf("Today's date: %s, pick a random departure date.", time.Now().Format("2006-01-02")))
	if err != nil {
		log.Println("could not get locatr for random departure date: ", err)
		return
	}
	err = depattureDate.First().Click()
	if err != nil {
		log.Println("could not click on random departure date: ", err)
		return
	}
	log.Println("Clicked on random departure date")
	time.Sleep(3 * time.Second)
	travelClass, err := playWrightLocatr.GetLocatr("Travel Class button.")
	if err != nil {
		log.Println("could not get locatr for travel class button: ", err)
		return
	}
	err = travelClass.First().Click()
	if err != nil {
		log.Println("could not click on travel class button: ", err)
		return
	}
	log.Println("Clicked on travel class button")
	premiumEconomy, err := playWrightLocatr.GetLocatr("Premium Economy option.")
	if err != nil {
		log.Println("could not get locatr for premium economy option: ", err)
		return
	}
	err = premiumEconomy.First().Click()
	if err != nil {
		log.Println("could not click on premium economy option: ", err)
		return
	}
	log.Println("Clicked on premium economy option")
	time.Sleep(3 * time.Second)
	apply, err := playWrightLocatr.GetLocatr("Apply button for flights.")
	if err != nil {
		log.Println("could not get locatr for apply button: ", err)
		return
	}
	err = apply.First().Click()
	if err != nil {
		log.Println("could not click on apply button: ", err)
		return
	}
	log.Println("Clicked on apply button")
	time.Sleep(3 * time.Second)
	searchFlights, err := playWrightLocatr.GetLocatr("Search button for flights.")
	if err != nil {
		log.Println("could not get locatr for search flights button: ", err)
		return
	}
	err = searchFlights.First().Click()
	if err != nil {
		log.Println("could not click on search flights button: ", err)
		return
	}
	log.Println("Clicked on search flights button")
	time.Sleep(10 * time.Second)
	if err := pw.Stop(); err != nil {
		log.Printf("Error stopping pw: %v", err)
	}
}
