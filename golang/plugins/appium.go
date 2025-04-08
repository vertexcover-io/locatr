package plugins

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vertexcover-io/locatr/golang/internal/appium"
	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/internal/utils"
	"github.com/vertexcover-io/locatr/golang/internal/xml"
	"github.com/vertexcover-io/locatr/golang/logging"
	"github.com/vertexcover-io/locatr/golang/types"
)

// appiumPlugin encapsulates browser automation functionality using the Appium client.
//
// Attributes:
//   - PlatformName: The name of the platform (e.g. "android", "ios")
type appiumPlugin struct {
	client             *appium.Client
	originalResolution *types.Resolution
	targetResolution   *types.Resolution
	// Name of the platform (e.g. "android", "ios")
	PlatformName string
}

// NewAppiumPlugin initializes a new plugin instance from appium session id or capabilities by creating a new session.
//
// Parameters:
//   - serverUrl: The URL of the Appium server
//   - sessionIdOrCapabilities: The ID (string) of the Appium session or capabilities (map[string]any) to create a new session
func NewAppiumPlugin(serverUrl string, sessionIdOrCapabilities any) (*appiumPlugin, error) {
	var (
		sessionId string
		err       error
	)
	switch v := sessionIdOrCapabilities.(type) {
	case string:
		sessionId = v
	case map[string]any:
		sessionId, err = appium.NewSession(serverUrl, v)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("expected sessionId (string) or capabilities (map[string]any), got %T", sessionIdOrCapabilities)
	}

	client, err := appium.NewClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}

	caps, err := client.GetCapabilities()
	if err != nil {
		return nil, err
	}
	platFormName := caps.Value.PlatformName
	if platFormName == "" {
		platFormName = caps.Value.Cap.PlatformName
	}
	platFormName = strings.ToLower(platFormName)
	if platFormName != "android" && platFormName != "ios" {
		return nil, fmt.Errorf("'%s' platform is currently not supported", platFormName)
	}

	plugin := &appiumPlugin{client: client, PlatformName: platFormName}
	_, _ = plugin.TakeScreenshot() // This will set the original resolution
	return plugin, nil
}

// evaluateJSExpression executes a JavaScript expression in the context of the current page.
// If the script is not attached, it will be attached first.
//
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the expression
func (plugin *appiumPlugin) evaluateJSExpression(expression string, args ...any) (any, error) {
	// Check if script is already attached
	isAttached, err := plugin.client.ExecuteScript("return window.locatrScriptAttached === true", []any{})
	if err != nil || isAttached == nil || !isAttached.(bool) {
		_, err := plugin.client.ExecuteScript(constants.JS_CONTENT, []any{})
		if err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
	}

	result, err := plugin.client.ExecuteScript(fmt.Sprintf("return %s", expression), args)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// minifyHTML retrieves the minified DOM from the current page.
func (plugin *appiumPlugin) minifyHTML() (*types.DOM, error) {
	result, err := plugin.evaluateJSExpression("minifyHTML()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get minified DOM: %v", err)
	}

	rootElement, err := utils.ParseElementSpec(result)
	if err != nil {
		return nil, err
	}

	result, err = plugin.evaluateJSExpression("createLocatorMap()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get locator map: %v", err)
	}

	locatorMap, err := utils.ParseLocatorMap(result)
	if err != nil {
		return nil, err
	}

	dom := &types.DOM{
		RootElement: rootElement,
		Metadata: &types.DOMMetadata{
			LocatorType: types.CssSelectorType, LocatorMap: locatorMap,
		},
	}
	return dom, nil
}

// minifyXML retrieves the minified DOM from the current page.
func (plugin *appiumPlugin) minifyXML() (*types.DOM, error) {
	pageSource, err := plugin.client.GetPageSource()
	if err != nil {
		return nil, err
	}

	eSpec, err := xml.MinifySource(pageSource, plugin.PlatformName)
	if err != nil {
		return nil, err
	}
	locatrMap, err := xml.MapElementsToJson(pageSource, plugin.PlatformName)
	if err != nil {
		return nil, err
	}
	dom := &types.DOM{
		RootElement: eSpec,
		Metadata: &types.DOMMetadata{
			LocatorType: types.XPathType, LocatorMap: locatrMap,
		},
	}
	return dom, nil
}

