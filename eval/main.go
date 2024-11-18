package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
)

var DefaultEvalFolder = "./eval/evalFiles"

func runEval(browser playwright.Browser, eval *evalConfigYaml) []evalResult {
	var results []evalResult = make([]evalResult, 0)
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Error creating page: %s", err)
		return nil
	}
	defer page.Close()
	if _, err := page.Goto(eval.Url); err != nil {
		log.Fatalf("Error navigating to %s: %s", eval.Url, err)
		return nil
	}
	if eval.Config.PageLoadTimeout > 0 {
		log.Printf("Waiting for %d seconds for page to load", eval.Config.PageLoadTimeout)
		time.Sleep(time.Duration(eval.Config.PageLoadTimeout) * time.Second)
	}
	playWrightLocatr := getLocatrFromYamlConfig(eval, page)
	var lastLocatr playwright.Locator
	for _, step := range eval.Steps {
		log.Printf("Running step %s", step.Name)
		if step.Action != "" {
			switch step.Action {
			case "click":
				if err := lastLocatr.First().Click(); err != nil {
					log.Printf("Error clicking on locator: %s", err)
				} else {
					log.Fatalf("Clicked on item %s", step.Name)
				}
			case "fill":
				if err := lastLocatr.First().Fill(step.FillText); err != nil {
					log.Printf("Error filling text: %s", err)
				} else {
					log.Fatalf("Filled text %s in locatr %s", step.FillText, step.Name)
				}
			case "hover":
				if err := lastLocatr.First().Hover(); err != nil {
					log.Printf("Error hovering on locator: %s", err)
				} else {
					log.Fatalf("Hovered on item %s", step.Name)
				}
			default:
				log.Fatalf("Unknown action %s", step.Action)
			}
			log.Printf("Waiting for %d seconds after action %s", step.Timeout, step.Action)
			time.Sleep(time.Duration(step.Timeout) * time.Second)
		}
		if step.UserRequest == "" {
			continue
		}
		lastLocatr, err = playWrightLocatr.GetLocatr(step.UserRequest)
		if err != nil {
			log.Fatalf("Error getting locatr for step %s: %s", step.Name, err)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           false,
				GeneratedLocatrs: nil,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            err.Error(),
			})
			continue
		}
		currentLocatrs := playWrightLocatr.GetLocatrResults()
		if !compareSlices(step.ExpectedLocatrs,
			currentLocatrs[len(currentLocatrs)-1].AllLocatrs) {
			log.Printf("Expected locatrs %v, but got %v",
				step.ExpectedLocatrs, currentLocatrs[len(currentLocatrs)-1].AllLocatrs)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           false,
				GeneratedLocatrs: currentLocatrs[len(currentLocatrs)-1].AllLocatrs,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "All generated locatrs do not match expected locatrs",
			})
		} else {
			log.Printf("Step %s finished successfully", step.Name)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           true,
				GeneratedLocatrs: currentLocatrs[len(currentLocatrs)-1].AllLocatrs,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "",
			})
		}
	}
	return results
}

func main() {
	evalFolderPath := flag.String("evalFolder", DefaultEvalFolder, "Path to folder with eval files")
	flag.Parse()
	evalYamlPath := DefaultEvalFolder
	if *evalFolderPath != "" {
		evalYamlPath = *evalFolderPath
	}
	evalFiles := getAllYamlFiles(evalYamlPath)
	if len(evalFiles) == 0 {
		log.Fatal("No yaml files found in folder")
	}
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Error running playwright: %s", err)
		return
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		log.Fatalf("Error launching browser: %s", err)
		return
	}
	for _, evalFile := range evalFiles {
		eval, err := readYamlFile(fmt.Sprintf("%s/%s", evalYamlPath, evalFile))
		if err != nil {
			log.Fatalf("Error reading yaml file %s", evalFile)
		}
		log.Printf("Running eval %s", eval.Name)
		results := runEval(browser, eval)
		if results != nil {
			writeEvalResultToCsv(results, fmt.Sprintf("%s.csv", evalFile))
			log.Printf("Eval %s results written to %s.csv", eval.Name, evalFile)
		}
		log.Printf("Eval %s finished", eval.Name)
	}
	err = browser.Close()
	if err != nil {
		log.Fatalf("Error closing browser: %s", err)
	}
}
