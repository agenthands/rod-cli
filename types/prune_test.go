package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func assertPrune(t *testing.T, in, want string) {
	t.Helper()
	var n, w yaml.Node
	if err := yaml.Unmarshal([]byte(in), &n); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(want), &w); err != nil {
		t.Fatal(err)
	}
	// yaml.v3 cannot emit the compact literal indent, so compare both through one encoder.
	got, _ := yaml.Marshal(pruneInteractive(&n))
	exp, _ := yaml.Marshal(&w)
	if string(got) != string(exp) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, exp)
	}
}

// Formerly TestPruneInteractiveKeepsControlsAndAncestors: unnamed list/listitem
// wrappers are now flattened; the main landmark stays.
func TestPruneInteractiveFlattensUnnamedWrappers(t *testing.T) {
	assertPrune(t,
		"- main:\n  - paragraph: hello\n  - list:\n    - listitem:\n      - link \"Docs\" [ref=s1e4]\n- contentinfo: footer text",
		"- main:\n  - link \"Docs\" [ref=s1e4]\n")
}

func TestPruneInteractiveKeepsNamedGroup(t *testing.T) {
	assertPrune(t,
		"- group \"Payment\":\n  - generic:\n    - radio \"Card\" [ref=s1e2]\n",
		"- group \"Payment\":\n  - radio \"Card\" [ref=s1e2]\n")
}

func TestPruneInteractiveFlattensMultiChildWrapperKeepsSiblings(t *testing.T) {
	assertPrune(t,
		"- list:\n  - link \"A\" [ref=s1e1]\n  - link \"B\" [ref=s1e2]\n- button \"C\" [ref=s1e3]\n",
		"- link \"A\" [ref=s1e1]\n- link \"B\" [ref=s1e2]\n- button \"C\" [ref=s1e3]\n")
}
