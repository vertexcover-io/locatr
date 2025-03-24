package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/logging"
	"github.com/vertexcover-io/locatr/golang/mode"
	"github.com/vertexcover-io/locatr/golang/plugins"
	"gopkg.in/yaml.v3"
)

var DefaultEvalFolder = "."

type evalConfigYaml struct {
	Name   string `yaml:"name"`
	Url    string `yaml:"url"`
	Config struct {
		ResultsFilePath string `yaml:"resultsFilePath"`
		PageLoadTimeout int    `yaml:"pageLoadTimeout"`
	} `yaml:"config"`
	Steps []struct {
		Name            string   `yaml:"name"`
		UserRequest     string   `yaml:"userRequest"`
		ExpectedLocatrs []string `yaml:"expectedLocatrs"`
		Timeout         int      `yaml:"timeout"`
		Action          string   `yaml:"action"`
		FillText        string   `yaml:"fillText"`
		ElementNo       int      `yaml:"elementNo"`
		Key             string   `yaml:"key"`
	} `yaml:"steps"`
}

type evalResult struct {
	Url              string   `json:"url"`
	UserRequest      string   `json:"userRequest"`
	Passed           bool     `json:"passed"`
	GeneratedLocatrs []string `json:"generatedLocatrs"`
	ExpectedLocatrs  []string `json:"expectedLocatrs"`
	Error            string   `json:"error"`
}

func getAllYamlFiles(folder string, logger *slog.Logger) ([]string, error) {
	files, err := os.ReadDir(folder)
	if err != nil {
		logger.Error("Failed to read directory", "folder", folder, "error", err)
		return nil, err
	}

	var res []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") {
			res = append(res, file.Name())
		}
	}
	logger.Info("Successfully retrieved YAML files",
		"folder", folder, "fileCount", len(res))
	return res, nil
}

func runEval(context playwright.BrowserContext, eval *evalConfigYaml, modeString string, logger *slog.Logger) []evalResult {
	var results []evalResult = make([]evalResult, 0)
	page, err := context.NewPage()
	if err != nil {
		logger.Error("Error creating page", "error", err)
		return nil
	}
	defer page.Close()

	if _, err := page.Goto(eval.Url); err != nil {
		logger.Error("Error navigating to URL", "url", eval.Url, "error", err)
		return nil
	}

	if eval.Config.PageLoadTimeout > 0 {
		logger.Info("Waiting for page to load", "timeout", eval.Config.PageLoadTimeout)
		time.Sleep(time.Duration(eval.Config.PageLoadTimeout) * time.Second)
	}

	options := []locatr.Option{locatr.WithLogger(logger)}

	if modeString == "visual-analysis" {
		options = append(options, locatr.WithMode(&mode.VisualAnalysisMode{MaxAttempts: 3}))
	} else {
		options = append(options, locatr.WithMode(&mode.DOMAnalysisMode{MaxAttempts: 3, ChunksPerAttempt: 3}))
	}
	plugin := plugins.NewPlaywrightPlugin(&page)
	locatr, err := locatr.NewLocatr(plugin, options...)
	if err != nil {
		logger.Error("Error creating locatr", "error", err)
		return nil
	}

	var pwElement playwright.Locator

	var totalCost float64 = 0
	for _, step := range eval.Steps {
		logger.Info("Running step", "stepName", step.Name)

		if step.Action != "" {
			switch step.Action {
			case "click":
				if err := pwElement.Nth(step.ElementNo).Click(); err != nil {
					logger.Error("Error clicking on locator", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Info("Clicked on item", "stepName", step.Name)
				}
			case "fill":
				if err := pwElement.Nth(step.ElementNo).Fill(step.FillText); err != nil {
					logger.Error("Error filling text", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Info("Filled text in locator", "stepName", step.Name, "fillText", step.FillText)
				}
			case "hover":
				if err := pwElement.Nth(step.ElementNo).Hover(); err != nil {
					logger.Error("Error hovering on locator", "stepName", step.Name, "error", err)
					continue
				} else {
					logger.Info("Hovered on item", "stepName", step.Name)
				}
			case "press":
				if err := pwElement.Nth(step.ElementNo).Press(step.Key); err != nil {
					logger.Error("Error pressing key", "stepName", step.Name, "key", step.Key, "error", err)
					continue
				} else {
					logger.Info("Pressed key on locator", "stepName", step.Name, "key", step.Key)
				}
			default:
				logger.Warn("Unknown action", "action", step.Action)
				continue
			}

			logger.Info("Waiting after action", "timeout", step.Timeout, "action", step.Action)
			time.Sleep(time.Duration(step.Timeout) * time.Second)
		}

		if step.UserRequest == "" {
			continue
		}

		completion, err := locatr.Locate(step.UserRequest)
		totalCost += completion.CalculateCost(3.0, 15.0)
		if err != nil {
			logger.Error("Error getting locator for step", "stepName", step.Name, "error", err)
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

		pwElement = page.Locator(completion.Locators[0])
		isSameElement := false
		var compareErr error
		for _, expectedLocator := range step.ExpectedLocatrs {
			isSameElement, compareErr = locatr.Compare(expectedLocator, completion.Locators[0])
			if compareErr != nil {
				logger.Error("Error comparing locators", "stepName", step.Name, "error", compareErr, "generatedLocator", completion.Locators[0])
				continue
			}
			if isSameElement {
				logger.Error("Match found", "stepName", step.Name, "expected", expectedLocator, "generated", completion.Locators[0])
				break
			} else {
				logger.Error("Match not found", "stepName", step.Name, "expected", expectedLocator, "generated", completion.Locators[0])
			}
		}
		if compareErr != nil {
			logger.Error("Error comparing locators", "stepName", step.Name, "error", compareErr, "generatedLocator", completion.Locators[0])
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           false,
				GeneratedLocatrs: completion.Locators,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            compareErr.Error(),
			})
			continue
		}

		if isSameElement {
			logger.Info("Step finished successfully", "stepName", step.Name)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           true,
				GeneratedLocatrs: completion.Locators,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "",
			})
		} else {
			logger.Error("Expected locators do not match", "stepName", step.Name, "generatedLocatrs", completion.Locators)
			results = append(results, evalResult{
				Url:              eval.Url,
				UserRequest:      step.UserRequest,
				Passed:           false,
				GeneratedLocatrs: completion.Locators,
				ExpectedLocatrs:  step.ExpectedLocatrs,
				Error:            "All generated locators do not match expected locators",
			})
			continue
		}
	}

	fmt.Println("Total cost (USD): ", totalCost)
	return results
}

