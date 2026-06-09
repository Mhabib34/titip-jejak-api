// ─────────────────────────────────────────────
// config/index.js — Konfigurasi global TemuKan
// ─────────────────────────────────────────────

export const BASE_URL = __ENV.BASE_URL || "http://localhost:8080/api/v1";

// ── Thresholds standar (p95 < 1s, error rate < 5%) ─────────────────────────
export const THRESHOLDS = {
  http_req_duration: ["p(95)<1000"],
  http_req_failed:   ["rate<0.05"],

  // Threshold per-group (endpoint kritis)
  "http_req_duration{endpoint:login}":        ["p(95)<800"],
  "http_req_duration{endpoint:create_report}":["p(95)<1000"],
  "http_req_duration{endpoint:list_reports}": ["p(95)<1000"],
  "http_req_duration{endpoint:map_pins}":     ["p(95)<800"],
  "http_req_duration{endpoint:matches}":      ["p(95)<1000"],
};

// ── Stage definitions ────────────────────────────────────────────────────────
export const STAGES = {
  smoke: [
    { duration: "30s", target: 2 },
    { duration: "30s", target: 0 },
  ],

  load: [
    { duration: "1m",  target: 50  },  // ramp-up ke 50 VU
    { duration: "3m",  target: 100 },  // tahan di 100 VU
    { duration: "2m",  target: 200 },  // naik ke 200 VU
    { duration: "2m",  target: 200 },  // steady state
    { duration: "1m",  target: 0   },  // ramp-down
  ],

  stress: [
    { duration: "1m",  target: 100 },
    { duration: "2m",  target: 200 },
    { duration: "2m",  target: 400 },
    { duration: "2m",  target: 600 },
    { duration: "2m",  target: 800 },
    { duration: "1m",  target: 0   },
  ],

  spike: [
    { duration: "30s", target: 10  },  // baseline
    { duration: "15s", target: 500 },  // spike tiba-tiba!
    { duration: "1m",  target: 500 },  // tahan spike
    { duration: "15s", target: 10  },  // drop balik
    { duration: "1m",  target: 10  },  // recovery
    { duration: "30s", target: 0   },
  ],
};
