package plugin

import (
	"context"

	"github.com/dop251/goja"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// LifecycleEmitter defines the interface for browser lifecycle hooks
type LifecycleEmitter interface {
	OnRequest(e *proto.NetworkRequestWillBeSent)
	OnResponse(e *proto.NetworkResponseReceived)
	OnLoad(e *proto.PageLoadEventFired)
	OnDOMNodeInserted(e *proto.DOMChildNodeInserted)
}

// BindLifecycle attaches CDP event listeners to the active page and forwards them to JS functions
func (e *PluginEngine) BindLifecycle(ctx context.Context, page *rod.Page) {
	if e.vm != nil {
		e.vm.Set("api", NewPluginAPI(page))
	}

	// Enable the CDP DOM domain and fetch the full document tree so the backend
	// emits DOMChildNodeInserted events to the onDOMNodeInserted hook. Without an
	// enabled DOM domain Chrome never sends childNodeInserted, so the hook would
	// otherwise be wired but silent. The document is (re)fetched on each page load
	// so node tracking covers freshly navigated pages.
	fullDepth := -1
	_ = proto.DOMEnable{}.Call(page)
	_, _ = proto.DOMGetDocument{Depth: &fullDepth}.Call(page)

	go page.EachEvent(func(ev *proto.NetworkRequestWillBeSent) {
		e.invokeJSFunc("onRequest", ev)
	}, func(ev *proto.NetworkResponseReceived) {
		e.invokeJSFunc("onResponse", ev)
	}, func(ev *proto.PageLoadEventFired) {
		d := -1
		_, _ = proto.DOMGetDocument{Depth: &d}.Call(page)
		e.invokeJSFunc("onLoad", ev)
	}, func(ev *proto.DOMChildNodeInserted) {
		e.invokeJSFunc("onDOMNodeInserted", ev)
	})()
}

func (e *PluginEngine) invokeJSFunc(funcName string, arg interface{}) {
	if e.vm == nil {
		return
	}

	fnObj := e.vm.Get(funcName)
	if fnObj == nil {
		return
	}

	if call, ok := goja.AssertFunction(fnObj); ok {
		_, _ = call(goja.Undefined(), e.vm.ToValue(arg))
	}
}
