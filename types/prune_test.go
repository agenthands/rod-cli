package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPruneInteractiveKeepsControlsAndAncestors(t *testing.T) {
	in := "- main:\n  - paragraph: hello\n  - list:\n    - listitem:\n      - link \"Docs\" [ref=s1e4]\n- contentinfo: footer text"
	want := "- main:\n  - list:\n    - listitem:\n      - link \"Docs\" [ref=s1e4]\n"
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
