package js

import _ "embed"

//go:embed snapshotter.js
var InjectedSnapShot string

// InjectEngine installs snapshotEngine on a page or frame that was never
// prepared (no EvalOnNewDocument), e.g. a tab opened outside rod-cli.
var InjectEngine = "() => {" + InjectedSnapShot + "\n;window.snapshotEngine = snapshotEngine}"

//go:embed annotator.js
var AnnotatorUI string

const AriaSnapshot = "function(node, opts) { return snapshotEngine.ariaSnapshot(eval(node), eval(opts)); }"

const QueryEleByAria = `(selector) => {
    return snapshotEngine.queryAll(selector);
}`