// GetCurrentContext retrieves the current context of the plugin.
func (plugin *appiumPlugin) GetCurrentContext() (*string, error) {
	if plugin.PlatformName != "android" {
		return nil, fmt.Errorf("cannot read platform '%s' current context", plugin.PlatformName)
	}
	if currentActivity, err := plugin.client.GetCurrentActivity(); err != nil {
		return &currentActivity, nil
	}
	return nil, errors.New("no context found")
}

// GetMinifiedDOM retrieves the minified DOM from the current page.
func (plugin *appiumPlugin) GetMinifiedDOM() (*types.DOM, error) {
	if plugin.client.IsWebView() {
		return plugin.minifyHTML()
	}
	return plugin.minifyXML()
}

// ExtractFirstUniqueID extracts the first unique ID from the given fragment.
func (plugin *appiumPlugin) ExtractFirstUniqueID(fragment string) (string, error) {
	if plugin.client.IsWebView() {
		return utils.ExtractFirstUniqueHTMLID(fragment)
	}
	return utils.ExtractFirstUniqueXMLID(fragment)
}

// IsLocatorValid verifies if the given locator is valid.
func (plugin *appiumPlugin) IsLocatorValid(locator string) (bool, error) {
	var locatrType string
	if plugin.client.IsWebView() {
		locatrType = "css selector"
	} else {
		locatrType = "xpath"
	}
	_, err := plugin.client.FindElement(locatrType, locator)
	return err == nil, err
}

// SetViewportSize sets the viewport size.
func (plugin *appiumPlugin) SetViewportSize(width, height int) error {
	// We don't actually set the viewport size, we just
	// store the resolution for scaling the screenshot
	plugin.targetResolution = &types.Resolution{Width: width, Height: height}
	return nil
}

// TakeScreenshot captures a screenshot of the current viewport.
func (plugin *appiumPlugin) TakeScreenshot() ([]byte, error) {
	base64Image, err := plugin.client.ExecuteScript("mobile: viewportScreenshot", []any{})
	if err != nil {
		return nil, err
	}
	imageBytes, err := base64.StdEncoding.DecodeString(base64Image.(string))
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, err
	}
	plugin.originalResolution = &types.Resolution{
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}

	if plugin.targetResolution != nil {
		scaledBytes := utils.ScaleAndPadImage(img, plugin.targetResolution)
		var buf bytes.Buffer
		if err := png.Encode(&buf, scaledBytes); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	return imageBytes, nil
}

// parseAndCalculateCenter parses the coordinates and calculates the center of the element.
func parseAndCalculateCenter(x, y, width, height string) (*types.Point, error) {
	x1, err := strconv.Atoi(x)
	if err != nil {
		return nil, fmt.Errorf("invalid x value: %v", err)
	}

	y1, err := strconv.Atoi(y)
	if err != nil {
		return nil, fmt.Errorf("invalid y value: %v", err)
	}

	x2, err := strconv.Atoi(width)
	if err != nil {
		return nil, fmt.Errorf("invalid width value: %v", err)
	}

	y2, err := strconv.Atoi(height)
	if err != nil {
		return nil, fmt.Errorf("invalid height value: %v", err)
	}

	// Calculate center
	centerX := (x1 + x2) / 2
	centerY := (y1 + y2) / 2

	return &types.Point{X: float64(centerX), Y: float64(centerY)}, nil
}

// parseBoundsAndCalculateCenter parses the bounds and calculates the center of the element.
func (plugin *appiumPlugin) parseBoundsAndCalculateCenter(bounds string) (*types.Point, error) {
	re := regexp.MustCompile(`\[(\d+),(\d+)\]\[(\d+),(\d+)\]`)
	matches := re.FindStringSubmatch(bounds)
	if len(matches) != 5 {
		return nil, fmt.Errorf("invalid bounds format: %s", bounds)
	}
	return parseAndCalculateCenter(matches[1], matches[2], matches[3], matches[4])
}

