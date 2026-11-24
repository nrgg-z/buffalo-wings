export const API_BASE = "http://localhost:8080/api/v1";

const TOKEN_KEY = "bw_pos_token";
const USER_KEY = "bw_pos_user";

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function getStoredUser() {
  try {
    return JSON.parse(sessionStorage.getItem(USER_KEY) || "null");
  } catch {
    return null;
  }
}

export function setSession({ token, user }) {
  sessionStorage.setItem(TOKEN_KEY, token);
  sessionStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearSession() {
  sessionStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(USER_KEY);
}

export async function request(path, { method = "GET", body, auth = true } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth) {
    const token = getToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  let res;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw { status: 0, code: "network_error", message: "No se pudo conectar con el backend (¿está corriendo en :8080?)." };
  }

  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = { raw: text };
  }

  if (!res.ok) {
    throw {
      status: res.status,
      code: data?.error || "http_error",
      message: data?.message || data?.error || `Error HTTP ${res.status}`,
      retryAfter: res.headers.get("Retry-After"),
    };
  }
  return data;
}

export const api = {
  healthz: () => request("/healthz", { auth: false }),
  publicConfig: () => request("/config/public", { auth: false }),
  login: (employee_code, pin) =>
    request("/auth/login", { method: "POST", body: { employee_code, pin }, auth: false }),
  me: () => request("/auth/me"),

  // Mesas (HU-03)
  getTables: () => request("/tables"),
  assignTable: (id, guests, version) =>
    request(`/tables/${id}/assign`, { method: "POST", body: { guests, version } }),
  mergeTables: (tables) =>
    request("/tables/merge", { method: "POST", body: { tables } }),
  cleanTable: (id, version) =>
    request(`/tables/${id}/clean`, { method: "POST", body: { version } }),
  releaseTable: (id, version) =>
    request(`/tables/${id}/release`, { method: "POST", body: { version } }),

  // Catálogo (HU-04)
  getCatalog: () => request("/catalog"),
  setProductAvailability: (id, is_available) =>
    request(`/catalog/products/${id}/availability`, { method: "PATCH", body: { is_available } }),
setModifierAvailability: (id, is_available) =>
    request(`/catalog/modifiers/${id}/availability`, { method: "PATCH", body: { is_available } }),

  // Comandas (HU-05)
  getTableOrder: (tableId) => request(`/tables/${tableId}/order`),
  createOrAppendOrder: (table_id, items) =>
    request("/orders", { method: "POST", body: { table_id, items } }),
  voidOrderItem: (orderId, itemId, { manager_code, manager_pin, reason }) =>
    request(`/orders/${orderId}/items/${itemId}/void`, {
      method: "POST",
      body: { manager_code, manager_pin, reason },
    }),
    // KDS Cocina y Barra (HU-06)
  getKDSQueue: (station) => request(`/kds/queue?station=${encodeURIComponent(station)}`),
  getKDSReady: () => request("/kds/ready"),
  updateKDSItemStatus: (itemId, status) =>
    request(`/kds/items/${itemId}/status`, { method: "PATCH", body: { status } }),
  // Caja y Facturación (HU-07)
  getPrecheck: (orderId) => request(`/billing/orders/${orderId}/precheck`),
  payOrder: (orderId, { payment_method, tip_amount, amount_received, reference }) =>
    request(`/billing/orders/${orderId}/pay`, {
      method: "POST",
      body: { payment_method, tip_amount, amount_received, reference },
    }),
  getReceipt: (orderId) => request(`/billing/orders/${orderId}/receipt`),
  // Reportes y Cierre Diario (HU-08)
  getDailySummary: (date = "") =>
    request(`/reports/daily-summary${date ? `?date=${encodeURIComponent(date)}` : ""}`),
  getVoidsReport: (date = "") =>
    request(`/reports/voids${date ? `?date=${encodeURIComponent(date)}` : ""}`),
};
