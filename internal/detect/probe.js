// Shared, offline, dependency-free table-stakes stealth probe.
//
// This is the SINGLE canonical source for the table-stakes detection signals
// consumed by BOTH the `stealth-check` command (injected via eval from the
// StealthCheck action) and the Phase 24 detect.js harness fixture. The signal
// NAMES and SEMANTICS here are the contract: detect.js computes the same
// table-stakes subset (plus extra informational WebRTC/CDP probes it owns as the
// fixture page), and tests/detection_test.go reads these names back from the
// live page. Do NOT author a divergent second probe — extend this one.
//
// On injection it computes the synchronous table-stakes signals plus the async
// permissions-consistency signal and writes a flat per-signal object onto the
// single known global `window.__detect`. `window.__detect.ready` flips true only
// after the async permissions probe settles (with a timeout fallback so the
// global is always eventually populated).
//
// Discipline (mirrors detect.js):
//   - No external fetch / CDN / network egress (offline single-binary constraint).
//   - Every probe is wrapped in try/catch and records its error string into its
//     own signal rather than throwing, so one failing probe never blanks the
//     whole global.
(function () {
  "use strict";

  var d = (window.__detect = {});
  d.ready = false;

  // Record a signal, swallowing probe errors into the value itself.
  function probe(name, fn) {
    try {
      d[name] = fn();
    } catch (e) {
      d[name] = "error: " + (e && e.message ? e.message : String(e));
    }
  }

  // --- Table-stakes signals -------------------------------------------------

  // navigator.webdriver — true on automated Chrome unless masked.
  probe("webdriver", function () {
    return navigator.webdriver;
  });

  // Plugin / mimeType counts — headless Chrome historically reports 0.
  probe("pluginsLength", function () {
    return navigator.plugins ? navigator.plugins.length : -1;
  });
  probe("mimeTypesLength", function () {
    return navigator.mimeTypes ? navigator.mimeTypes.length : -1;
  });

  // User-Agent — must NOT contain "HeadlessChrome".
  probe("userAgent", function () {
    return navigator.userAgent;
  });

  // WebGL vendor/renderer via WEBGL_debug_renderer_info.
  // UNMASKED_VENDOR_WEBGL = 37445, UNMASKED_RENDERER_WEBGL = 37446.
  probe("webglVendor", function () {
    var c = document.createElement("canvas");
    var gl = c.getContext("webgl") || c.getContext("experimental-webgl");
    if (!gl) return "no-context";
    var ext = gl.getExtension("WEBGL_debug_renderer_info");
    if (!ext) return "no-extension";
    return gl.getParameter(37445);
  });
  probe("webglRenderer", function () {
    var c = document.createElement("canvas");
    var gl = c.getContext("webgl") || c.getContext("experimental-webgl");
    if (!gl) return "no-context";
    var ext = gl.getExtension("WEBGL_debug_renderer_info");
    if (!ext) return "no-extension";
    return gl.getParameter(37446);
  });

  // navigator.languages joined.
  probe("languages", function () {
    return (navigator.languages || []).join(",");
  });

  // Screen dimensions (zero is a headless tell).
  probe("screen", function () {
    return screen.width + "x" + screen.height;
  });

  // window.chrome / chrome.runtime presence.
  probe("windowChrome", function () {
    return typeof window.chrome !== "undefined";
  });
  probe("chromeRuntime", function () {
    return typeof (window.chrome && window.chrome.runtime) !== "undefined";
  });

  // Resolved IANA timezone.
  probe("timezone", function () {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  });

  // --- Async probe: permissions consistency ---------------------------------
  //
  // permissionsConsistent: Notification.permission vs the state reported by
  // navigator.permissions.query({name:'notifications'}). The classic headless
  // mismatch is "default" vs "denied". `ready` flips true only after this
  // settles, with a global timeout fallback so a stalled probe never blocks
  // the reader forever.

  var settled = false;
  var settleTimer = null;

  function markReady() {
    if (d.ready) return;
    if (settleTimer) {
      clearTimeout(settleTimer);
      settleTimer = null;
    }
    d.ready = true;
  }

  function settle() {
    if (settled) return;
    settled = true;
    markReady();
  }

  (function () {
    try {
      var notif =
        typeof Notification !== "undefined" ? Notification.permission : "no-Notification";
      if (navigator.permissions && navigator.permissions.query) {
        navigator.permissions
          .query({ name: "notifications" })
          .then(function (status) {
            d.notificationPermission = notif;
            d.permissionsQueryState = status.state;
            d.permissionsConsistent =
              notif === "denied" ? status.state === "denied" : status.state !== "denied";
            settle();
          })
          .catch(function (e) {
            d.permissionsConsistent = "error: " + (e && e.message ? e.message : String(e));
            settle();
          });
      } else {
        d.notificationPermission = notif;
        d.permissionsQueryState = "no-permissions-api";
        d.permissionsConsistent = "no-permissions-api";
        settle();
      }
    } catch (e) {
      d.permissionsConsistent = "error: " + (e && e.message ? e.message : String(e));
      settle();
    }
  })();

  // Global timeout fallback: always populate `ready` even if the probe stalls.
  settleTimer = setTimeout(markReady, 3000);
})();
