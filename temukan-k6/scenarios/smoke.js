// ──────────────────────────────────────────────────────────────
// scenarios/smoke.js
// Sanity check — 2 VU, 1 menit
// Pastikan semua endpoint bisa diakses dan return 2xx
// ──────────────────────────────────────────────────────────────
import { sleep } from "k6";
import { THRESHOLDS, STAGES, BASE_URL } from "../config/index.js";
import { authenticate, authHeaders, publicHeaders } from "../helpers/auth.js";
import { randomReport, randomUpdatePayload }        from "../helpers/data.js";
import {
  getEndpoint, postEndpoint, putEndpoint,
  deleteEndpoint, patchEndpoint, parseBody,
} from "../helpers/checks.js";

export const options = {
  stages:     STAGES.smoke,
  thresholds: THRESHOLDS,
};

export default function () {
  // ── 1. Health check (publik) ─────────────────────────────────
  getEndpoint(`${BASE_URL}/health`, publicHeaders, "health");
  sleep(0.5);

  // ── 2. Stats (publik) ────────────────────────────────────────
  getEndpoint(`${BASE_URL}/stats`, publicHeaders, "stats");
  sleep(0.5);

  // ── 3. Map pins (publik) ─────────────────────────────────────
  getEndpoint(`${BASE_URL}/map/pins`, publicHeaders, "map_pins");
  sleep(0.5);

  // ── 4. List laporan publik ───────────────────────────────────
  getEndpoint(`${BASE_URL}/reports?status=active&limit=5`, publicHeaders, "list_reports");
  sleep(0.5);

  // ── 5. Auth flow (register + login) ─────────────────────────
  const auth = authenticate();
  if (!auth) return;

  const headers = authHeaders(auth.token);
  sleep(0.5);

  // ── 6. Profil saya ───────────────────────────────────────────
  getEndpoint(`${BASE_URL}/auth/me`, headers, "auth_me");
  sleep(0.5);

  // ── 7. Buat laporan ──────────────────────────────────────────
  const createRes = postEndpoint(
    `${BASE_URL}/reports`,
    randomReport(),
    headers,
    "create_report",
    201
  );
  const reportId = parseBody(createRes)?.data?.id;
  sleep(0.5);

  if (reportId) {
    // ── 8. Detail laporan ──────────────────────────────────────
    getEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "get_report");
    sleep(0.5);

    // ── 9. Update laporan ──────────────────────────────────────
    putEndpoint(
      `${BASE_URL}/reports/${reportId}`,
      randomUpdatePayload(),
      headers,
      "update_report"
    );
    sleep(0.5);

    // ── 10. Hapus laporan ──────────────────────────────────────
    deleteEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "delete_report");
    sleep(0.5);
  }

  // ── 11. Laporan milik saya ───────────────────────────────────
  getEndpoint(`${BASE_URL}/reports/my`, headers, "my_reports");
  sleep(0.5);

  // ── 12. Matches ──────────────────────────────────────────────
  getEndpoint(`${BASE_URL}/matches`, headers, "matches");
  sleep(0.5);

  // ── 13. Notifikasi ───────────────────────────────────────────
  getEndpoint(`${BASE_URL}/notifications`, headers, "notifications");
  sleep(0.5);

  // ── 14. Mark all read ─────────────────────────────────────────
  patchEndpoint(`${BASE_URL}/notifications/read-all`, null, headers, "notif_read_all");
  sleep(0.5);

  // ── 15. Logout ───────────────────────────────────────────────
  postEndpoint(`${BASE_URL}/auth/logout`, {}, headers, "logout", 200);
}
