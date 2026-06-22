// xss_scanner.js - Example XSS scanner plugin for rod-cli
// This plugin hooks into the lifecycle events and logs network
// requests that may contain reflected XSS payloads.

var findings = [];
var requestLog = [];

// XSS test payloads
var payloads = [
  "<script>alert('XSS')</script>",
  "\"><script>alert('XSS')</script>",
  "<img src=x onerror=alert('XSS')>",
  "<svg onload=alert('XSS')>",
];

// onRequest is called by the rod-cli plugin engine before each network request
function onRequest(event) {
  if (event && event.Request && event.Request.URL) {
    requestLog.push({
      url: event.Request.URL,
      method: event.Request.Method || "GET",
    });
  }
}

// onResponse is called when a response comes back
function onResponse(event) {
  if (event && event.Response) {
    // Check if any response URL contains a payload echo
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

// onLoad fires when a page finishes loading
function onLoad(event) {
  // After page load, we could check the DOM via the api global
  if (typeof api !== "undefined") {
    try {
      var snapshot = api.GetSnapshot();
      for (var i = 0; i < payloads.length; i++) {
        if (snapshot.indexOf(payloads[i]) > -1) {
          findings.push({
            type: "reflected_in_dom",
            payload: payloads[i],
            evidence: snapshot.substring(
              snapshot.indexOf(payloads[i]) - 30,
              snapshot.indexOf(payloads[i]) + payloads[i].length + 30
            ),
          });
        }
      }
    } catch (e) {
      // api may not be available in all contexts
    }
  }
}

// getFindings returns all collected findings
function getFindings() {
  return JSON.stringify(findings);
}

// getRequestLog returns all collected requests
function getRequestLog() {
  return JSON.stringify(requestLog);
}
