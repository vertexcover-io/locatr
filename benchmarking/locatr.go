package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/playwrightLocatr"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"gopkg.in/yaml.v3"
)

var DefaultEvalFolder = "schema/eval"
var DefaultResultFolder = "schema/results/original_locatr"

type Step struct {
	UserRequest string   `yaml:"userRequest"`
	Locatrs     []string `yaml:"locatrs"`
}

type Schema struct {
	Url   string `yaml:"url"`
	Steps []Step `yaml:"steps"`
}

func getAllYamlFiles(folder string) ([]string, error) {
	files, err := os.ReadDir(folder)
	if err != nil {
		logger.Logger.Info("Failed to read directory", "folder", folder, "error", err)
		return nil, err
	}

	var res []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") {
			res = append(res, file.Name())
		}
	}
	logger.Logger.Info("Successfully retrieved YAML files", "folder", folder, "fileCount", len(res))
	return res, nil
}

func readYamlFile(filePath string) (*Schema, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		logger.Logger.Info("Error reading file", "filePath", filePath, "error", err)
		return nil, err
	}

	var eval Schema
	err = yaml.Unmarshal(yamlFile, &eval)
	if err != nil {
		logger.Logger.Info("Error unmarshalling YAML file", "filePath", filePath, "error", err)
		return nil, err
	}

	logger.Logger.Info("Successfully read and unmarshalled YAML file", "filePath", filePath)
	return &eval, nil
}

func writeResultsToYaml(results *Schema, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		logger.Logger.Info("Error creating file", "filePath", filePath, "error", err)
		return err
	}
	defer file.Close()

	yaml.NewEncoder(file).Encode(results)
	return nil
}

func getLocatr(page playwright.Page, rerank bool) *playwrightLocatr.PlaywrightLocator {
	llmClient, err := llm.NewLlmClient(
		llm.Anthropic, "claude-3-5-sonnet-20241022", os.Getenv("ANTHROPIC_API_KEY"),
	)
	if err != nil {
		logger.Logger.Info("Error creating LLM client", "error", err)
		return nil
	}
	locatrOptions := locatr.BaseLocatrOptions{LlmClient: llmClient}

	if rerank {
		reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))
		locatrOptions.ReRankClient = reRankClient
	}
	return playwrightLocatr.NewPlaywrightLocatr(page, locatrOptions)
}

func extractResultSchema(context playwright.BrowserContext, eval *Schema) (*Schema, error) {
	page, err := context.NewPage()
	if err != nil {
		logger.Logger.Info("Error creating page", "error", err)
		return nil, err
	}
	defer page.Close()

	if _, err := page.Goto(
		eval.Url,
		playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		},
	); err != nil {
		logger.Logger.Info("Error navigating to URL", "url", eval.Url, "error", err)
		return nil, err
	}

	playWrightLocatr := getLocatr(page, true)

	output_steps := make([]Step, 0)
	for _, step := range eval.Steps {
		logger.Logger.Info("Running step", "stepRequest", step.UserRequest)

		if step.UserRequest == "" {
			continue
		}

		_, err := playWrightLocatr.GetLocatr(step.UserRequest)
		if err != nil {
			logger.Logger.Info("Error getting locator for step", "stepRequest", step.UserRequest, "error", err)
			output_steps = append(output_steps, Step{
				UserRequest: step.UserRequest,
				Locatrs:     []string{},
			})
			continue
		}

		generatedResults := playWrightLocatr.GetLocatrResults()
		generatedLocatrs := generatedResults[len(generatedResults)-1].AllLocatrs
		output_steps = append(output_steps, Step{
			UserRequest: step.UserRequest,
			Locatrs:     generatedLocatrs,
		})

	}

	return &Schema{Url: eval.Url, Steps: output_steps}, nil
}

func main() {
	logger.Level.Set(slog.LevelDebug)

	err := godotenv.Load()
	if err != nil {
		logger.Logger.Error("Error loading .env file")
	}

	var evalFiles []string
	evalFiles, err = getAllYamlFiles(DefaultEvalFolder)
	if err != nil {
		logger.Logger.Info("Error retrieving YAML files", "folder", DefaultEvalFolder, "error", err)
		return
	}
	if len(evalFiles) == 0 {
		logger.Logger.Info("No YAML files found in folder", "folder", DefaultEvalFolder)
		return
	}
	logger.Logger.Info("Schema files", "files", evalFiles)

	pw, err := playwright.Run()
	if err != nil {
		logger.Logger.Info("Error running playwright", "error", err)
		return
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		logger.Logger.Info("Error launching browser", "error", err)
		return
	}
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 991,
		},
		BypassCSP: playwright.Bool(true),
	})
	if err != nil {
		logger.Logger.Info("Error creating context", "error", err)
		return
	}

	for _, evalFile := range evalFiles {
		eval, err := readYamlFile(fmt.Sprintf("%s/%s", DefaultEvalFolder, evalFile))
		logger.Logger.Info("Schema", "eval", eval)
		if err != nil {
			logger.Logger.Info("Error reading YAML file, skipping", "file", evalFile, "error", err)
			continue
		}

		logger.Logger.Info("Running eval", "evalURL", eval.Url)

		result_schema, err := extractResultSchema(context, eval)
		if err != nil {
			logger.Logger.Info("Error extracting result schema", "error", err)
			continue
		}
		if result_schema != nil {
			writeResultsToYaml(result_schema, fmt.Sprintf("%s/%s", DefaultResultFolder, evalFile))
			logger.Logger.Info("Schema results written", "evalURL", eval.Url, "file", fmt.Sprintf("%s.yaml", evalFile))
		}

		logger.Logger.Info("Schema finished", "evalURL", eval.Url)
	}

	err = browser.Close()
	if err != nil {
		logger.Logger.Info("Error closing browser", "error", err)
	}
}
