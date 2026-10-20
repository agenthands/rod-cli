package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/agenthands/rod-cli/types/js"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/gson"
)

var errNotDefined = errors.New("ReferenceError: snapshotEngine is not defined")

// stubSnapshotSeams swaps pageEval/pageInfo for the test and restores them.
func stubSnapshotSeams(t *testing.T, eval func(js string) (*proto.RuntimeRemoteObject, error)) {
	t.Helper()
	origEval, origInfo := pageEval, pageInfo
	t.Cleanup(func() { pageEval, pageInfo = origEval, origInfo })
	pageEval = func(_ *rod.Page, script string, _ ...any) (*proto.RuntimeRemoteObject, error) { return eval(script) }
	pageInfo = func(*rod.Page) (*proto.TargetTargetInfo, error) {
		return &proto.TargetTargetInfo{URL: "about:blank", Title: "t"}, nil
	}
}

func yamlResult(s string) *proto.RuntimeRemoteObject {
	return &proto.RuntimeRemoteObject{Value: gson.New(s)}
}

func TestSnapshotInjectsEngineOnceThenRetrySucceeds(t *testing.T) {
	injects, injected := 0, false
	stubSnapshotSeams(t, func(script string) (*proto.RuntimeRemoteObject, error) {
		if script == js.InjectEngine {
			injects++
			injected = true
			return yamlResult(""), nil
		}
		if !injected {
			return nil, errNotDefined
		}
		return yamlResult(`- button "Go" [ref=s2e1]`), nil
	})
	snap, err := BuildSnapshot(nil, SnapshotOpts{Gen: 2})
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if injects != 1 {
		t.Fatalf("injects = %d, want 1", injects)
	}
	if want := `button "Go" [ref=s2e1]`; !strings.Contains(snap.String(), want) {
		t.Fatalf("snapshot missing %q:\n%s", want, snap.String())
	}
}

func TestSnapshotAlwaysNotDefinedInjectsExactlyOnce(t *testing.T) {
	injects := 0
	stubSnapshotSeams(t, func(script string) (*proto.RuntimeRemoteObject, error) {
		if script == js.InjectEngine {
			injects++
			return yamlResult(""), nil
		}
		return nil, errNotDefined
	})
	if _, err := BuildSnapshot(nil, SnapshotOpts{Gen: 2}); err == nil {
		t.Fatal("BuildSnapshot succeeded, want error")
	}
	if injects != 1 {
		t.Fatalf("injects = %d, want 1", injects)
	}
}
