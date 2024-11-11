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
	if _, err := page.Goto("https://www.bigbasket.com/"); err != nil {
		log.Fatalf("could not navigate to bigbasket.com: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	rerankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		ReRankClient:    rerankClient,
		ResultsFilePath: "bigbasket.json",
		CachePath:       ".bigbasket_cache",
		LogConfig: locatr.LogConfig{
			Level: locatr.Debug,
		},
	}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)
	defer playWrightLocatr.WriteResultsToFile()

	shopByCategory, err := playWrightLocatr.GetLocatr("The second Shop by category button.")
	playWrightLocatr.WriteResultsToFile()
	if err != nil {
		log.Fatalf("could not get locatr for flight bookings button: %v", err)
		return
	}
	err = shopByCategory.Click()
	if err != nil {
		log.Fatalf("could not click on shop by category button: %v", err)
		return
	}
	log.Println("Clicked on shop by category button.")
	time.Sleep(1 * time.Second)

	fruitsAndVegetables, err := playWrightLocatr.GetLocatr("Fruits and Vegetables button.")
	if err != nil {
		log.Fatalf("could not get locatr for fruits and vegetables button: %v", err)
		return
	}
	err = fruitsAndVegetables.First().Hover()
	if err != nil {
		log.Fatalf("could not hover on fruits and vegetables button: %v", err)
		return
	}
	log.Println("Hovered on fruits and vegetables button.")
	floweBouquets, err := playWrightLocatr.GetLocatr("Flower Bouquets, Bunches button.")
	if err != nil {
		log.Fatalf("could not get locatr for flower bouquets, bunches button: %v", err)
		return
	}
	err = floweBouquets.First().Hover()
	if err != nil {
		log.Fatalf("could not hover on flower bouquets, bunches button: %v", err)
		return
	}
	log.Println("Hovered on flower bouquets, bunches button.")
	otherFlowers, err := playWrightLocatr.GetLocatr("Other Flowers button.")
	if err != nil {
		log.Fatalf("could not get locatr for other flowers button: %v", err)
		return
	}
	err = otherFlowers.First().Click()
	if err != nil {
		log.Fatalf("could not click on other flowers button: %v", err)
		return
	}
	log.Println("Clicked on other flowers button.")
	time.Sleep(5 * time.Second) // wait for page to load

}
