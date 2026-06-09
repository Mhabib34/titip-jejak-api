// ──────────────────────────────────────────────────────────────
// helpers/checks.js — Wrapper request + assertion standar
// ──────────────────────────────────────────────────────────────
import http  from "k6/http";
import { check } from "k6";

// GET dengan tag endpoint
export function getEndpoint(url, headers, endpointName) {
  const res = http.get(url, { headers, tags: { endpoint: endpointName } });
  check(res, {
    [`${endpointName}: status 2xx`]: (r) => r.status >= 200 && r.status < 300,
    [`${endpointName}: body ada`]:    (r) => r.body && r.body.length > 0,
    [`${endpointName}: status OK`]:   (r) => {
      try { return JSON.parse(r.body)?.status === "OK"; }
      catch { return false; }
    },
  });
  return res;
}

export function postEndpoint(url, payload, headers, endpointName, expectedStatus = 201) {
  const res = http.post(url, JSON.stringify(payload), {
    headers, tags: { endpoint: endpointName },
  });
  check(res, {
    [`${endpointName}: status ${expectedStatus}`]: (r) => r.status === expectedStatus,
    [`${endpointName}: status OK`]: (r) => {
      try { return JSON.parse(r.body)?.status === "OK"; }
      catch { return false; }
    },
  });
  return res;
}

export function putEndpoint(url, payload, headers, endpointName) {
  const res = http.put(url, JSON.stringify(payload), { headers, tags: { endpoint: endpointName } });
  check(res, {
    [`${endpointName}: status 200`]:  (r) => r.status === 200,
    [`${endpointName}: status OK`]:   (r) => {
      try { return JSON.parse(r.body)?.status === "OK"; }
      catch { return false; }
    },
  });
  return res;
}

export function deleteEndpoint(url, headers, endpointName) {
  const res = http.del(url, null, { headers, tags: { endpoint: endpointName } });
  check(res, {
    [`${endpointName}: status 200`]:  (r) => r.status === 200,
    [`${endpointName}: status OK`]:   (r) => {
      try { return JSON.parse(r.body)?.status === "OK"; }
      catch { return false; }
    },
  });
  return res;
}

export function patchEndpoint(url, payload, headers, endpointName) {
  const body = payload ? JSON.stringify(payload) : null;
  const res  = http.patch(url, body, { headers, tags: { endpoint: endpointName } });
  check(res, {
    [`${endpointName}: status 200`]:  (r) => r.status === 200,
    [`${endpointName}: status OK`]:   (r) => {
      try { return JSON.parse(r.body)?.status === "OK"; }
      catch { return false; }
    },
  });
  return res;
}

export function parseBody(res) {
  try { return JSON.parse(res.body); }
  catch { return null; }
}