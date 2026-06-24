// Self-authored, offline, dependency-free bot-detection probe page.
//
// On load it computes the table-stakes detection signals plus informational
// WebRTC/CDP probes and writes a flat per-signal verdict object onto the single
// known global `window.__detect`. The e2e harness reads these back via `eval`.
//
// Discipline:
//   - No external fetch / CDN / network egress (offline single-binary constraint).
//   - Every probe is wrapped in try/catch and records its error string into its
//     own signal rather than throwing, so one failing probe never blanks the
//     whole global.
//   - Async probes (permissions, WebRTC) resolve into the global *before*
//     `window.__detect.ready = true` is set, with a timeout fallback so the
//     global is always eventually populated.
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

  // User-Agent — harness asserts it does NOT contain "HeadlessChrome".
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

  // Screen dimensions + outer window size (zero outer size is a headless tell).
  probe("screen", function () {
    return screen.width + "x" + screen.height;
  });
  probe("outerSize", function () {
    return window.outerWidth + "x" + window.outerHeight;
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

  // --- Informational, non-blocking probes (recorded, not asserted-blocking) --

  // CDP-presence heuristic (best-effort, informational only). The classic tell:
  // serializing an Error triggers the stack getter, which DevTools/CDP remote-
  // object preview may observe. This is a HEURISTIC — `console.debug` is not
  // guaranteed to invoke accessor getters, so "no-signal" is a possible false
  // negative even under CDP. It measures exposure, it does not prove its absence;
  // see docs/cdp-footprint.md for the honest ceiling (fix deferred to v2 CDP-01).
  probe("cdpTell", function () {
    var fired = false;
    var e = new Error();
    Object.defineProperty(e, "stack", {
      configurable: true,
      get: function () {
        fired = true;
        return "";
      },
    });
    // Force serialization, which reads `.stack`.
    try {
      console.debug(e);
    } catch (_) {
      /* ignore */
    }
    return fired ? "stack-getter-fired" : "no-signal";
  });

  // --- Async probes: permissions consistency + WebRTC ICE leak ---------------
  //
  // permissionsConsistent: Notification.permission vs the state reported by
  // navigator.permissions.query({name:'notifications'}). The classic headless
  // mismatch is "default" vs "denied"/"prompt".
  //
  // webrtcIce: gather ICE candidates and record any host/local IP discovered.
  // This is the KNOWN-RED WebRTC leak signal — record current truth, do not fix
  // (HARDEN-01 is Phase 27).

  var pending = 2;
  var settleTimer = null;

  function markReady() {
    if (d.ready) return;
    d.ready = true;
  }

  function settleOne() {
    pending -= 1;
    if (pending <= 0) {
      if (settleTimer) {
        clearTimeout(settleTimer);
        settleTimer = null;
      }
      markReady();
    }
  }

  // Permissions consistency.
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
            settleOne();
          })
          .catch(function (e) {
            d.permissionsConsistent = "error: " + (e && e.message ? e.message : String(e));
            settleOne();
          });
      } else {
        d.notificationPermission = notif;
        d.permissionsQueryState = "no-permissions-api";
        d.permissionsConsistent = "no-permissions-api";
        settleOne();
      }
    } catch (e) {
      d.permissionsConsistent = "error: " + (e && e.message ? e.message : String(e));
      settleOne();
    }
  })();

  // WebRTC ICE leak.
  (function () {
    try {
      if (typeof RTCPeerConnection === "undefined") {
        d.webrtcIce = "no-RTCPeerConnection";
        settleOne();
        return;
      }
      var pc = new RTCPeerConnection({ iceServers: [] });
      var ips = {};
      var done = false;
      function finish() {
        if (done) return;
        done = true;
        var list = Object.keys(ips);
        d.webrtcIce = list.length ? list.join(",") : "";
        try {
          pc.close();
        } catch (_) {
          /* ignore */
        }
        settleOne();
      }
      pc.onicecandidate = function (ev) {
        if (!ev || !ev.candidate) {
          // null candidate = gathering complete.
          finish();
          return;
        }
        // The SDP candidate line is: `candidate:<foundation> <component>
        // <transport> <priority> <connection-address> <port> typ <type> ...`.
        // The connection-address (field index 4) is the real host surface —
        // capture it regardless of form so the probe is not blind to IPv6 or to
        // modern Chrome's mDNS masking (`<uuid>.local`), which an IPv4-only regex
        // silently misses (recording "" for the wrong reason). A real IPv4/IPv6
        // address here is the leak Phase 27 EvadeWebRTC must eliminate; a `.local`
        // mDNS hostname is the masked baseline truth.
        var parts = (ev.candidate.candidate || "").split(" ");
        if (parts.length > 4 && parts[4]) ips[parts[4]] = true;
      };
      pc.createDataChannel("probe");
      pc.createOffer()
        .then(function (offer) {
          return pc.setLocalDescription(offer);
        })
        .catch(function (e) {
          d.webrtcIce = "error: " + (e && e.message ? e.message : String(e));
          finish();
        });
      // Safety net in case gathering never completes.
      setTimeout(finish, 1500);
    } catch (e) {
      d.webrtcIce = "error: " + (e && e.message ? e.message : String(e));
      settleOne();
    }
  })();

  // Global timeout fallback: always populate `ready` even if a probe stalls.
  settleTimer = setTimeout(markReady, 3000);
})();
