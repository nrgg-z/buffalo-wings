import { api, clearSession, getToken } from "./api.js";
import { renderLogin } from "./views/login.js";
import { renderTables } from "./views/tables.js";
import { renderCatalog } from "./views/catalog.js";
import { renderOrders } from "./views/orders.js";
import { renderKDS } from "./views/kds.js";
import { renderReports } from "./views/reports.js";
import { renderBilling } from "./views/billing.js";

const appEl = document.getElementById("app");
let state = {
  user: null,
  config: { inactivity_timeout_sec: 180, tax_rate: 0.13, suggested_tip_rate: 0.1 },
  activeTab: "mesas",
  selectedTable: null,
  selectedOrder: null,
};
let inactivityTimer = null;
let clockInterval = null;
let healthInterval = null;
let activeViewCleanup = null;

function cleanupActiveView() {
  if (typeof activeViewCleanup === "function") {
    activeViewCleanup();
    activeViewCleanup = null;
  }
}

function resetInactivityTimer() {
  if (!state.user) return;
  clearTimeout(inactivityTimer);
  const ms = (state.config?.inactivity_timeout_sec || 180) * 1000;
  inactivityTimer = setTimeout(() => logout("Sesión cerrada por inactividad (180s)."), ms);
}

["click", "keydown", "touchstart"].forEach((evt) =>
  window.addEventListener(evt, resetInactivityTimer, { passive: true })
);

function logout(notice = "") {
  cleanupActiveView();
  clearTimeout(inactivityTimer);
  clearInterval(clockInterval);
  clearSession();
  state.user = null;
  state.selectedTable = null;
  state.selectedOrder = null;
  renderLogin(appEl, {
    notice,
    onSuccess: (user) => {
      state.user = user;
      state.activeTab = user.role === "KITCHEN" ? "kds-cocina" : user.role === "CASHIER" ? "caja" : "mesas";
      resetInactivityTimer();
      renderShell();
    },
  });
}

