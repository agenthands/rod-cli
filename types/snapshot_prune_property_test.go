package types

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// ariaNode is the model behind one generated engine-shaped aria-YAML item.
type ariaNode struct {
	role, name, ref string
	text            string // scalar body; ignored when children/url exist
	url             string // optional `/url:` child
	children        []*ariaNode
}

var (
	propInteractive    = []string{"button", "link", "textbox", "checkbox", "radio", "combobox", "option", "menuitem", "tab", "switch", "slider", "heading"}
	propNonInteractive = []string{"generic", "list", "listitem", "paragraph", "main", "group", "navigation", "region", "img", "table", "tablist", "radiogroup", "menubar", "cell"}
)

func (n *ariaNode) interactive() bool { return slices.Contains(propInteractive, n.role) }

func (n *ariaNode) hasInteractive() bool {
	if n.interactive() {
		return true
	}
	for _, c := range n.children {
		if c.hasInteractive() {
			return true
		}
	}
	return false
}

func (n *ariaNode) walk(f func(*ariaNode)) {
	f(n)
	for _, c := range n.children {
		c.walk(f)
	}
}

func (n *ariaNode) key() string {
	k := n.role
	if n.name != "" {
		k += fmt.Sprintf(" %q", n.name)
	}
	if n.ref != "" {
		k += " [ref=" + n.ref + "]"
	}
	return k
}

func str(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v} }

// yamlNode renders n the way the engine does: a bare scalar for a leaf, else a
// single-key map whose value is text or a sequence of children.
func (n *ariaNode) yamlNode() *yaml.Node {
	if len(n.children) == 0 && n.url == "" {
		if n.text == "" {
			return str(n.key())
		}
		return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{str(n.key()), str(n.text)}}
	}
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	if n.url != "" {
		seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{str("/url"), str(n.url)}})
	}
	for _, c := range n.children {
		seq.Content = append(seq.Content, c.yamlNode())
	}
	return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{str(n.key()), seq}}
}

func ariaGen(t *rapid.T, label string, depth int, nextRef *int) *ariaNode {
	n := &ariaNode{}
	if rapid.Bool().Draw(t, label+".interactive") {
		n.role = rapid.SampledFrom(propInteractive).Draw(t, label+".role")
	} else {
		n.role = rapid.SampledFrom(propNonInteractive).Draw(t, label+".role")
	}
	if rapid.Bool().Draw(t, label+".named") {
		n.name = rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]{0,8}`).Draw(t, label+".name")
	}
	if rapid.Bool().Draw(t, label+".hasRef") {
		*nextRef++
		n.ref = fmt.Sprintf("s1e%d", *nextRef)
	}
	if rapid.Bool().Draw(t, label+".hasURL") {
		n.url = "/" + rapid.StringMatching(`[a-z]{1,6}`).Draw(t, label+".url")
	}
	n.text = rapid.SampledFrom([]string{"", "hello", "lorem ipsum", "Total due"}).Draw(t, label+".text")
	if depth > 0 {
		k := rapid.IntRange(0, 3).Draw(t, label+".kids")
		for i := range k {
			n.children = append(n.children, ariaGen(t, fmt.Sprintf("%s.%d", label, i), depth-1, nextRef))
		}
	}
	return n
}

func ariaDocGen(t *rapid.T) ([]*ariaNode, string) {
	ref := 0
	var roots []*ariaNode
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for i := range rapid.IntRange(1, 4).Draw(t, "roots") {
		n := ariaGen(t, fmt.Sprintf("r%d", i), 3, &ref)
		roots = append(roots, n)
		seq.Content = append(seq.Content, n.yamlNode())
	}
	b, err := yaml.Marshal(seq)
	if err != nil {
		t.Fatalf("marshal generated doc: %v", err)
	}
	return roots, string(b)
}

// interactiveSnapshot runs BuildSnapshot(nil, {Interactive}) over raw aria YAML
// and returns the YAML body of the rendered snapshot.
func interactiveSnapshot(t *rapid.T, raw string) string {
	origEval, origInfo := pageEval, pageInfo
	defer func() { pageEval, pageInfo = origEval, origInfo }()
	pageEval = func(_ *rod.Page, _ string, _ ...any) (*proto.RuntimeRemoteObject, error) { return yamlResult(raw), nil }
	pageInfo = func(*rod.Page) (*proto.TargetTargetInfo, error) {
		return &proto.TargetTargetInfo{URL: "about:blank"}, nil
	}
	snap, err := BuildSnapshot(nil, SnapshotOpts{Gen: 1, Interactive: true})
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	out := snap.String()
	_, body, _ := strings.Cut(out, "```yaml\n")
	body, _, _ = strings.Cut(body, "\n```")
	return body
}

func TestSnapshotPruneKeepsEveryInteractiveRef(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		roots, raw := ariaDocGen(t)
		out := interactiveSnapshot(t, raw)
		for _, r := range roots {
			r.walk(func(n *ariaNode) {
				if n.interactive() && n.ref != "" && !strings.Contains(out, "[ref="+n.ref+"]") {
					t.Fatalf("interactive %q lost\nin:\n%s\nout:\n%s", n.key(), raw, out)
				}
			})
		}
	})
}

func TestSnapshotPruneDropsRefsWithoutInteractiveDescendant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		roots, raw := ariaDocGen(t)
		out := interactiveSnapshot(t, raw)
		for _, r := range roots {
			r.walk(func(n *ariaNode) {
				if n.ref != "" && !n.hasInteractive() && strings.Contains(out, "[ref="+n.ref+"]") {
					t.Fatalf("non-interactive %q kept\nin:\n%s\nout:\n%s", n.key(), raw, out)
				}
			})
		}
	})
}

func TestSnapshotPruneIsFixedPoint(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		_, raw := ariaDocGen(t)
		once := interactiveSnapshot(t, raw)
		if twice := interactiveSnapshot(t, once); twice != once {
			t.Fatalf("not a fixed point\nonce:\n%s\ntwice:\n%s", once, twice)
		}
	})
}
