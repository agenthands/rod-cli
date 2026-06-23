// =============================================================================
// xss_scanner.js — flagship example plugin for rod-cli
// =============================================================================
//
// What it does:
//   Watches a browsing session for signs of reflected XSS. It records every
//   outgoing request, then checks each response URL and the post-load DOM for
//   echoes of a small set of known XSS payloads. Anything that looks reflected
//   is recorded as a finding.
//
// How to use it:
//   1. Load the plugin:
//        plugin load ./plugins/examples/xss_scanner.js
//   2. Drive a (deliberately) vulnerable page — submit a payload, follow a
//      reflecting link, etc. The hooks below collect data as you browse.
//   3. Read what it found:
//        plugin run getFindings      # JSON array of candidate XSS reflections
//        plugin run getRequestLog    # JSON array of every observed request
//
// Where the exploit logic lives:
//   The XSS payloads and the matching logic live HERE, in this user-space
//   example script — never in the rod-cli binary. The script only reads page
//   state already surfaced by the sandboxed `api` global; it adds nothing the
//   operator could not already observe in their own session.
//
// Sandbox notes:
//   This runs in the goja JS engine: no console, fs, require, or import. The
//   `api` global is bound at `plugin load` and is only present inside hooks,
//   so api usage is wrapped in a `typeof api` guard plus a try/catch.
// =============================================================================

// -----------------------------------------------------------------------------
// State — module-level arrays accumulated across the session and read back by
// the accessors at the bottom of the file.
// -----------------------------------------------------------------------------
var findings = []; // candidate reflections (reflected_candidate / reflected_in_dom)
var requestLog = []; // every observed outgoing request ({url, method})

// -----------------------------------------------------------------------------
// Payloads — the XSS strings we look for being reflected. Kept here in user
// space (not in the binary) so operators can freely extend the list.
// -----------------------------------------------------------------------------
var payloads = [
  "<script>alert('XSS')</script>",
  "\"><script>alert('XSS')</script>",
  "<img src=x onerror=alert('XSS')>",
  "<svg onload=alert('XSS')>",
];

// -----------------------------------------------------------------------------
// Hooks — one function per CDP lifecycle event. Each is optional; the engine
// silently no-ops any hook a plugin does not define. Every hook defends against
// a missing/partial payload before reading fields.
// -----------------------------------------------------------------------------

// onRequest — CDP event: proto.NetworkRequestWillBeSent.
// Reads event.Request.URL and event.Request.Method. Records each outgoing
// request so getRequestLog() can replay what the session touched.
function onRequest(event) {
  if (event && event.Request && event.Request.URL) {
    requestLog.push({
      url: event.Request.URL,
      method: event.Request.Method || "GET",
    });
  }
}

// onResponse — CDP event: proto.NetworkResponseReceived.
// Reads event.Response.URL. Flags a response whose URL contains a
// URL-encoded payload echo as a reflected_candidate finding.
function onResponse(event) {
  if (event && event.Response) {
    var url = event.Response.URL || "";
    for (var i = 0; i < payloads.length; i++) {
      if (url.indexOf(encodeURIComponent(payloads[i])) > -1) {
        findings.push({
          type: "reflected_candidate",
          url: url,
          payload: payloads[i],
        });
      }
    }
  }
}

// onLoad — CDP event: proto.PageLoadEventFired.
// After the page settles, reads the rendered DOM via api.GetSnapshot() and
// flags any payload that appears verbatim in the HTML as reflected_in_dom,
// capturing a short surrounding evidence window. Guarded by `typeof api` and
// try/catch because `api` only exists inside a loaded session.
function onLoad(event) {
  if (typeof api !== "undefined") {
    try {
      var snapshot = api.GetSnapshot();
      for (var i = 0; i < payloads.length; i++) {
        var at = snapshot.indexOf(payloads[i]);
        if (at > -1) {
          findings.push({
            type: "reflected_in_dom",
            payload: payloads[i],
            evidence: snapshot.substring(
              at - 30,
              at + payloads[i].length + 30
            ),
          });
        }
      }
    } catch (e) {
      // api may not be available (e.g. parsed outside a live session) — ignore.
    }
  }
}

// -----------------------------------------------------------------------------
// Accessors — invoked by `plugin run <name>` via the engine's RunFunc. Each
// returns JSON.stringify(...) so the CLI prints clean JSON.
// -----------------------------------------------------------------------------

// getFindings returns all collected XSS findings as JSON.
function getFindings() {
  return JSON.stringify(findings);
}

// getRequestLog returns every observed request as JSON.
function getRequestLog() {
  return JSON.stringify(requestLog);
}