function renderActiveView() {
  cleanupActiveView();
  const root = document.getElementById("view-root");
  if (!root) return;

  if (state.activeTab === "mesas") {
    renderTables(root, {
      user: state.user,
      onOpenOrder: (table) => {
        state.selectedTable = table;
        state.activeTab = "comandas";
        renderShell();
      },
    });
    return;
  }

  if (state.activeTab === "catalogo") {
    renderCatalog(root, { user: state.user });
    return;
  }

  if (state.activeTab === "comandas") {
    renderOrders(root, {
      user: state.user,
      initialTable: state.selectedTable,
      onSelectTable: (table) => {
        state.selectedTable = table;
      },
      onGoToBilling: (table, order) => {
        state.selectedTable = table;
        state.selectedOrder = order;
        state.activeTab = "caja";
        renderShell();
      },
    });
    return;
  }

  if (state.activeTab === "kds-cocina") {
    activeViewCleanup = renderKDS(root, { user: state.user, station: "KITCHEN" });
    return;
  }
if (state.activeTab === "reportes") {
  renderReports(root, { user: state.user });
  return;
}
  if (state.activeTab === "caja") {
  renderBilling(root, {
    user: state.user,
    initialTable: state.selectedTable,
    initialOrder: state.selectedOrder,
    onGoToTables: () => {
      state.selectedTable = null;
      state.selectedOrder = null;
      state.activeTab = "mesas";
      renderShell();
    },
    
  });
  return;
}

  if (state.activeTab === "kds-barra") {
    activeViewCleanup = renderKDS(root, { user: state.user, station: "BAR" });
    return;
  }

  root.innerHTML = `
    <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-6 shadow-sm">
      <h2 class="font-display text-2xl uppercase text-bw-ink">Módulo: ${state.activeTab}</h2>
      ${state.selectedOrder ? `<p class="text-sm text-bw-orange font-semibold mt-1">Comanda activa: #${state.selectedOrder.id} (Mesa ${state.selectedTable?.code})</p>` : ""}
      <p class="text-sm text-secondary mt-2">Listo para conectar en el siguiente paso.</p>
    </div>
  `;
}

async function refreshBackendStatus() {
  const dot = document.getElementById("backend-status-dot");
  const label = document.getElementById("backend-status-label");
  if (!dot || !label) return;

  let online = false;
  try {
    const health = await api.healthz();
    online = health?.status === "ok" && health?.db_status === "up";
  } catch {
    online = false;
  }

  dot.classList.toggle("bg-bw-forest", online);
  dot.classList.toggle("bg-error", !online);
  dot.classList.toggle("animate-pulse", online);
  label.textContent = online ? "En línea · Multiplaza" : "Sin conexión · Multiplaza";
}

function renderShell() {
  clearInterval(clockInterval);
  clearInterval(healthInterval);
  const tabs = [
    { id: "mesas", label: "Mesas" },
    { id: "comandas", label: "Comandas" },
    { id: "kds-cocina", label: "Cocina KDS" },
    { id: "kds-barra", label: "Barra KDS" },
    { id: "caja", label: "Caja" },
    { id: "catalogo", label: "Catálogo" },
    { id: "reportes", label: "Reportes" },
  ];

  appEl.innerHTML = `
    <header class="fixed top-0 left-0 right-0 z-50 h-16 bg-surface-container-lowest/95 backdrop-blur border-b border-surface-container-high px-4 lg:px-6 flex items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-lg bg-bw-orange text-white font-display flex items-center justify-center text-lg">BW</div>
          <div class="flex flex-col leading-none">
            <span class="font-display text-base text-bw-ink tracking-wider uppercase">Buffalo Wings</span>
            <span class="text-[10px] font-bold text-secondary tracking-wider uppercase">El Salvador · POS</span>
          </div>
        </div>
        <nav class="hidden md:flex items-center gap-1 p-1 bg-surface-container-low rounded-xl">
          ${tabs.map((t) => `
            <button data-tab="${t.id}" class="nav-tab px-3 py-1.5 rounded-lg text-xs font-bold uppercase tracking-wider transition ${
              state.activeTab === t.id ? "bg-bw-orange text-white" : "text-on-surface-variant hover:text-on-surface"
            }">${t.label}</button>
          `).join("")}
        </nav>
      </div>

      <div class="flex items-center gap-3">
        <div class="hidden xl:flex items-center gap-2 bg-surface-container-low px-3 py-1.5 rounded-lg text-xs font-medium">
          <span id="backend-status-dot" class="w-2.5 h-2.5 rounded-full bg-secondary"></span>
          <span id="backend-status-label">Verificando · Multiplaza</span>
        </div>
        <div class="flex items-center gap-1.5 bg-surface-container-low px-3 py-1.5 rounded-lg text-xs font-bold">
          <span class="material-symbols-outlined text-base text-secondary">schedule</span>
          <span id="pos-clock">--:--</span>
        </div>
        <div class="flex items-center gap-2 bg-surface-container-low px-3 py-1.5 rounded-xl">
          <div class="text-right leading-tight">
            <div class="text-xs font-bold text-on-surface">${state.user.full_name}</div>
            <div class="text-[10px] font-bold text-bw-orange uppercase tracking-wider">${state.user.role} · ${state.user.employee_code}</div>
          </div>
          <button id="btn-logout" title="Cerrar sesión" class="p-1.5 rounded-lg hover:bg-red-50 text-error transition">
            <span class="material-symbols-outlined text-lg">logout</span>
          </button>
        </div>
      </div>
    </header>

    <main id="view-root" class="pt-20 pb-10 px-4 lg:px-6 max-w-7xl mx-auto"></main>
  `;

  appEl.querySelector("#btn-logout").addEventListener("click", () => logout());
  appEl.querySelectorAll(".nav-tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      state.activeTab = btn.dataset.tab;
      renderShell();
    });
  });

  const clockEl = appEl.querySelector("#pos-clock");
  const tick = () => {
    if (clockEl) clockEl.textContent = new Date().toLocaleTimeString("es-SV", { hour: "2-digit", minute: "2-digit" });
  };
  tick();
  clockInterval = setInterval(tick, 15000);
  refreshBackendStatus();
  healthInterval = setInterval(refreshBackendStatus, 15000);

  renderActiveView();
}

async function boot() {
  try { state.config = await api.publicConfig(); } catch {}
  if (getToken()) {
    try {
      const meRes = await api.me();
      state.user = meRes.user || meRes;
      resetInactivityTimer();
      renderShell();
      return;
    } catch { clearSession(); }
  }
  logout();
}

boot();