// candidate represents a candidate element and its score.
type candidate struct {
	element *types.ElementSpec
	score   float64
}

// GetElementLocators retrieves locators from a given point and scroll position on the page.
func (plugin *appiumPlugin) GetElementLocators(location *types.Location) ([]string, error) {
	// Remap the incoming location to original coordinates.
	if plugin.targetResolution != nil && plugin.originalResolution != nil {
		location.Point = *utils.RemapPoint(&location.Point, plugin.originalResolution, plugin.targetResolution)
	}

	var (
		wg            sync.WaitGroup
		searchElement func(element *types.ElementSpec)
		candidateChan = make(chan candidate, 10)
	)
	searchElement = func(element *types.ElementSpec) {
		defer wg.Done()

		var (
			err          error
			elementPoint *types.Point
			attrs        = element.Attributes
		)

		if plugin.PlatformName == "android" && attrs["bounds"] != "" {
			elementPoint, err = plugin.parseBoundsAndCalculateCenter(attrs["bounds"])
		} else if plugin.PlatformName == "ios" && attrs["visible"] == "true" {
			elementPoint, err = parseAndCalculateCenter(attrs["x"], attrs["y"], attrs["width"], attrs["height"])
		}
		if err != nil {
			logging.DefaultLogger.Error("failed to calculate center", "error", err)
		}

		if elementPoint != nil {
			score := math.Abs(location.Point.X-elementPoint.X) + math.Abs(location.Point.Y-elementPoint.Y)
			select {
			case candidateChan <- candidate{element: element, score: score}:
			default:
			}
		}

		for i := range element.Children {
			wg.Add(1)
			go searchElement(&element.Children[i])
		}
	}

	dom, err := plugin.GetMinifiedDOM()
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	go searchElement(dom.RootElement)

	go func() {
		wg.Wait()
		close(candidateChan)
	}()

	var best *candidate
	for cand := range candidateChan {
		if best == nil || cand.score < best.score {
			best = &cand
		}
	}

	if best == nil {
		return nil, errors.New("element not found at given location")
	}

	return dom.Metadata.LocatorMap[best.element.Id], nil
}

// GetElementLocation retrieves the location of the element associated with the given locator.
func (plugin *appiumPlugin) GetElementLocation(locator string) (*types.Location, error) {
	var (
		wg            sync.WaitGroup
		searchElement func(element *types.ElementSpec)
		resultChan    = make(chan *types.ElementSpec, 1)
		uniqueId      = xml.GenerateUniqueId(locator)
	)

	searchElement = func(element *types.ElementSpec) {
		defer wg.Done()

		if element.Id == uniqueId {
			select {
			case resultChan <- element:
			default:
			}
			return
		}

		for _, child := range element.Children {
			wg.Add(1)
			go searchElement(&child)
		}
	}

	dom, err := plugin.GetMinifiedDOM()
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	go searchElement(dom.RootElement)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Move the select statement into a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	select {
	case result, ok := <-resultChan:
		if !ok {
			return nil, fmt.Errorf("couldn't locate element associated with locator: '%s'", locator)
		}

		attrs := result.Attributes
		var elementPoint *types.Point

		if plugin.PlatformName == "android" && attrs["bounds"] != "" {
			elementPoint, err = plugin.parseBoundsAndCalculateCenter(attrs["bounds"])
		} else if plugin.PlatformName == "ios" && attrs["visible"] == "true" {
			elementPoint, err = parseAndCalculateCenter(
				attrs["x"], attrs["y"], attrs["width"], attrs["height"],
			)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to calculate center: %v", err)
		}

		if plugin.targetResolution != nil && plugin.originalResolution != nil {
			elementPoint = utils.RemapPointInverse(
				elementPoint, plugin.originalResolution, plugin.targetResolution,
			)
		}
		location := &types.Location{
			Point: *elementPoint,
			// Scroll position is set to 0.0, 0.0 because DOM only contains
			// what is available in the viewport.
			ScrollPosition: types.Point{X: 0.0, Y: 0.0},
		}
		return location, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout while searching for element with locator: %s", locator)
	}
}
