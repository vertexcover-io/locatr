package locatr

import (
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed static/*
var staticFiles embed.FS

func readStaticFile(filename string) ([]byte, error) {
	return staticFiles.ReadFile(filename)
}

func writeLocatorsToCache(cachePath string, key string, locators []string) error {
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(cachePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := append([]string{key}, locators...)
	err = writer.Write(record)
	if err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}

func readLocatorsFromCache(cachePath string, key string) ([]string, error) {
	file, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	for _, record := range records {
		if len(record) > 0 && record[0] == key {
			return record[1:], nil
		}
	}

	return nil, errors.New("key not found")
}
