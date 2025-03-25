package rawxml

import (
	"context"
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/minifier"
)

type rawtextPlugin struct {
	fileContent string
	platform    string
	doc         *xmlquery.Node
}

func newRawTextPlugin(fileContent, platform string) (*rawtextPlugin, error) {
	doc, err := xmlquery.Parse(strings.NewReader(fileContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse xml: %w", err)
	}
	pl := &rawtextPlugin{
		fileContent: fileContent,
		platform:    platform,
		doc:         doc,
	}
	return pl, nil
}

func (pl *rawtextPlugin) GetMinifiedDomAndLocatorMap(context.Context) (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	pageSource := pl.fileContent
	platform := pl.platform

	eSpec, err := minifier.MinifyXMLSource(pageSource, platform)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to minify source: %w", err)
	}
	locatrMap, err := minifier.MapXMLElementsToJson(pageSource, platform)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to map elements to json: %w", err)
	}

	return eSpec, locatrMap, "xpath", nil
}

func (pl *rawtextPlugin) GetCurrentContext(context.Context) string {
	if pl.platform == "android" {
		return "DUMMY_ANDROID_CONTEXT"
	}
	return ""
}

func (pl *rawtextPlugin) IsValidLocator(ctx context.Context, locatr string) (bool, error) {
	elem := xmlquery.Find(pl.doc, locatr)
	exists := len(elem) > 0
	return exists, nil
}
