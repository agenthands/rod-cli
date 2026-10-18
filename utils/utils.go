package utils

import (
	"github.com/agenthands/rod-cli/types/js"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// EngineWorld is the isolated world snapshotEngine lives in, so page scripts
// cannot see it.
const EngineWorld = "rodcli"

// engineGlobal returns the global object of frame's EngineWorld. Chrome reuses
// the named world per frame, so this is the world the new-document script ran in.
// ponytail: one Page.createIsolatedWorld round trip per call; cache the context
// per frame and document if snapshot latency matters.
func engineGlobal(frame *rod.Page) (*proto.RuntimeRemoteObject, error) {
	w, err := proto.PageCreateIsolatedWorld{FrameID: frame.FrameID, WorldName: EngineWorld}.Call(frame)
	if err != nil {
		return nil, err
	}
	res, err := proto.RuntimeEvaluate{Expression: "globalThis", ContextID: w.ExecutionContextID}.Call(frame)
	if err != nil {
		return nil, err
	}
	return res.Result, nil
}

// EngineEval calls fn with args in frame's EngineWorld and returns the value.
func EngineEval(frame *rod.Page, fn string, args ...any) (*proto.RuntimeRemoteObject, error) {
	g, err := engineGlobal(frame)
	if err != nil {
		return nil, err
	}
	return frame.Evaluate(rod.Eval(fn, args...).This(g))
}

// QueryEleByAria resolves selector with snapshotEngine and returns the element
// as a main-world object, so element scripts still see page globals.
func QueryEleByAria(frame *rod.Page, selector string) (*rod.Element, error) {
	g, err := engineGlobal(frame)
	if err != nil {
		return nil, err
	}
	el, err := frame.ElementByJS(rod.Eval(js.QueryEleByAria, selector).This(g))
	if err != nil {
		return nil, err
	}
	node, err := el.Describe(0, false)
	if err != nil {
		return nil, err
	}
	return frame.ElementFromNode(&proto.DOMNode{BackendNodeID: node.BackendNodeID})
}
