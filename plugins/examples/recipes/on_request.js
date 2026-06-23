// on_request.js - onRequest recipe for rod-cli
// Hook: onRequest -> CDP event proto.NetworkRequestWillBeSent
// Logs the URL and method of every outgoing network request.
// Read results with: plugin run getRequestLog

var requestLog = [];

// onRequest fires before each network request leaves the browser.
function onRequest(event) {
  if (event && event.Request && event.Request.URL) {
    requestLog.push({
      url: event.Request.URL,
      method: event.Request.Method || "GET",
    });
  }
}

// getRequestLog returns all collected requests as JSON.
function getRequestLog() {
  return JSON.stringify(requestLog);
}
