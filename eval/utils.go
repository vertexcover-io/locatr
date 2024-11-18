package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr"
	"gopkg.in/yaml.v3"
)

func getAllYamlFiles(folder string) []string {
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal(err)
	}
	var res []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") {
			res = append(res, file.Name())
		}
	}
	return res
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
		log.Fatalf("Error reading file %s: %s", filePath, err)
		return nil, err

	}
	var eval evalConfigYaml
	err = yaml.Unmarshal(yamlFile, &eval)
	if err != nil {
		log.Fatalf("Error unmarshalling yaml file %s: %s", filePath, err)
		return nil, err
	}
	return &eval, nil
}
func getLocatrFromYamlConfig(evalConfig *evalConfigYaml, page playwright.Page) locatr.PlaywrightLocator {
	locatrOptions := locatr.BaseLocatrOptions{}
	if evalConfig.Config.UseCache {
		locatrOptions.UseCache = true
	}
	if evalConfig.Config.CachePath != "" {
		locatrOptions.CachePath = evalConfig.Config.CachePath
	}
	if evalConfig.Config.ResulstsFilePath != "" {
		locatrOptions.ResultsFilePath = evalConfig.Config.ResulstsFilePath
	}
	if evalConfig.Config.UseReRank {
		reRankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))
		locatrOptions.ReRankClient = reRankClient
	}
	locatrOptions.LogConfig = locatr.LogConfig{
		Level: locatr.Debug,
	}
	return *locatr.NewPlaywrightLocatr(page, locatrOptions)
}

func writeEvalResultToCsv(results []evalResult, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Error creating csv file %s: %s", fileName, err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Url", "UserRequest", "Passed", "GeneratedLocatrs", "ExpectedLocatrs", "Error"}
	err = writer.Write(header)
	if err != nil {
		log.Fatalf("Error writing header to csv file %s: %s", fileName, err)
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
			log.Fatalf("Error writing row to csv file %s: %s", fileName, err)
			return
		}
	}

	err = writer.Error()
	if err != nil {
		log.Fatalf("Error occurred during writing the csv file %s: %s", fileName, err)
	}
}
