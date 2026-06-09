// ──────────────────────────────────────────────────────────────
// scenarios/stress.js
// Stress test — dorong server ke batas maksimal
//
// Fokus pada endpoint paling berat:
//   - POST /reports (DB write + matching engine trigger)
//   - GET /reports (query + pagination)
//   - GET /map/pins (banyak data, query berat)
//   - GET /matches  (query scoring kompleks)
// ──────────────────────────────────────────────────────────────
import { sleep } from "k6";
import { THRESHOLDS, STAGES, BASE_URL } from "../config/index.js";
import { authenticate, authHeaders, publicHeaders } from "../helpers/auth.js";
import { randomReport }         from "../helpers/data.js";
import { getEndpoint, postEndpoint, deleteEndpoint, parseBody } from "../helpers/checks.js";

export const options = {
  stages: STAGES.stress,
  thresholds: {
    // Stress: boleh lebih longgar sedikit dari standar
    http_req_duration: ["p(95)<2000", "p(99)<5000"],
    http_req_failed:   ["rate<0.10"],

    // Endpoint kritis tetap dijaga
    "http_req_duration{endpoint:map_pins}":     ["p(95)<1500"],
    "http_req_duration{endpoint:list_reports}": ["p(95)<2000"],
    "http_req_duration{endpoint:create_report}":["p(95)<2000"],
    "http_req_duration{endpoint:matches}":      ["p(95)<2000"],
  },
};

export default function () {
  const auth = authenticate();
  if (!auth) {
    sleep(1);
    return;
  }

  const headers = authHeaders(auth.token);

  // Peta — endpoint paling sering dipanggil di frontend
  getEndpoint(`${BASE_URL}/map/pins`, publicHeaders, "map_pins");
  sleep(0.3);

  // List laporan dengan berbagai filter
  getEndpoint(
    `${BASE_URL}/reports?status=active&city=Medan&limit=20&page=${(__ITER % 5) + 1}`,
    publicHeaders,
    "list_reports"
  );
  sleep(0.3);

  // Buat laporan (write + trigger matching)
  const createRes = postEndpoint(
    `${BASE_URL}/reports`,
    randomReport(),
    headers,
    "create_report",
    201
  );
  const reportId = parseBody(createRes)?.data?.id;
  sleep(0.3);

  // Bersihkan setelah create (jangan flood DB)
  if (reportId) {
    deleteEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "delete_report");
  }
  sleep(0.3);

  // Matches — query scoring berat
  getEndpoint(`${BASE_URL}/matches`, headers, "matches");
  sleep(0.3);

  // Notifikasi
  getEndpoint(`${BASE_URL}/notifications`, headers, "notifications");
  sleep(0.2);
}
