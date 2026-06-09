#!/bin/bash
# ──────────────────────────────────────────────────────────────
# run.sh — Runner script untuk semua skenario k6 TemuKan
#
# Usage:
#   ./run.sh smoke
#   ./run.sh load
#   ./run.sh stress
#   ./run.sh spike
#   ./run.sh all
#   ./run.sh smoke --out json=results/smoke.json   (custom output)
#
# Env override:
#   BASE_URL=http://staging.temukan.id/api/v1 ./run.sh load
# ──────────────────────────────────────────────────────────────

set -euo pipefail

SCENARIO=${1:-smoke}
SHIFT_ARGS="${@:2}"
BASE_URL=${BASE_URL:-"http://localhost:8080/api/v1"}
RESULTS_DIR="results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

mkdir -p "$RESULTS_DIR"

run_scenario() {
  local name=$1
  local file="scenarios/${name}.js"

  if [ ! -f "$file" ]; then
    echo "❌ File tidak ditemukan: $file"
    exit 1
  fi

  echo ""
  echo "╔══════════════════════════════════════════╗"
  echo "║  TemuKan Performance Test                ║"
  echo "║  Skenario : $(printf '%-30s' "$name") ║"
  echo "║  Target   : $(printf '%-30s' "$BASE_URL") ║"
  echo "║  Waktu    : $(printf '%-30s' "$(date '+%H:%M:%S %d/%m/%Y')") ║"
  echo "╚══════════════════════════════════════════╝"
  echo ""

  k6 run \
    --env BASE_URL="$BASE_URL" \
    --out "json=${RESULTS_DIR}/${name}_${TIMESTAMP}.json" \
    $SHIFT_ARGS \
    "$file"

  echo ""
  echo "✅ Selesai: hasil disimpan di ${RESULTS_DIR}/${name}_${TIMESTAMP}.json"
}

case "$SCENARIO" in
  smoke)   run_scenario smoke ;;
  load)    run_scenario load ;;
  stress)  run_scenario stress ;;
  spike)   run_scenario spike ;;
  all)
    echo "🚀 Menjalankan semua skenario secara berurutan..."
    run_scenario smoke
    echo "⏳ Jeda 30 detik sebelum load test..."
    sleep 30
    run_scenario load
    echo "⏳ Jeda 60 detik sebelum stress test..."
    sleep 60
    run_scenario stress
    echo "⏳ Jeda 60 detik sebelum spike test..."
    sleep 60
    run_scenario spike
    echo ""
    echo "🎉 Semua skenario selesai! Cek folder: $RESULTS_DIR/"
    ;;
  *)
    echo "Usage: ./run.sh [smoke|load|stress|spike|all]"
    exit 1
    ;;
esac
