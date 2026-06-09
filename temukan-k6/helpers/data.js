// ─────────────────────────────────────────────────────────────
// helpers/data.js — Generator payload random untuk laporan
// ─────────────────────────────────────────────────────────────

const REPORT_TYPES  = ["found", "missing"];
const GENDERS       = ["male", "female", "unknown"];
const DESCRIPTIONS  = [
  "Pria tua, rambut putih, pakai baju batik biru, terlihat kebingungan",
  "Wanita paruh baya, berkerudung merah, membawa tas belanja hijau",
  "Pria muda, kaos hitam, celana jeans, berbicara sendiri",
  "Nenek-nenek, pakai daster bunga kuning, tidak ingat alamat rumah",
  "Kakek, baju koko putih, peci hitam, duduk di halte tidak mau bergerak",
  "Wanita tua, rambut disemir hitam, berbicara bahasa daerah Batak",
  "Pria dewasa, luka di tangan kiri, tidak membawa identitas",
  "Perempuan muda, terlihat linglung, membawa boneka kecil",
];
const LOCATIONS     = [
  "Depan Pasar Petisah, Jalan Pegadaian",
  "Halte Trans Mebidang, Jalan Gatot Subroto",
  "Taman Ahmad Yani, dekat air mancur",
  "Masjid Raya Al-Mashun, pintu timur",
  "Stasiun Medan, peron 2",
  "Plaza Medan Fair, pintu masuk utara",
  "Rumah Sakit Adam Malik, IGD",
  "Simpang Pos, depan kantor pos",
];

// Koordinat sekitar Medan
function randomCoord() {
  const lat = 3.4 + Math.random() * 0.4;   // ~3.40 – 3.80
  const lng = 98.5 + Math.random() * 0.4;  // ~98.50 – 98.90
  return {
    latitude:  parseFloat(lat.toFixed(6)),
    longitude: parseFloat(lng.toFixed(6)),
  };
}

function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

export function randomReport() {
  const { latitude, longitude } = randomCoord();
  return {
    type:               pick(REPORT_TYPES),
    gender:             pick(GENDERS),
    estimated_age:      Math.floor(Math.random() * 80) + 10,  // 10–90
    description:        pick(DESCRIPTIONS),
    last_seen_location: pick(LOCATIONS),
    city:               "Medan",
    province:           "Sumatera Utara",
    latitude,
    longitude,
    name:               Math.random() > 0.5 ? `Pak/Bu ${__VU}` : null,
  };
}

// Payload update minimal (patch-style)
export function randomUpdatePayload() {
  return {
    description: `Update dari VU ${__VU} iter ${__ITER} — ${Date.now()}`,
    status:      Math.random() > 0.8 ? "resolved" : "active",
  };
}
