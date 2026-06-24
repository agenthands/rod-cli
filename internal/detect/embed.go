package detect

import _ "embed"

//go:embed detect.html
var detectHTML string

//go:embed detect.js
var detectJS string

// Probe is the shared, canonical table-stakes stealth probe (probe.js). It is
// injected via eval by the StealthCheck action and computes the same signal
// names the detect.js harness fixture exposes on window.__detect, so the
// command and the harness agree on one source of truth.
//
//go:embed probe.js
var Probe string
