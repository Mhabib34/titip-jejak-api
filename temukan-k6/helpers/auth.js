// ─────────────────────────────────────────────────────────────
// helpers/auth.js — Auto-register & login, token management
// ─────────────────────────────────────────────────────────────
import http    from "k6/http";
import { check, sleep } from "k6";
import { BASE_URL }     from "../config/index.js";

const ROLES = ["finder", "seeker", "volunteer"];

// Buat kredensial unik per VU supaya tidak tabrakan di DB
export function makeCredentials() {
  const uid  = `${__VU}_${__ITER}_${Date.now()}`;
  const role = ROLES[__VU % ROLES.length];
  return {
    name:     `TestUser_${uid}`,
    email:    `testuser_${uid}@temukan-perf.test`,
    password: "Password123!",
    role,
    phone:    "081234567890",
  };
}

// Register → login → kembalikan { token, userId, email, role }
// Kalau register 409 (email sudah ada), langsung coba login.
export function authenticate() {
  const creds = makeCredentials();
  const headers = {
    "Content-Type":  "application/json",
    "X-Client-Type": "mobile", // pakai mobile agar token ada di response body
  };

  // 1. Register
  const regRes = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify(creds),
    { headers, tags: { endpoint: "register" } }
  );

  const regOk = check(regRes, {
    "register: status 201 atau 409": (r) => [201, 409].includes(r.status),
  });

  // 2. Login (selalu — ambil token fresh)
  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: creds.email, password: creds.password }),
    { headers, tags: { endpoint: "login" } }
  );

  const loginOk = check(loginRes, {
    "login: status 200":    (r) => r.status === 200,
    "login: ada token":     (r) => {
      try {
        const body = JSON.parse(r.body);
        return body?.data?.tokens?.access_token !== undefined;
      } catch { return false; }
    },
  });

  if (!loginOk) {
    console.error(`[VU ${__VU}] Login gagal: ${loginRes.status} — ${loginRes.body}`);
    return null;
  }

  const body  = JSON.parse(loginRes.body);
  const token = body.data.tokens.access_token;
  const userId= body.data.user.id;

  return { token, userId, email: creds.email, role: creds.role };
}

// Header builder dengan Bearer token
export function authHeaders(token) {
  return {
    "Content-Type":   "application/json",
    "X-Client-Type":  "mobile",
    "Authorization":  `Bearer ${token}`,
  };
}

// Header publik (tanpa auth)
export const publicHeaders = {
  "Content-Type":  "application/json",
  "X-Client-Type": "mobile",
};
