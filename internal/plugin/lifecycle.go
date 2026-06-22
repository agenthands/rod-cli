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
	go page.EachEvent(func(ev *proto.NetworkRequestWillBeSent) {
		e.invokeJSFunc("onRequest", ev)
	}, func(ev *proto.NetworkResponseReceived) {
		e.invokeJSFunc("onResponse", ev)
	}, func(ev *proto.PageLoadEventFired) {
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
