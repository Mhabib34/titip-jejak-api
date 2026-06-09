// ──────────────────────────────────────────────────────────────
// scenarios/load.js
// Beban normal — simulasi user journey berbobot (weighted)
//
// Distribusi traffic:
//   60% → Penemu/pencari browsing laporan & peta (read-heavy)
//   30% → Pelapor aktif (create/update report)
//   10% → Cek notifikasi & matches
// ──────────────────────────────────────────────────────────────
import { sleep } from "k6";
import { THRESHOLDS, STAGES, BASE_URL } from "../config/index.js";
import { authenticate, authHeaders, publicHeaders } from "../helpers/auth.js";
import { randomReport, randomUpdatePayload }        from "../helpers/data.js";
import {
  getEndpoint, postEndpoint, putEndpoint,
  deleteEndpoint, parseBody,
} from "../helpers/checks.js";

export const options = {
  stages:     STAGES.load,
  thresholds: THRESHOLDS,
};

// ── Journey A: Browser (60%) ─────────────────────────────────
function journeyBrowser(auth) {
  const headers = authHeaders(auth.token);

  // Lihat peta
  getEndpoint(`${BASE_URL}/map/pins`, publicHeaders, "map_pins");
  sleep(1);

  // Browse laporan dengan filter
  const filters = [
    "type=missing&city=Medan&status=active",
    "type=found&status=active&limit=10",
    "gender=male&age_min=50&age_max=80",
    "q=batik&status=active",
  ];
  const filter = filters[__VU % filters.length];
  getEndpoint(`${BASE_URL}/reports?${filter}`, publicHeaders, "list_reports");
  sleep(1);

  // Lihat stats homepage
  getEndpoint(`${BASE_URL}/stats`, publicHeaders, "stats");
  sleep(2);
}

// ── Journey B: Pelapor Aktif (30%) ───────────────────────────
function journeyReporter(auth) {
  const headers = authHeaders(auth.token);

  // Buat laporan
  const createRes = postEndpoint(
    `${BASE_URL}/reports`,
    randomReport(),
    headers,
    "create_report",
    201
  );
  const reportId = parseBody(createRes)?.data?.id;
  sleep(1);

  if (reportId) {
    // Cek laporan yang baru dibuat
    getEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "get_report");
    sleep(1);

    // Update status
    putEndpoint(
      `${BASE_URL}/reports/${reportId}`,
      randomUpdatePayload(),
      headers,
      "update_report"
    );
    sleep(1);

    // Cleanup (hapus laporan test agar DB tidak penuh)
    if (Math.random() > 0.3) {
      deleteEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "delete_report");
    }
    sleep(1);
  }

  // Lihat laporan milik sendiri
  getEndpoint(`${BASE_URL}/reports/my?limit=5`, headers, "my_reports");
  sleep(2);
}

// ── Journey C: Notifikasi & Matches (10%) ────────────────────
function journeyNotifications(auth) {
  const headers = authHeaders(auth.token);

  getEndpoint(`${BASE_URL}/notifications?is_read=false`, headers, "notifications");
  sleep(1);

  getEndpoint(`${BASE_URL}/matches?min_score=60`, headers, "matches");
  sleep(2);
}

// ── Main ─────────────────────────────────────────────────────
export default function () {
  const auth = authenticate();
  if (!auth) {
    sleep(2);
    return;
  }

  // Weighted random berdasarkan VU number
  const roll = Math.random();

  if (roll < 0.60) {
    journeyBrowser(auth);
  } else if (roll < 0.90) {
    journeyReporter(auth);
  } else {
    journeyNotifications(auth);
  }
}
