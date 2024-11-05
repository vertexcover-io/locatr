package locatr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (l *BaseLocatr) addCachedLocatrs(url string, locatrName string, locatrs []string) {
	if _, ok := l.cachedLocatrs[url]; !ok {
		l.logger.Debug(fmt.Sprintf("Domain %s not found in cache... Creating new cachedLocatrsDto", url))
		l.cachedLocatrs[url] = []cachedLocatrsDto{}
	}
	found := false
	for i, v := range l.cachedLocatrs[url] {
		if v.LocatrName == locatrName {
			l.logger.Debug(fmt.Sprintf("Found locatr %s in cache... Updating locators", locatrName))
			l.cachedLocatrs[url][i].Locatrs = GetUniqueStringArray(append(l.cachedLocatrs[url][i].Locatrs, locatrs...))
			return
		}
	}
	if !found {
		l.logger.Debug(fmt.Sprintf("Locatr %s not found in cache... Creating new locatr", locatrName))
		l.cachedLocatrs[url] = append(l.cachedLocatrs[url], cachedLocatrsDto{LocatrName: locatrName, Locatrs: locatrs})
	}
}
func (l *BaseLocatr) getLocatrsFromState(key string, currentUrl string) ([]string, error) {
	if locatrs, ok := l.cachedLocatrs[currentUrl]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				l.logger.Debug(fmt.Sprintf("Key %s found in cache", key))
				return v.Locatrs, nil
			}
		}
	}
	l.logger.Debug(fmt.Sprintf("Key %s not found in cache", key))
	return nil, errors.New("key not found")
}
func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		l.logger.Debug(fmt.Sprintf("Cache file not found: %v", err))
		return nil // ignore this error for now
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache file (%s): %v", cachePath, err)
	}
	err = json.Unmarshal(byteValue, &l.cachedLocatrs)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache file (%s): %v", cachePath, err)
	}
	return nil
}
func writeLocatorsToCache(cachePath string, cacheString []byte) error {
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(cachePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(cacheString); err != nil {
		return fmt.Errorf("failed to write cache: %v", err)
	}

	return nil
}
func (l *BaseLocatr) initializeState() {
	if l.initialized || !l.options.UseCache {
		l.logger.Debug("Cache disabled or already initialized")
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to load cache: %v", err))
		return
	}
	l.logger.Debug("Cache loaded successfully")
	l.initialized = true
}
