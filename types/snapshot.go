package types

import (
	"fmt"
	"github.com/agenthands/godoll/retry"
	"github.com/agenthands/rod-cli/types/js"
	"github.com/agenthands/rod-cli/utils"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// seams for testing: indirections through which production calls are routed so
// that otherwise-unreachable error branches can be exercised. Defaults are the
// real functions; tests swap them and restore via defer. Zero behavior change.
var (
	yamlMarshal    = yaml.Marshal
	yamlUnmarshal  = yaml.Unmarshal
	pageEval       = func(p *rod.Page, js string, args ...any) (*proto.RuntimeRemoteObject, error) { return p.Eval(js, args...) }
	pageInfo       = func(p *rod.Page) (*proto.TargetTargetInfo, error) { return p.Info() }
	executeTemple  = utils.ExecuteTemple
	queryEleByAria = utils.QueryEleByAria
)

const snapshotTpl = `
- Page URL: {{ .URL }}
- Page Title: {{ .Title }}
- Frame Count: {{ .Frames }}
- Page Snapshot
` + "```yaml\n" + "{{ .Snapshot }}" + "\n```\n"

// SnapshotOpts controls one snapshot capture. Gen is the ref generation
// (s<Gen>e<N>). Root is a JS expression for the root element; empty means
// document.body. Interactive keeps only interactive nodes and their ancestors.
type SnapshotOpts struct {
	Gen         int
	Root        string
	Interactive bool
}

type Snapshot struct {
	frames       []*rod.Page
	textSnapshot string
}

func BuildSnapshot(p *rod.Page, o SnapshotOpts) (*Snapshot, error) {
	snapshot := &Snapshot{
		frames: []*rod.Page{},
	}
	yamlDoc, err := snapshot.captureSnapshotWithFrames(p, o)
	if err != nil {
		return nil, err
	}
	if o.Interactive {
		yamlDoc = pruneInteractive(yamlDoc)
	}

	yamlBytes, err := yamlMarshal(yamlDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed,because of yaml marsha")
	}

	info, err := pageInfo(p)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed")
	}

	tplInfo := map[string]any{
		"URL":      info.URL,
		"Title":    info.Title,
		"Snapshot": strings.TrimSpace(string(yamlBytes)),
		"Frames":   len(snapshot.frames),
	}
	res, err := executeTemple(snapshotTpl, tplInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, because of tple exec failed")
	}
	snapshot.textSnapshot = res
	return snapshot, nil
}

func (s *Snapshot) String() string {
	return s.textSnapshot
}

// snapshotEvalMaxRetries/snapshotEvalRetryDelay bound how many times the
// AriaSnapshot eval is retried after a transient failure. Navigate's
// post-navigation DOM-settle wait (defaultWaitStableTimeout, actions.go) is
// bounded and best-effort — a slow, ad-interstitial-laden page (observed live:
// a client-side "portal opens in 20s" gate) can outlast it, so a snapshot
// issued right after navigation can race a moment where document.body is
// transiently null (a mid-navigation document teardown/rebuild), crashing the
// eval with a TypeError before it ever reaches ariaSnapshot()'s null-element
// guard. A short bounded retry absorbs that race; a genuinely broken page
// (detached target, closed browser) still fails, just after a bounded extra
// wait.
const (
	snapshotEvalMaxRetries = 3
	snapshotEvalRetryDelay = 200 * time.Millisecond
)

func (s *Snapshot) captureSnapshotWithFrames(p *rod.Page, o SnapshotOpts) (*yaml.Node, error) {
	s.frames = append(s.frames, p)
	frameIndex := len(s.frames) - 1

	root := o.Root
	if root == "" {
		root = "document.body"
	}
	opts := fmt.Sprintf("({ref: true, generation: %d})", o.Gen)
	var rawSnapshot *proto.RuntimeRemoteObject
	injected := false
	err := retry.Retry(func() error {
		var evalErr error
		rawSnapshot, evalErr = pageEval(p, js.AriaSnapshot, root, opts)
		if evalErr != nil && !injected && strings.Contains(evalErr.Error(), "snapshotEngine is not defined") {
			injected = true
			if _, injErr := pageEval(p, js.InjectEngine); injErr != nil {
				return injErr
			}
			rawSnapshot, evalErr = pageEval(p, js.AriaSnapshot, root, opts)
		}
		return evalErr
	}, retry.WithMaxRetries(snapshotEvalMaxRetries), retry.WithExponentialBackoff(), retry.WithDelay(snapshotEvalRetryDelay))
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, frame index: %d", frameIndex)
	}

	var snapNode yaml.Node

	err = yamlUnmarshal([]byte(rawSnapshot.Value.String()), &snapNode)
	if err != nil {
		return nil, errors.Wrapf(err, "capture snapshot with frames failed, frame index: %d", frameIndex)
	}
	return s.walk(&snapNode, frameIndex, p, o)

}

