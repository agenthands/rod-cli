// on_load.js - onLoad recipe for rod-cli
// Hook: onLoad -> CDP event proto.PageLoadEventFired
// Reads page state through the api global after each page load and records
// the length of the HTML snapshot.
// Read results with: plugin run getLoadLog

var loadLog = [];

// onLoad fires when the page finishes loading. The api global is bound at
// load time, so guard with typeof and a try/catch for contexts without it.
function onLoad(event) {
  if (typeof api !== "undefined") {
    try {
      var snapshot = api.GetSnapshot();
      loadLog.push({
        snapshotLength: snapshot.length,
      });
    } catch (e) {
      // api may not be available in all contexts
    }
  }
}

// getLoadLog returns the recorded load events as JSON.
function getLoadLog() {
  return JSON.stringify(loadLog);
}
