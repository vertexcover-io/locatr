package plugins

import (
	"context"
	"errors"

	"github.com/vertexcover-io/locatr/pkg/internal/html"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/internal/xml"
	"github.com/vertexcover-io/locatr/pkg/types"
)

var (
	ErrModeNotSupported = errors.New("raw-text plugin only supports DOM evalutaion mode")
)

type PageType int

const (
	HTMLPageType PageType = iota
	XMLPageType
)

type Platform int

const (
	AndroidPlatform Platform = iota
	IosPlatform
	WebPlatform
)

const (
	androidText = "android"
	iosText     = "ios"
	webText     = "web"
	invalidText = "n/a"
)

func (platform Platform) Str() string {
	switch platform {
	case AndroidPlatform:
		return androidText
	case IosPlatform:
		return iosText
	case WebPlatform:
		return webText
	default:
		return invalidText
	}
}

type rawTextPlugin struct {
	content  string
	pageType PageType
	platform Platform
}

func NewRawTextPlugin(content string, pageType PageType, platform Platform) *rawTextPlugin {
	return &rawTextPlugin{
		content:  content,
		pageType: pageType,
		platform: platform,
	}
}

func (plugin *rawTextPlugin) minifyHTML() (*types.DOM, error) {
	pageSource := plugin.content

	eSpec, err := html.MinifySource(pageSource)
	if err != nil {
		return nil, err
	}
	locatrMap, err := html.CreateLocatorMap(pageSource)
	if err != nil {
		return nil, err
	}

	dom := &types.DOM{
		RootElement: eSpec,
		Metadata: &types.DOMMetadata{
			LocatorType: types.XPathType,
			LocatorMap:  locatrMap,
		},
	}
	return dom, nil
}

func (plugin *rawTextPlugin) minifyXML() (*types.DOM, error) {
	pageSource := plugin.content
	platform := plugin.platform.Str()

	eSpec, err := xml.MinifySource(pageSource, platform)
	if err != nil {
		return nil, err
	}
	locatrMap, err := xml.CreateLocatorMap(pageSource, platform)
	if err != nil {
		return nil, err
	}
	dom := &types.DOM{
		RootElement: eSpec,
		Metadata: &types.DOMMetadata{
			LocatorType: types.XPathType,
			LocatorMap:  locatrMap,
		},
	}
	return dom, nil
}

func (plugin *rawTextPlugin) GetMinifiedDOM(ctx context.Context) (*types.DOM, error) {
	if plugin.pageType == HTMLPageType {
		return plugin.minifyHTML()
	}
	return plugin.minifyXML()
}

func (plugin *rawTextPlugin) ExtractFirstUniqueHTMLID(ctx context.Context, fragment string) (string, error) {
	if plugin.pageType == HTMLPageType {
		return utils.ExtractFirstUniqueHTMLID(fragment)
	}
	return utils.ExtractFirstUniqueXMLID(fragment)
}

func (plugin *rawTextPlugin) IsLocatorValid(ctx context.Context, locator string) (bool, error) {
	pageSource := plugin.content

	if plugin.pageType == HTMLPageType {
		return html.IsValidXPath(locator, pageSource)
	}
	return xml.IsValidXPath(locator, pageSource)
}

func (plugin *rawTextPlugin) GetCurrentContext(ctx context.Context) (*string, error) {
	return nil, ErrModeNotSupported
}

func (plugin *rawTextPlugin) SetViewportSize(ctx context.Context, width, height int) error {
	return ErrModeNotSupported
}

func (plugin *rawTextPlugin) TakeScreenshot(ctx context.Context) ([]byte, error) {
	return nil, ErrModeNotSupported
}

func (plugin *rawTextPlugin) GetElementLocators(ctx context.Context, location *types.Location) ([]string, error) {
	return nil, ErrModeNotSupported
}

func (plugin *rawTextPlugin) GetElementLocation(ctx context.Context, locator string) (*types.Location, error) {
	return nil, ErrModeNotSupported
}