func (s *Snapshot) walk(node *yaml.Node, frameIndex int, frame *rod.Page, o SnapshotOpts) (*yaml.Node, error) {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		rootNode := node.Content[0]
		processedRoot, err := s.walk(rootNode, frameIndex, frame, o)
		if err != nil {
			return nil, err
		}
		node.Content[0] = processedRoot
		return node, nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		// Process mapping nodes (key-value pairs)
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Process key
			newKey, err := s.walk(keyNode, frameIndex, frame, o)
			if err != nil {
				return nil, err
			}

			// Process value
			newValue, err := s.walk(valueNode, frameIndex, frame, o)
			if err != nil {
				return nil, err
			}

			node.Content[i] = newKey
			node.Content[i+1] = newValue
		}
	case yaml.SequenceNode:
		// Process sequence nodes
		for i, item := range node.Content {
			processedItem, err := s.walk(item, frameIndex, frame, o)
			if err != nil {
				return nil, err
			}
			node.Content[i] = processedItem
		}
	case yaml.ScalarNode:
		// Process scalar nodes
		if node.Tag == "!!str" {

			value := node.Value
			if frameIndex > 0 {
				node.Value = strings.Replace(value, "[ref=", fmt.Sprintf("[ref=f%d", frameIndex), 1)
			}

			if strings.HasPrefix(value, "iframe ") {
				re := regexp.MustCompile(`\[ref=(.*?)\]`)
				matches := re.FindStringSubmatch(value)
				if len(matches) > 1 {
					ref := matches[1]

					// Create a new mapping node to represent the pair
					// default for error
					pairNode := &yaml.Node{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: node.Value},
							{Kind: yaml.ScalarNode, Value: "<could not capture iframe snapshot>"},
						},
					}

					childFrameEle, err := queryEleByAria(frame, ref)

					if err != nil {
						return pairNode, nil
					}
					childFrame, err := childFrameEle.Frame()
					if err != nil {
						return pairNode, nil
					}
					childOpts := o
					childOpts.Root = "" // child frames always snapshot document.body
					childSnapshot, err := s.captureSnapshotWithFrames(childFrame, childOpts)

					if err != nil {
						return pairNode, nil
					}

					if len(childSnapshot.Content) == 0 {
						pairNode.Content = []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: node.Value},
							{Kind: yaml.ScalarNode, Value: "<could not capture iframe snapshot>"},
						}
					} else {
						pairNode.Content = []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: node.Value},
							childSnapshot.Content[0],
						}
					}

					return pairNode, nil
				}
			}
		}
	}
	return node, nil
}

func (s *Snapshot) LocatorInFrame(ref string) (*rod.Element, error) {
	frame := s.frames[0]
	matches := regexp.MustCompile(`^f(\d+)(.*)`).FindStringSubmatch(ref)
	if len(matches) > 0 {
		frameIndex, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, errors.Wrapf(err, "locator frame failed, because of frame index is not number")
		}

		if frameIndex < 0 || frameIndex >= len(s.frames) {
			return nil, errors.Errorf("locator frame failed, because of frame index is out of range")
		}
		frame = s.frames[frameIndex]
		ref = matches[2]
	}
	ele, err := queryEleByAria(frame, ref)
	if err != nil {
		return nil, errors.Wrapf(err, "locator frame failed, because of query element by aria failed")
	}
	return ele, nil

}

var interactiveRoles = strings.Fields("button link textbox searchbox checkbox radio combobox listbox option menuitem tab switch slider spinbutton heading")

func isInteractive(v string) bool {
	for _, r := range interactiveRoles {
		if strings.HasPrefix(v, r) && (len(v) == len(r) || v[len(r)] == ' ' || v[len(r)] == '[') {
			return true
		}
	}
	return false
}

// pruneInteractive keeps interactive nodes and their ancestors, dropping
// everything else (including a control's non-interactive children). Returns nil when nothing under n is kept (a document
// node is returned with an empty sequence instead).
// ponytail: keeps unnamed wrapper ancestors; flatten them if output is still too large
func pruneInteractive(n *yaml.Node) *yaml.Node {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) > 0 {
			if c := pruneInteractive(n.Content[0]); c != nil {
				n.Content[0] = c
			} else {
				n.Content[0] = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			}
		}
		return n
	case yaml.SequenceNode:
		kept := n.Content[:0]
		for _, c := range n.Content {
			if p := pruneInteractive(c); p != nil {
				kept = append(kept, p)
			}
		}
		n.Content = kept
	case yaml.MappingNode:
		kept := n.Content[:0]
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if p := pruneInteractive(v); p != nil {
				kept = append(kept, k, p)
			} else if isInteractive(k.Value) {
				if len(n.Content) == 2 {
					return k // `- link "x": {/url: ...}` collapses to `- link "x"`
				}
				kept = append(kept, k, v)
			}
		}
		n.Content = kept
	case yaml.ScalarNode:
		if isInteractive(n.Value) {
			return n
		}
		return nil
	default:
		return nil
	}
	if len(n.Content) == 0 {
		return nil
	}
	return n
}
