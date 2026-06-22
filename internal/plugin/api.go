package plugin

import (
	"github.com/go-rod/rod"
)

// PluginAPI exposes state context to the JavaScript environment
type PluginAPI struct {
	page *rod.Page
}

// NewPluginAPI creates a new API instance bound to a specific page
func NewPluginAPI(page *rod.Page) *PluginAPI {
	return &PluginAPI{page: page}
}

// GetCookies returns the cookies for the current page
func (a *PluginAPI) GetCookies() (interface{}, error) {
	if a.page == nil {
		return nil, nil
	}
	cookies, err := a.page.Browser().GetCookies()
	if err != nil {
		return nil, err
	}
	return cookies, nil
}

// GetSnapshot returns the HTML snapshot of the current page
func (a *PluginAPI) GetSnapshot() (string, error) {
	if a.page == nil {
		return "", nil
	}
	return a.page.HTML()
}
