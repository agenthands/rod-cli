// on_response.js - onResponse recipe for rod-cli
// Hook: onResponse -> CDP event proto.NetworkResponseReceived
// Logs the URL and status code of every response received.
// Read results with: plugin run getResponseLog

var responseLog = [];

// onResponse fires when a response arrives.
function onResponse(event) {
  if (event && event.Response) {
    responseLog.push({
      url: event.Response.URL || "",
      status: event.Response.Status,
    });
  }
}

// getResponseLog returns all collected responses as JSON.
function getResponseLog() {
  return JSON.stringify(responseLog);
}
