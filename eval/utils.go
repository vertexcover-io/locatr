package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/playwrightLocatr"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"gopkg.in/yaml.v3"
)

func getAllYamlFiles(folder string) ([]string, error) {
	files, err := os.ReadDir(folder)
	if err != nil {
		logger.Logger.Error("Failed to read directory", "folder", folder, "error", err)
		return nil, err
	}

	var res []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") {
			res = append(res, file.Name())
		}
	}
	logger.Logger.Info("Successfully retrieved YAML files",
		"folder", folder, "fileCount", len(res))
	return res, nil
}
func contains(locatrs []string, loc string) bool {
	for _, l := range locatrs {
		if l == loc {
			return true
		}
	}
	return false
}

func compareSlices(yamlLocatrs []string, locatrs []string) bool {
	for _, loc := range yamlLocatrs {
		if contains(locatrs, loc) {
			return true
		}
	}
	return false
}
func readYamlFile(filePath string) (*evalConfigYaml, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		logger.Logger.Error("Error reading file", "filePath", filePath, "error", err)
		return nil, err
	}

	var eval evalConfigYaml
	err = yaml.Unmarshal(yamlFile, &eval)
	if err != nil {
		logger.Logger.Error("Error unmarshalling YAML file", "filePath", filePath, "error", err)
		return nil, err
	}

	logger.Logger.Info("Successfully read and unmarshalled YAML file", "filePath", filePath)
	return &eval, nil
}

func getLocatrFromYamlConfig(evalConfig *evalConfigYaml, page playwright.Page) *playwrightLocatr.PlaywrightLocator {
	locatrOptions := locatr.BaseLocatrOptions{}
	if evalConfig.Config.UseCache {
		locatrOptions.UseCache = true
	}
	if evalConfig.Config.CachePath != "" {
		locatrOptions.CachePath = evalConfig.Config.CachePath
	}
	if evalConfig.Config.ResultsFilePath != "" {
		locatrOptions.ResultsFilePath = evalConfig.Config.ResultsFilePath
	}
	if evalConfig.Config.UseReRank {
		reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))
		locatrOptions.ReRankClient = reRankClient
	}
	return playwrightLocatr.NewPlaywrightLocatr(page, locatrOptions)
}

func writeEvalResultToCsv(results []evalResult, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		logger.Logger.Error("Error creating CSV file", "fileName", fileName, "error", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Url", "UserRequest", "Passed", "GeneratedLocatrs", "ExpectedLocatrs", "Error"}
	err = writer.Write(header)
	if err != nil {
		logger.Logger.Error("Error writing header to CSV file", "fileName", fileName, "error", err)
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
			logger.Logger.Error("Error writing row to CSV file", "fileName", fileName, "error", err, "row", row)
			return
		}
	}

	err = writer.Error()
	if err != nil {
		logger.Logger.Error("Error occurred during writing the CSV file", "fileName", fileName, "error", err)
	}
}
