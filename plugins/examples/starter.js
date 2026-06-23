// starter.js - Copyable starter template for rod-cli plugins
//
// HOW TO USE:
//   1. Copy this file to a new name (e.g. my_plugin.js).
//   2. Fill in the bodies of the hooks you need; delete the ones you don't.
//      All four hooks are OPTIONAL — an undefined hook is silently no-op'd
//      by the engine, so removing one changes nothing.
//   3. Load it with:   plugin load my_plugin.js
//   4. Drive the browser, then read collected state with:
//        plugin run getResults
//
// The script runs in the goja sandbox: there is NO console, fs, require, or
// import. Use only the hook payloads (event.Request / event.Response /
// event.Node) and the `api` global (available inside hooks after load).

var results = [];

// onRequest fires before each network request leaves the browser.
// Payload: event.Request.URL, event.Request.Method, event.Request.Headers
function onRequest(event) {
  /* TODO: handle request */
}

// onResponse fires when a response arrives.
// Payload: event.Response.URL, event.Response.Status, event.Response.Headers
function onResponse(event) {
  /* TODO: handle response */
}

// onLoad fires when the page finishes loading.
// Read page state via the api global, e.g. api.GetSnapshot().
function onLoad(event) {
  /* TODO: handle load */
}

// onDOMNodeInserted fires each time a child node is inserted into the DOM.
// Payload: event.ParentNodeID, event.Node (the inserted node).
function onDOMNodeInserted(event) {
  /* TODO: handle DOM insertion */
}

// getResults returns the collected state as JSON for `plugin run getResults`.
function getResults() {
  return JSON.stringify(results);
}
