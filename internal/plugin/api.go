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

// GetLocalStorage returns the current page's window.localStorage as a
// key/value object. It mirrors GetCookies/GetSnapshot: when no page is
// bound it returns (nil, nil). The JS arrow function iterates every
// localStorage key into a plain object, which rod/gson hands back as a
// map[string]interface{} via Val().
func (a *PluginAPI) GetLocalStorage() (interface{}, error) {
	if a.page == nil {
		return nil, nil
	}
	result, err := a.page.Eval(`() => {
		var out = {};
		for (var i = 0; i < localStorage.length; i++) {
			var k = localStorage.key(i);
			out[k] = localStorage.getItem(k);
		}
		return out;
	}`)
	if err != nil {
		return nil, err
	}
	return result.Value.Val(), nil
}
