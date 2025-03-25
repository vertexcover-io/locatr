package rawxml

import (
	"context"
	"fmt"
	"io"

	locatr "github.com/vertexcover-io/locatr/golang"
)

type Platform string

const (
	PLATFORM_ANDROID Platform = "android"
	PLATFORM_IOS     Platform = "ios"
)

type rawtextLocatr struct {
	locatr *locatr.BaseLocatr
}

func NewRawTextLocator(
	xml io.Reader,
	platform Platform,
	opts locatr.BaseLocatrOptions,
) (*rawtextLocatr, error) {
	fc, err := io.ReadAll(xml)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	plugin, err := newRawTextPlugin(string(fc), string(platform))
	if err != nil {
		return nil, err
	}

	baseLocatr := locatr.NewBaseLocatr(plugin, opts)
	locatr := &rawtextLocatr{
		locatr: baseLocatr,
	}
	return locatr, nil
}

func (lc *rawtextLocatr) GetLocatrStr(ctx context.Context, userReq string) (*locatr.LocatrOutput, error) {
	return lc.locatr.GetLocatorStr(ctx, userReq)
}

func (lc *rawtextLocatr) GetLocatrResults() []locatr.LocatrResult {
	return lc.locatr.GetLocatrResults()
}

func (lc *rawtextLocatr) WriteResultsToFile() {
	lc.locatr.WriteLocatrResultsToFile()
}
