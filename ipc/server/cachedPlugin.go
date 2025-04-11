package main

import (
	"context"

	"github.com/vertexcover-io/locatr/pkg/types"
)

type cachedPlugin struct {
	plugin         types.PluginInterface
	minifiedDOM    *types.DOM
	currentContext string
}

func NewCachedPlugin(plugin types.PluginInterface) *cachedPlugin {
	return &cachedPlugin{plugin: plugin}
}

func (p *cachedPlugin) GetCurrentContext(ctx context.Context) (*string, error) {
	if p.currentContext != "" {
		return &p.currentContext, nil
	}
	currentContext, err := p.plugin.GetCurrentContext(ctx)
	if err != nil {
		return nil, err
	}
	p.currentContext = *currentContext
	return currentContext, nil
}

func (p *cachedPlugin) GetMinifiedDOM(ctx context.Context) (*types.DOM, error) {
	if p.minifiedDOM != nil {
		return p.minifiedDOM, nil
	}
	dom, err := p.plugin.GetMinifiedDOM(ctx)
	if err != nil {
		return nil, err
	}
	p.minifiedDOM = dom
	return dom, nil
}

func (p *cachedPlugin) ExtractFirstUniqueID(ctx context.Context, fragment string) (string, error) {
	return p.plugin.ExtractFirstUniqueID(ctx, fragment)
}

func (p *cachedPlugin) IsLocatorValid(ctx context.Context, locator string) (bool, error) {
	return p.plugin.IsLocatorValid(ctx, locator)
}

func (p *cachedPlugin) SetViewportSize(ctx context.Context, width, height int) error {
	return p.plugin.SetViewportSize(ctx, width, height)
}

func (p *cachedPlugin) TakeScreenshot(ctx context.Context) ([]byte, error) {
	return p.plugin.TakeScreenshot(ctx)
}

func (p *cachedPlugin) GetElementLocators(ctx context.Context, location *types.Location) ([]string, error) {
	return p.plugin.GetElementLocators(ctx, location)
}

func (p *cachedPlugin) GetElementLocation(ctx context.Context, locator string) (*types.Location, error) {
	return p.plugin.GetElementLocation(ctx, locator)
}
