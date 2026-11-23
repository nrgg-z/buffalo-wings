import { api } from "../api.js";

const ALERT_STYLES = {
  GREEN: {
    border: "border-emerald-300",
    header: "bg-emerald-50 text-bw-forest",
    badge: "bg-emerald-100 text-bw-forest",
    label: "A tiempo",
  },
  YELLOW: {
    border: "border-amber-300",
    header: "bg-amber-50 text-amber-800",
    badge: "bg-amber-100 text-amber-800",
    label: "Demorado",
  },
  RED: {
    border: "border-red-400",
    header: "bg-red-50 text-error",
    badge: "bg-red-100 text-error",
    label: "Crítico (+15m)",
  },
};

export function renderKDS(container, { user, station }) {
  const isKitchen = station === "KITCHEN";
  const title = isKitchen ? "Cocina KDS" : "Barra KDS";
  const subtitle = isKitchen ? "Estación de Alitas, Boneless y Platos" : "Estación de Bebidas y Cervezas";
  const canSeeQueue = ["KITCHEN", "MANAGER", "ADMIN"].includes(user.role);
  const canOperate = ["KITCHEN", "WAITER", "MANAGER", "ADMIN"].includes(user.role);

  let queueItems = [];
  let readyItems = [];
  let feedback = null;
  let lastUpdated = null;
  let destroyed = false;

  async function load(silent = false) {
    if (destroyed) return;
    if (!silent) {
      container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Cargando ${title}...</div>`;
    }
    try {
      const [qRes, rRes] = await Promise.all([
        canSeeQueue ? api.getKDSQueue(station) : Promise.resolve({ items: [] }),
        api.getKDSReady(),
      ]);
      if (destroyed) return;
      queueItems = qRes.items || [];
      readyItems = (rRes.items || []).filter((it) => it.station === station);
      lastUpdated = new Date().toLocaleTimeString("es-SV", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
      render();
    } catch (err) {
      if (destroyed) return;
      if (!silent) {
        container.innerHTML = `<div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-medium">${err.message}</div>`;
      } else {
        feedback = { type: "err", text: err.message };
        render();
      }
    }
  }

  function groupByOrder(items) {
    const map = new Map();
    for (const it of items) {
      if (!map.has(it.order_id)) {
        map.set(it.order_id, {
          order_id: it.order_id,
          table_code: it.table_code,
          maxElapsed: it.elapsed_minutes || 0,
          alert_level: it.alert_level || "GREEN",
          items: [],
        });
      }
      const g = map.get(it.order_id);
      g.items.push(it);
      if ((it.elapsed_minutes || 0) > g.maxElapsed) {
        g.maxElapsed = it.elapsed_minutes;
        g.alert_level = it.alert_level || g.alert_level;
      }
    }
    return [...map.values()];
  }

  function render() {
    const tickets = groupByOrder(queueItems);
    const pendingCount = queueItems.filter((i) => i.status === "PENDING").length;
    const progressCount = queueItems.filter((i) => i.status === "IN_PROGRESS").length;
    const readyCount = readyItems.length;

    container.innerHTML = `
      <div class="space-y-5">
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-widest text-secondary">${subtitle}</div>
            <h1 class="font-display text-2xl uppercase text-bw-ink tracking-wide mt-0.5">${title}</h1>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <span class="px-3 py-1.5 rounded-full text-xs font-bold bg-amber-50 text-amber-800 border border-amber-200">
              Pendientes: ${pendingCount}
            </span>
            <span class="px-3 py-1.5 rounded-full text-xs font-bold bg-blue-50 text-blue-700 border border-blue-200">
              En preparación: ${progressCount}
            </span>
            <span class="px-3 py-1.5 rounded-full text-xs font-bold bg-emerald-50 text-bw-forest border border-emerald-200">
              Listos: ${readyCount}
            </span>
            <button id="btn-kds-refresh" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">refresh</span> Actualizar
            </button>
          </div>
        </div>

        <div class="flex items-center justify-between text-[11px] text-secondary px-1">
          <span>Auto-actualización cada 10s · Semáforo: Verde (&lt;10m), Amarillo (10-14m), Rojo (15m+)</span>
          <span>${lastUpdated ? `Última sincronización: ${lastUpdated}` : ""}</span>
        </div>

        ${feedback ? `
          <div class="p-3.5 rounded-xl border text-xs font-semibold flex items-center justify-between ${
            feedback.type === "ok" ? "bg-emerald-50 border-emerald-200 text-bw-forest" : "bg-red-50 border-red-200 text-error"
          }">
            <span>${feedback.text}</span>
            <button id="close-kds-fb" class="ml-4 opacity-70 hover:opacity-100">✕</button>
          </div>
        ` : ""}

        ${canSeeQueue ? `
          <section class="space-y-3">
            <h2 class="font-display text-xl uppercase text-bw-ink">Cola de Producción (${tickets.length} ${tickets.length === 1 ? "ticket" : "tickets"})</h2>

            ${tickets.length ? `
              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 items-start">
                ${tickets.map((t) => {
                  const st = ALERT_STYLES[t.alert_level] || ALERT_STYLES.GREEN;
                  return `
                    <div class="bg-surface-container-lowest rounded-2xl border-2 ${st.border} shadow-sm overflow-hidden">
                      <div class="px-4 py-3 ${st.header} flex items-center justify-between border-b border-surface-container-high">
                        <div>
                          <div class="font-display text-2xl leading-none">Mesa ${t.table_code}</div>
                          <div class="text-[11px] font-bold uppercase opacity-80">Comanda #${t.order_id}</div>
                        </div>
                        <div class="text-right">
                          <span class="inline-block px-2.5 py-1 rounded-full text-xs font-bold ${st.badge}">
                            ${t.maxElapsed} min
                          </span>
                          <div class="text-[10px] font-bold uppercase mt-0.5">${st.label}</div>
                        </div>
                      </div>

                      <div class="p-4 space-y-3">
                        ${t.items.map((it) => `
                          <div class="p-3 rounded-xl bg-surface-container-low/70 border border-surface-container-high space-y-2">
                            <div class="flex items-start justify-between gap-2">
                              <div>
                                <div class="font-bold text-sm text-on-surface">${it.quantity}× ${it.product_name}</div>
                                ${(it.modifiers || []).length ? `
                                  <div class="text-xs text-secondary mt-0.5">${it.modifiers.join(" · ")}</div>
                                ` : ""}
                                ${it.notes ? `
                                  <div class="text-xs font-semibold text-bw-orange mt-1">Nota: ${it.notes}</div>
                                ` : ""}
                              </div>
                              <span class="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase border ${
                                it.status === "IN_PROGRESS"
                                  ? "bg-blue-50 text-blue-700 border-blue-200"
                                  : "bg-amber-50 text-amber-800 border-amber-200"
                              }">
                                ${it.status === "IN_PROGRESS" ? "Preparando" : "Pendiente"}
                              </span>
                            </div>

                            ${canOperate ? `
                              <div class="flex gap-2 pt-1">
                                ${it.status === "PENDING" ? `
                                  <button data-item-id="${it.id}" data-next="IN_PROGRESS" class="btn-kds-step flex-1 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold uppercase transition">
                                    Iniciar
                                  </button>
                                ` : ""}
                                <button data-item-id="${it.id}" data-next="READY" class="btn-kds-step flex-1 py-2 rounded-xl bg-bw-forest hover:opacity-90 text-white text-xs font-bold uppercase transition">
                                  Marcar Listo
                                </button>
                              </div>
                            ` : ""}
                          </div>
                        `).join("")}
                      </div>
                    </div>
                  `;
                }).join("")}
              </div>
            ` : `
              <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-8 text-center text-sm text-secondary">
                No hay pedidos pendientes en <strong>${title}</strong>.
              </div>
            `}
          </section>
        ` : `
          <div class="bg-amber-50 border border-amber-200 text-amber-800 rounded-2xl p-4 text-xs font-semibold">
            Tu rol (${user.role}) visualiza la bandeja de ítems listos para entregar.
          </div>
        `}

        <section class="space-y-3 pt-2">
          <h2 class="font-display text-xl uppercase text-bw-forest">Listos para Entregar (${readyItems.length})</h2>
          ${readyItems.length ? `
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              ${readyItems.map((it) => `
                <div class="bg-emerald-50/60 border border-emerald-200 rounded-2xl p-4 flex items-center justify-between gap-3">
                  <div class="space-y-0.5">
                    <div class="text-xs font-bold uppercase text-bw-forest">Mesa ${it.table_code} · Comanda #${it.order_id}</div>
                    <div class="font-bold text-sm text-on-surface">${it.quantity}× ${it.product_name}</div>
                    ${(it.modifiers || []).length ? `<div class="text-xs text-secondary">${it.modifiers.join(" · ")}</div>` : ""}
                  </div>
                  ${canOperate ? `
                    <button data-item-id="${it.id}" data-next="DELIVERED" class="btn-kds-step shrink-0 px-3.5 py-2 rounded-xl bg-bw-ink hover:opacity-90 text-white text-xs font-bold uppercase">
                      Entregado
                    </button>
                  ` : ""}
                </div>
              `).join("")}
            </div>
          ` : `
            <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 text-xs text-secondary">
              No hay ítems esperando entrega en esta estación.
            </div>
          `}
        </section>
      </div>
    `;

    container.querySelector("#close-kds-fb")?.addEventListener("click", () => {
      feedback = null;
      render();
    });

    container.querySelector("#btn-kds-refresh")?.addEventListener("click", () => {
      feedback = null;
      load(false);
    });

    container.querySelectorAll(".btn-kds-step").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const id = Number(btn.dataset.itemId);
        const next = btn.dataset.next;
        btn.disabled = true;
        try {
          const updated = await api.updateKDSItemStatus(id, next);
          const labels = { IN_PROGRESS: "EN PREPARACIÓN", READY: "LISTO", DELIVERED: "ENTREGADO" };
          feedback = { type: "ok", text: `${updated.product_name} (Mesa ${updated.table_code}) marcado como ${labels[next] || next}.` };
          await load(true);
        } catch (err) {
          feedback = { type: "err", text: err.message };
          render();
        }
      });
    });
  }

  load(false);
  const poll = setInterval(() => load(true), 10000);

  return () => {
    destroyed = true;
    clearInterval(poll);
  };
}