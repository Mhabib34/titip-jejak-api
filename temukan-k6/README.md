# TemuKan — k6 Performance Test Suite

Script performance test untuk API TemuKan menggunakan [k6](https://k6.io).

## Prasyarat

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Windows (via Chocolatey)
choco install k6
```

## Struktur Project

```
temukan-k6/
├── config/
│   └── index.js          # Base URL, thresholds, stage definitions
├── helpers/
│   ├── auth.js           # Auto-register + login, header builder
│   ├── data.js           # Random payload generator
│   └── checks.js         # Request wrappers + assertion helpers
├── scenarios/
│   ├── smoke.js          # 2 VU, 1 menit — sanity check semua endpoint
│   ├── load.js           # 100–200 VU — beban normal, weighted journey
│   ├── stress.js         # Sampai 800 VU — batas maksimal server
│   └── spike.js          # Lonjakan tiba-tiba 10 → 500 VU
├── results/              # Output JSON (auto-dibuat saat run)
└── run.sh                # Runner script
```

## Cara Menjalankan

```bash
# Beri permission execute
chmod +x run.sh

# Jalankan smoke test dulu (sanity check)
./run.sh smoke

# Load test
./run.sh load

# Stress test
./run.sh stress

# Spike test
./run.sh spike

# Semua skenario berurutan (dengan jeda otomatis)
./run.sh all
```

### Override Base URL

```bash
# Target staging
BASE_URL=http://staging.temukan.id/api/v1 ./run.sh load

# Atau langsung dengan k6
k6 run --env BASE_URL=http://localhost:8080/api/v1 scenarios/smoke.js
```

### Output ke Grafana / InfluxDB

```bash
k6 run --out influxdb=http://localhost:8086/k6 scenarios/load.js
```

## Skenario & Thresholds

| Skenario | VU | Durasi | p95 | Error Rate |
|----------|----|--------|-----|------------|
| Smoke    | 2  | ~1 menit | < 1s | < 5% |
| Load     | 50–200 | ~9 menit | < 1s | < 5% |
| Stress   | 100–800 | ~10 menit | < 2s | < 10% |
| Spike    | 10→500→10 | ~4 menit | < 3s | < 15% |

## User Journey (Load Test)

```
60% → Browser Journey
      map/pins → list reports (dengan filter) → stats

30% → Reporter Journey
      create report → get report → update → delete

10% → Notification Journey
      GET /notifications → GET /matches
```

## Tips

- **Jalankan smoke dulu** sebelum load/stress. Kalau smoke gagal, load test tidak ada gunanya.
- **Perhatikan error log** di terminal. k6 print error rate real-time.
- **Cleanup DB**: script otomatis hapus laporan test setelah dibuat (di load & stress).
- **Jeda antar skenario**: `./run.sh all` otomatis beri jeda 30–60 detik antar skenario agar server recovery.
- **Hasil JSON** tersimpan di folder `results/` dengan timestamp, bisa dianalisis dengan k6 cloud atau Grafana.

## Interpretasi Hasil

```
✓ checks.........................: 98.50%  # target > 95%
✓ http_req_duration p(95).......: 743ms   # target < 1000ms
✓ http_req_failed...............: 1.50%   # target < 5%

http_reqs......................: 12430   # total requests
http_req_duration avg..........: 245ms
```

- **checks < 95%** → ada endpoint yang error, cek log
- **p95 mendekati threshold** → server mulai kewalahan
- **http_req_failed tinggi** → 5xx errors, kemungkinan OOM/DB bottleneck
