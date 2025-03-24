package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/logger"
)

var DefaultEvalFolder = "eval/evalFiles"

func runEval(browser playwright.Browser, eval *evalConfigYaml) []evalResult {
	var results []evalResult = make([]evalResult, 0)
	page, err := browser.NewPage()
	if err != nil {
		logger.Logger.Error("Error creating page", "error", err)
		return nil
	}
	defer page.Close()

	if _, err := page.Goto(eval.Url); err != nil {
		logger.Logger.Error("Error navigating to URL", "url", eval.Url, "error", err)
		return nil
	}

	if eval.Config.PageLoadTimeout > 0 {
		logger.Logger.Info("Waiting for page to load", "timeout", eval.Config.PageLoadTimeout)
		time.Sleep(time.Duration(eval.Config.PageLoadTimeout) * time.Second)
	}

	playWrightLocatr := getLocatrFromYamlConfig(eval, page)
	var lastLocatr playwright.Locator

	for _, step := range eval.Steps {
		logger.Logger.Info("Running step", "stepName", step.Name)

		if step.Action != "" {
			switch step.Action {
			case "click":
				if err := lastLocatr.Nth(step.ElementNo).Click(); err != nil {
					logger.Logger.Error("Error clicking on locator", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Logger.Info("Clicked on item", "stepName", step.Name)
				}
			case "fill":
				if err := lastLocatr.Nth(step.ElementNo).Fill(step.FillText); err != nil {
					logger.Logger.Error("Error filling text", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Logger.Info("Filled text in locator", "stepName", step.Name, "fillText", step.FillText)
				}
			case "hover":
				if err := lastLocatr.Nth(step.ElementNo).Hover(); err != nil {
					logger.Logger.Error("Error hovering on locator", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Logger.Info("Hovered on item", "stepName", step.Name)
				}
			case "press":
				if err := lastLocatr.Nth(step.ElementNo).Press(step.Key); err != nil {
					logger.Logger.Error("Error pressing key", "stepName", step.Name, "key", step.Key, "error", err)
					continue
				} else {
					logger.Logger.Info("Pressed key on locator", "stepName", step.Name, "key", step.Key)
				}
			default:
				logger.Logger.Warn("Unknown action", "action", step.Action)
				continue
			}

			logger.Logger.Info("Waiting after action", "timeout", step.Timeout, "action", step.Action)
			time.Sleep(time.Duration(step.Timeout) * time.Second)
		}

		if step.UserRequest == "" {
			continue
		}

		ctx := context.Background()

		locatrOutput, err := playWrightLocatr.GetLocatr(ctx, step.UserRequest)
		if err != nil {
			logger.Logger.Error("Error getting locator for step", "stepName", step.Name, "error", err)
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

		lastLocatr = page.Locator(locatrOutput.Selectors[0])
		currentResults := playWrightLocatr.GetLocatrResults()
		currentLocatrs := currentResults[len(currentResults)-1].AllLocatrs

		if !compareSlices(step.ExpectedLocatrs, currentLocatrs) {
			logger.Logger.Warn("Expected locators do not match", "expectedLocatrs", step.ExpectedLocatrs, "generatedLocatrs", currentLocatrs)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           false,
				GeneratedLocatrs: currentLocatrs,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "All generated locators do not match expected locators",
			})
		} else {
			logger.Logger.Info("Step finished successfully", "stepName", step.Name)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           true,
				GeneratedLocatrs: currentLocatrs,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "",
			})
		}
	}

	return results
}

func main() {
	evalFolderPath := flag.String("evalFolder", DefaultEvalFolder, "Path to folder with eval files")
	runOnly := flag.String("runOnly", "", "Run only the specified eval file")
	flag.Parse()

	var evalFiles []string
	var evalYamlPath string = DefaultEvalFolder

	if *runOnly == "" {
		if *evalFolderPath != "" {
			evalYamlPath = *evalFolderPath
		}
		evalFiles, err := getAllYamlFiles(evalYamlPath)
		if err != nil {
			logger.Logger.Error("Error retrieving YAML files", "folder", evalYamlPath, "error", err)
			return
		}
		if len(evalFiles) == 0 {
			logger.Logger.Error("No YAML files found in folder", "folder", evalYamlPath)
			return
		}
	} else {
		evalFiles = []string{*runOnly}
	}

	pw, err := playwright.Run()
	if err != nil {
		logger.Logger.Error("Error running playwright", "error", err)
		return
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		logger.Logger.Error("Error launching browser", "error", err)
		return
	}

	for _, evalFile := range evalFiles {
		eval, err := readYamlFile(fmt.Sprintf("%s/%s", evalYamlPath, evalFile))
		if err != nil {
			logger.Logger.Error("Error reading YAML file, skipping", "file", evalFile, "error", err)
			continue
		}

		logger.Logger.Info("Running eval", "evalName", eval.Name)

		results := runEval(browser, eval)
		if results != nil {
			writeEvalResultToCsv(results, fmt.Sprintf("%s.csv", evalFile))
			logger.Logger.Info("Eval results written", "evalName", eval.Name, "file", fmt.Sprintf("%s.csv", evalFile))
		}

		logger.Logger.Info("Eval finished", "evalName", eval.Name)
	}

	err = browser.Close()
	if err != nil {
		logger.Logger.Error("Error closing browser", "error", err)
	}
}