func readYamlFile(filePath string, logger *slog.Logger) (*evalConfigYaml, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error("Error reading file", "filePath", filePath, "error", err)
		return nil, err
	}

	var eval evalConfigYaml
	err = yaml.Unmarshal(yamlFile, &eval)
	if err != nil {
		logger.Error("Error unmarshalling YAML file", "filePath", filePath, "error", err)
		return nil, err
	}

	logger.Info("Successfully read and unmarshalled YAML file", "filePath", filePath)
	return &eval, nil
}

func writeEvalResultToCsv(results []evalResult, fileName string, logger *slog.Logger) {
	file, err := os.Create(fileName)
	if err != nil {
		logger.Error("Error creating CSV file", "fileName", fileName, "error", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Url", "UserRequest", "Passed", "GeneratedLocatrs", "ExpectedLocatrs", "Error"}
	err = writer.Write(header)
	if err != nil {
		logger.Error("Error writing header to CSV file", "fileName", fileName, "error", err)
		return
	}

	for _, result := range results {
		row := []string{
			result.Url,
			result.UserRequest,
			fmt.Sprintf("%t", result.Passed),
			strings.Join(result.GeneratedLocatrs, ","),
			strings.Join(result.ExpectedLocatrs, ","),
			result.Error,
		}
		err = writer.Write(row)
		if err != nil {
			logger.Error("Error writing row to CSV file", "fileName", fileName, "error", err, "row", row)
			return
		}
	}

	err = writer.Error()
	if err != nil {
		logger.Error("Error occurred during writing the CSV file", "fileName", fileName, "error", err)
	}
}

func main() {
	evalFolderPath := flag.String("evalFolder", DefaultEvalFolder, "Path to folder with eval files")
	runOnly := flag.String("runOnly", "", "Run only the specified eval file")
	mode := flag.String("mode", "dom-analysis", "Mode to run the eval in")
	flag.Parse()

	logFile, err := os.Create("locatr-eval-logs.jsonl")
	if err != nil {
		panic("failed to create log file")
	}
	defer logFile.Close()
	logger := logging.NewLogger(slog.LevelDebug, logFile)

	var evalFiles []string
	var evalYamlPath string = DefaultEvalFolder

	if *runOnly == "" {
		if *evalFolderPath != "" {
			evalYamlPath = *evalFolderPath
		}
		evalFiles, err := getAllYamlFiles(evalYamlPath, logger)
		if err != nil {
			logger.Error("Error retrieving YAML files", "folder", evalYamlPath, "error", err)
			return
		}
		if len(evalFiles) == 0 {
			logger.Error("No YAML files found in folder", "folder", evalYamlPath)
			return
		}
	} else {
		evalFiles = []string{*runOnly}
	}
	if *mode != "dom-analysis" && *mode != "visual-analysis" {
		logger.Error("Invalid mode", "mode", *mode)
		return
	}

	pw, err := playwright.Run()
	if err != nil {
		logger.Error("Error running playwright", "error", err)
		return
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		logger.Error("Error launching browser", "error", err)
		return
	}
	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		BypassCSP: playwright.Bool(true),
	})
	if err != nil {
		logger.Error("Error creating context", "error", err)
		return
	}

	for _, evalFile := range evalFiles {
		eval, err := readYamlFile(fmt.Sprintf("%s/%s", evalYamlPath, evalFile), logger)
		if err != nil {
			logger.Error("Error reading YAML file, skipping", "file", evalFile, "error", err)
			continue
		}

		logger.Info("Running eval", "evalName", eval.Name)

		results := runEval(browserContext, eval, *mode, logger)
		if results != nil {
			fileName := fmt.Sprintf("%s-%s.csv", evalFile, *mode)
			writeEvalResultToCsv(results, fileName, logger)
			logger.Info("Eval results written", "evalName", eval.Name, "file", fileName)
		}

		logger.Info("Eval finished", "evalName", eval.Name)
	}

	err = browser.Close()
	if err != nil {
		logger.Error("Error closing browser", "error", err)
	}
}
