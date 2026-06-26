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

  // --- Phase 33: advanced fingerprint-dimension signals (sync) --------------

  // fonts: a measureText width signature across a fixed probe set. NOTE: godoll's
  // current font injector (scriptMockFonts) returns the original measureText width
  // on every branch, so this signature does NOT change when font-spoof toggles —
  // it is shipped as a stable informational signal (and for stealth-check), not a
  // coherence/toggle discriminator. The harness proves toggle effectiveness on
  // mediaDevices/battery/codecs instead.
  probe("fonts", function () {
    var c = document.createElement("canvas");
    var ctx = c.getContext("2d");
    if (!ctx) return "no-2d-context";
    var sample = "mmmmmmmmmmlli";
    var families = ["monospace", "sans-serif", "serif", "Arial", "Times New Roman", "Courier New"];
    var parts = [];
    for (var i = 0; i < families.length; i++) {
      ctx.font = "72px " + families[i];
      parts.push(Math.round(ctx.measureText(sample).width));
    }
    return parts.join(",");
  });

  // codecs: a canPlayType signature for representative MIME types. godoll's codec
  // injector overrides HTMLMediaElement.prototype.canPlayType, so a toggle ON/OFF
  // can change these readings. Empty string ("" = not supported) is normalized to
  // "no" so the signature stays comparable.
  probe("codecs", function () {
    var v = document.createElement("video");
    var a = document.createElement("audio");
    var cases = [
      ['video/mp4; codecs="avc1.42E01E"', v],
      ['video/webm; codecs="vp9"', v],
      ['video/ogg; codecs="theora"', v],
      ["audio/mpeg", a],
      ['audio/ogg; codecs="opus"', a],
    ];
    var parts = [];
    for (var i = 0; i < cases.length; i++) {
      var r = cases[i][1].canPlayType(cases[i][0]);
      parts.push(cases[i][0] + "=" + (r === "" ? "no" : r));
    }
    return parts.join("|");
  });

  // --- Async probes: permissions + media devices + battery ------------------
  //
  // `ready` flips true only after EVERY async probe settles (a reference-counted
  // gate), with a global timeout fallback so a stalled probe never blocks the
  // reader forever. Probes:
  //   - permissionsConsistent: Notification.permission vs the state reported by
  //     navigator.permissions.query({name:'notifications'}) (the classic headless
  //     "default" vs "denied" mismatch).
  //   - mediaDevices: navigator.mediaDevices.enumerateDevices() count + kinds
  //     (godoll scriptMockMediaDevices overrides this; headless default differs).
  //   - battery: navigator.getBattery() presence + level + charging (godoll
  //     scriptMockBattery overrides getBattery to resolve a fixed BatteryManager).

  var pending = 0;
  var timedOut = false;
  var settleTimer = null;

  function markReady() {
    if (d.ready) return;
    if (settleTimer) {
      clearTimeout(settleTimer);
      settleTimer = null;
    }
    d.ready = true;
  }

  // settleOne decrements the pending counter; ready flips when all probes settle.
  function settleOne() {
    if (timedOut) return;
    pending--;
    if (pending <= 0) markReady();
  }

  // asyncProbe registers one async probe in the gate; a synchronous throw inside
  // the registrant still settles its slot so the gate can never deadlock.
  function asyncProbe(fn) {
    pending++;
    try {
      fn(settleOne);
    } catch (e) {
      settleOne();
    }
  }

  // Registration guard: hold one extra count across the whole registration phase
  // and release it only after every probe is registered (releaseGuard below). This
  // prevents an early `ready` flip if a probe settles SYNCHRONOUSLY before its
  // siblings are registered (e.g. a missing permissions API) — pending can never
  // hit 0 mid-registration while the guard is held.
  pending++;

  // permissions consistency
  asyncProbe(function (done) {
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
          done();
        })
        .catch(function (e) {
          d.permissionsConsistent = "error: " + (e && e.message ? e.message : String(e));
          done();
        });
    } else {
      d.notificationPermission = notif;
      d.permissionsQueryState = "no-permissions-api";
      d.permissionsConsistent = "no-permissions-api";
      done();
    }
  });

  // media devices: count + sorted kind set (a stable comparable signature).
  asyncProbe(function (done) {
    if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
      navigator.mediaDevices
        .enumerateDevices()
        .then(function (list) {
          d.mediaDevicesCount = list.length;
          var kinds = {};
          for (var i = 0; i < list.length; i++) {
            var k = (list[i] && list[i].kind) || "unknown";
            kinds[k] = (kinds[k] || 0) + 1;
          }
          d.mediaDevicesKinds = Object.keys(kinds).sort().join(",");
          d.mediaDevices = d.mediaDevicesCount + ":" + d.mediaDevicesKinds;
          done();
        })
        .catch(function (e) {
          d.mediaDevices = "error: " + (e && e.message ? e.message : String(e));
          done();
        });
    } else {
      d.mediaDevices = "no-mediaDevices-api";
      done();
    }
  });

  // battery: presence + level + charging (a stable comparable signature).
  asyncProbe(function (done) {
    if (typeof navigator.getBattery === "function") {
      try {
        navigator
          .getBattery()
          .then(function (b) {
            d.batteryPresent = true;
            d.batteryLevel = b.level;
            d.batteryCharging = b.charging;
            d.battery = "present:" + b.level + ":" + b.charging;
            done();
          })
          .catch(function (e) {
            d.battery = "error: " + (e && e.message ? e.message : String(e));
            done();
          });
      } catch (e) {
        d.battery = "error: " + (e && e.message ? e.message : String(e));
        done();
      }
    } else {
      d.batteryPresent = false;
      d.battery = "no-getBattery";
      done();
    }
  });

  // Release the registration guard: now that every probe is registered, settling
  // the guard lets `ready` flip as soon as the real probes have all completed
  // (or immediately, if they already settled synchronously during registration).
  settleOne();

  // Global timeout fallback: always populate `ready` even if a probe stalls.
  settleTimer = setTimeout(function () {
    timedOut = true;
    markReady();
  }, 3000);
})();
