package locatr

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type MockPlugin struct{}

func (m *MockPlugin) evaluateJsFunction(js string) (string, error) {
	return "", nil
}

func (m *MockPlugin) evaluateJsScript(js string) error {
	return nil
}

type MockLlmClient struct{}

func (m *MockLlmClient) ChatCompletion(prompt string) (*chatCompletionResponse, error) {
	return nil, nil
}

func TestAddCachedLocatrs(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true}
	baseLocatr := NewBaseLocatr(mockPlugin, mockLlmClient, options)

	tests := []struct {
		url        string
		locatrName string
		locatrs    []string
		expected   map[string][]cachedLocatrsDto
	}{
		{
			url:        "http://example.com",
			locatrName: "testLocator",
			locatrs:    []string{"locator1"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
				},
			},
		},
		{
			url:        "http://example.com",
			locatrName: "anotherLocator",
			locatrs:    []string{"locator2"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
					{LocatrName: "anotherLocator", Locatrs: []string{"locator2"}},
				},
			},
		},
		{
			url:        "http://example.com",
			locatrName: "testLocator",
			locatrs:    []string{"locator3"},
			expected: map[string][]cachedLocatrsDto{
				"http://example.com": {
					{LocatrName: "testLocator", Locatrs: []string{"locator1", "locator3"}},
					{LocatrName: "anotherLocator", Locatrs: []string{"locator2"}},
				},
			},
		},
	}

	for _, tt := range tests {
		baseLocatr.addCachedLocatrs(tt.url, tt.locatrName, tt.locatrs)
		if !reflect.DeepEqual(baseLocatr.cachedLocatrs, tt.expected) {
			t.Errorf("addCachedLocatrs() = %v, want %v", baseLocatr.cachedLocatrs, tt.expected)
		}
	}
}
func TestInitilizeState(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, CachePath: "test_cache.json"}
	baseLocatr := NewBaseLocatr(mockPlugin, mockLlmClient, options)

	// Test when cache is successfully loaded
	err := os.WriteFile(options.CachePath, []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}
	defer os.Remove(options.CachePath)

	baseLocatr.initializeState()
	if !baseLocatr.initialized {
		t.Errorf("Expected initilized to be true, got false")
	}
}

func TestLoadLocatorsCache(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true, CachePath: "test_cache.json"}
	baseLocatr := NewBaseLocatr(mockPlugin, mockLlmClient, options)

	// Test loading a valid cache file
	err := os.WriteFile(options.CachePath, []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}
	defer os.Remove(options.CachePath)

	err = baseLocatr.loadLocatorsCache(options.CachePath)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := map[string][]cachedLocatrsDto{
		"http://example.com": {
			{LocatrName: "testLocator", Locatrs: []string{"locator1"}},
		},
	}
	if !reflect.DeepEqual(baseLocatr.cachedLocatrs, expected) {
		t.Errorf("Expected %v, got %v", expected, baseLocatr.cachedLocatrs)
	}

	// Test loading an invalid cache file
	err = os.WriteFile(options.CachePath, []byte(`invalid json`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test cache file: %v", err)
	}

	err = baseLocatr.loadLocatorsCache(options.CachePath)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Test loading a non-existent cache file
	err = baseLocatr.loadLocatorsCache("non_existent_cache.json")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
func TestWriteLocatorsToCache(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache.json")

	// Test writing to a valid cache path
	cacheData := []byte(`{"http://example.com":[{"locatr_name":"testLocator","locatrs":["locator1"]}]}`)
	err := writeLocatorsToCache(cachePath, cacheData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file content
	content, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}
	if !reflect.DeepEqual(content, cacheData) {
		t.Errorf("Expected %s, got %s", cacheData, content)
	}

	// Test writing to a non-existent directory
	nonExistentDir := filepath.Join(tempDir, "non_existent_dir", "test_cache.json")
	err = writeLocatorsToCache(nonExistentDir, cacheData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file content
	content, err = os.ReadFile(nonExistentDir)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}
	if !reflect.DeepEqual(content, cacheData) {
		t.Errorf("Expected %s, got %s", cacheData, content)
	}

	// Test handling errors when creating directories or files
	invalidPath := string([]byte{0})
	err = writeLocatorsToCache(invalidPath, cacheData)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetLocatrsFromState(t *testing.T) {
	mockPlugin := &MockPlugin{}
	mockLlmClient := &MockLlmClient{}
	options := BaseLocatrOptions{UseCache: true}
	baseLocatr := NewBaseLocatr(mockPlugin, mockLlmClient, options)

	testUrl := "https://example.com"
	baseLocatr.cachedLocatrs = map[string][]cachedLocatrsDto{
		testUrl: {
			{
				LocatrName: "test_key",
				Locatrs:    []string{"loc1", "loc2", "loc3"},
			},
			{
				LocatrName: "another_key",
				Locatrs:    []string{"loc4", "loc5"},
			},
		},
		"https://another-example.com": {
			{
				LocatrName: "test_key",
				Locatrs:    []string{"loc6", "loc7"},
			},
		},
	}

	locatrs, err := baseLocatr.getLocatrsFromState("test_key", testUrl)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := []string{"loc1", "loc2", "loc3"}
	if len(locatrs) != len(expected) {
		t.Errorf("Expected %v locatrs, got %v", len(expected), len(locatrs))
	}
	for i, v := range expected {
		if locatrs[i] != v {
			t.Errorf("Expected locatr %v at position %v, got %v", v, i, locatrs[i])
		}
	}

	locatrs, err = baseLocatr.getLocatrsFromState("non_existing_key", testUrl)
	if err == nil {
		t.Error("Expected error for non-existing key, got nil")
	}
	if err.Error() != "key not found" {
		t.Errorf("Expected 'key not found' error, got %v", err.Error())
	}
	if locatrs != nil {
		t.Errorf("Expected nil locatrs for non-existing key, got %v", locatrs)
	}

	locatrs, err = baseLocatr.getLocatrsFromState("test_key", "https://non-existing-url.com")
	if err == nil {
		t.Error("Expected error for non-existing URL, got nil")
	}
	if err.Error() != "key not found" {
		t.Errorf("Expected 'key not found' error, got %v", err.Error())
	}
	if locatrs != nil {
		t.Errorf("Expected nil locatrs for non-existing URL, got %v", locatrs)
	}

	locatrs, err = baseLocatr.getLocatrsFromState("test_key", "https://another-example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected = []string{"loc6", "loc7"}
	if len(locatrs) != len(expected) {
		t.Errorf("Expected %v locatrs, got %v", len(expected), len(locatrs))
	}
	for i, v := range expected {
		if locatrs[i] != v {
			t.Errorf("Expected locatr %v at position %v, got %v", v, i, locatrs[i])
		}
	}
}
