// ──────────────────────────────────────────────────────────────
// scenarios/spike.js
// Spike test — simulasi viral/breaking news
//
// Skenario: berita tentang orang hilang menjadi viral,
// ratusan orang langsung membuka aplikasi dalam waktu singkat.
//
// Yang diukur:
//   - Apakah server crash saat spike?
//   - Berapa lama recovery setelah spike turun?
//   - Error rate saat puncak spike
// ──────────────────────────────────────────────────────────────
import { sleep } from "k6";
import { STAGES, BASE_URL } from "../config/index.js";
import { authenticate, authHeaders, publicHeaders } from "../helpers/auth.js";
import { randomReport } from "../helpers/data.js";
import { getEndpoint, postEndpoint, parseBody } from "../helpers/checks.js";

export const options = {
  stages: STAGES.spike,
  thresholds: {
    // Spike: toleransi lebih longgar, prioritas server tidak crash
    http_req_duration: ["p(95)<3000"],
    http_req_failed:   ["rate<0.15"],  // max 15% error saat spike masih acceptable
    "http_req_duration{endpoint:map_pins}":     ["p(95)<2000"],
    "http_req_duration{endpoint:list_reports}": ["p(95)<3000"],
  },
};

// Saat spike, mayoritas user hanya read (buka app → lihat peta → browse)
export default function () {
  // Semua user langsung lihat peta & laporan (read-heavy saat viral)
  getEndpoint(`${BASE_URL}/map/pins?type=missing`, publicHeaders, "map_pins");
  sleep(0.2);

  getEndpoint(`${BASE_URL}/reports?status=active&limit=10`, publicHeaders, "list_reports");
  sleep(0.2);

  getEndpoint(`${BASE_URL}/stats`, publicHeaders, "stats");
  sleep(0.3);

  // Hanya sebagian kecil yang sampai login & buat laporan (20%)
  if (Math.random() < 0.20) {
    const auth = authenticate();
    if (!auth) {
      sleep(1);
      return;
    }

    const headers = authHeaders(auth.token);
    const createRes = postEndpoint(
      `${BASE_URL}/reports`,
      randomReport(),
      headers,
      "create_report",
      201
    );

    // Cek ID laporan yang baru dibuat
    const reportId = parseBody(createRes)?.data?.id;
    if (reportId) {
      getEndpoint(`${BASE_URL}/reports/${reportId}`, headers, "get_report");
    }
    sleep(0.5);
  }

  // Simulate user "membaca" sebentar sebelum request berikutnya
  sleep(0.5 + Math.random() * 1.0);
}
