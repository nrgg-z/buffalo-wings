import { api } from "../api.js";

const money = (n) => `$${Number(n || 0).toFixed(2)}`;

const METHOD_META = {
  CASH: { label: "Efectivo", bar: "bg-bw-forest" },
  CARD: { label: "Tarjeta", bar: "bg-blue-600" },
  QR: { label: "Pago QR", bar: "bg-bw-orange" },
};

function formatTime(iso) {
  if (!iso) return "--:--";
  try {
    return new Date(iso).toLocaleTimeString("es-SV", { hour: "2-digit", minute: "2-digit" });
  } catch {
    return iso;
  }
}

export async function renderReports(container, { user }) {
  const canView = ["MANAGER", "ADMIN"].includes(user.role);

  if (!canView) {
    container.innerHTML = `
      <div class="bg-amber-50 border border-amber-200 text-amber-800 rounded-2xl p-6 text-sm font-semibold">
        Acceso restringido: tu rol (${user.role}) no tiene permiso para consultar el cierre diario ni la auditoría de anulaciones. Solo <strong>MANAGER</strong> o <strong>ADMIN</strong>.
      </div>
    `;
    return;
  }

  let selectedDate = ""; // Vacío = fecha actual del backend
  let summary = null;
  let voids = [];
  let errorMsg = "";

  async function load(date = selectedDate) {
    container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Generando reporte diario...</div>`;
    errorMsg = "";
    try {
      const [sumRes, voidsRes] = await Promise.all([
        api.getDailySummary(date),
        api.getVoidsReport(date),
      ]);
      summary = sumRes;
      selectedDate = sumRes.date || date;
      voids = voidsRes.voids || [];
    } catch (err) {
      errorMsg = err.message || "No se pudo cargar el reporte.";
    }
    render();
  }

  function render() {
    if (errorMsg && !summary) {
      container.innerHTML = `
        <div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-semibold flex items-center justify-between">
          <span>${errorMsg}</span>
          <button id="btn-retry-rep" class="px-3 py-1.5 rounded-lg bg-white border border-red-200 text-xs font-bold uppercase">Reintentar</button>
        </div>
      `;
      container.querySelector("#btn-retry-rep")?.addEventListener("click", () => load(selectedDate));
      return;
    }

    const s = summary || {};
    const methods = s.by_payment_method || [];
    const topProducts = s.top_products || [];
    const kds = s.kitchen_metrics || { ready_items_count: 0, avg_prep_minutes: 0 };
    const totalRev = Number(s.total_revenue || 0);

    container.innerHTML = `
      <div class="space-y-5">
        <!-- Encabezado y Filtro de Fecha -->
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-widest text-secondary">Cierre de Caja y Auditoría</div>
            <h1 class="font-display text-2xl uppercase text-bw-ink tracking-wide mt-0.5">Reportes Gerenciales · ${selectedDate || "Hoy"}</h1>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <input id="input-report-date" type="date" value="${selectedDate}"
              class="px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-bold text-on-surface" />
            <button id="btn-report-today" class="px-3 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container text-on-surface hover:bg-surface-container-high">
              Hoy
            </button>
            <button id="btn-report-refresh" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">refresh</span> Actualizar
            </button>
            <button id="btn-report-print" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-bw-ink text-white hover:opacity-90 flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">print</span> Imprimir
            </button>
          </div>
        </div>

        ${errorMsg ? `
          <div class="p-3.5 rounded-xl bg-red-50 border border-red-200 text-error text-xs font-semibold">${errorMsg}</div>
        ` : ""}

        <!-- KPIs Principales -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm">
            <div class="text-[11px] font-bold uppercase tracking-wider text-secondary">Ingresos Totales</div>
            <div class="font-display text-3xl text-bw-forest mt-1">${money(s.total_revenue)}</div>
            <div class="text-xs text-secondary mt-1"><strong>${s.paid_orders || 0}</strong> ${(s.paid_orders || 0) === 1 ? "comanda cobrada" : "comandas cobradas"}</div>
          </div>

          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm">
            <div class="text-[11px] font-bold uppercase tracking-wider text-secondary">Subtotal / IVA / Propinas</div>
            <div class="font-display text-3xl text-bw-ink mt-1">${money(s.subtotal)}</div>
            <div class="text-xs text-secondary mt-1">IVA: <strong>${money(s.tax_total)}</strong> · Propinas: <strong>${money(s.tip_total)}</strong></div>
          </div>

          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm">
            <div class="text-[11px] font-bold uppercase tracking-wider text-secondary">Desempeño KDS</div>
            <div class="font-display text-3xl text-bw-orange mt-1">${Number(kds.avg_prep_minutes || 0).toFixed(1)} min</div>
            <div class="text-xs text-secondary mt-1"><strong>${kds.ready_items_count || 0}</strong> ítems completados</div>
          </div>

          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm">
            <div class="text-[11px] font-bold uppercase tracking-wider text-secondary">Anulaciones (Auditoría)</div>
            <div class="font-display text-3xl text-error mt-1">${s.voided_count || 0}</div>
            <div class="text-xs text-secondary mt-1">Monto anulado: <strong>${money(s.voided_amount)}</strong></div>
          </div>
        </div>

        <!-- Desglose por Método de Pago + Top Productos -->
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-5 items-stretch">
          <div class="lg:col-span-5 bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm space-y-4">
            <h2 class="font-display text-xl uppercase text-bw-ink">Desglose por Método de Pago</h2>
            ${methods.length ? `
              <div class="space-y-3.5">
                ${methods.map((m) => {
                  const meta = METHOD_META[m.method] || { label: m.method, bar: "bg-bw-ink" };
                  const pct = totalRev > 0 ? Math.min(100, Math.round((m.total / totalRev) * 100)) : 0;
                  return `
                    <div class="space-y-1.5">
                      <div class="flex items-center justify-between text-xs">
                        <span class="font-bold text-on-surface">${meta.label} <span class="text-secondary font-normal">(${m.count} ${m.count === 1 ? "pago" : "pagos"})</span></span>
                        <span class="font-bold text-bw-ink">${money(m.total)} <span class="text-secondary font-normal">(${pct}%)</span></span>
                      </div>
                      <div class="w-full h-2.5 rounded-full bg-surface-container-low overflow-hidden">
                        <div class="h-full rounded-full ${meta.bar}" style="width: ${pct}%"></div>
                      </div>
                    </div>
                  `;
                }).join("")}
              </div>
            ` : `
              <p class="text-xs text-secondary py-4">No hay cobros registrados para esta fecha.</p>
            `}
          </div>

          <div class="lg:col-span-7 bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm space-y-4">
            <h2 class="font-display text-xl uppercase text-bw-ink">Top Productos Vendidos</h2>
            ${topProducts.length ? `
              <div class="overflow-x-auto">
                <table class="w-full text-left text-xs">
                  <thead>
                    <tr class="border-b border-surface-container-high text-[10px] uppercase tracking-wider text-secondary">
                      <th class="py-2 pr-3">#</th>
                      <th class="py-2 pr-3">Producto</th>
                      <th class="py-2 pr-3 text-right">Unidades</th>
                      <th class="py-2 text-right">Venta Bruta</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-surface-container-low">
                    ${topProducts.map((p, idx) => `
                      <tr>
                        <td class="py-2.5 pr-3 font-bold text-secondary">${idx + 1}</td>
                        <td class="py-2.5 pr-3 font-bold text-on-surface">${p.product_name}</td>
                        <td class="py-2.5 pr-3 text-right font-semibold">${p.units_sold}</td>
                        <td class="py-2.5 text-right font-bold text-bw-forest">${money(p.revenue)}</td>
                      </tr>
                    `).join("")}
                  </tbody>
                </table>
              </div>
            ` : `
              <p class="text-xs text-secondary py-4">Todavía no hay productos vendidos en esta fecha.</p>
            `}
          </div>
        </div>

        <!-- Auditoría de Anulaciones -->
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="font-display text-xl uppercase text-bw-ink">Auditoría de Ítems Anulados (${voids.length})</h2>
            <span class="text-[11px] font-bold uppercase text-secondary">Autorizados con PIN de Gerente</span>
          </div>

          ${voids.length ? `
            <div class="overflow-x-auto">
              <table class="w-full text-left text-xs">
                <thead>
                  <tr class="border-b border-surface-container-high text-[10px] uppercase tracking-wider text-secondary">
                    <th class="py-2 pr-3">Hora</th>
                    <th class="py-2 pr-3">Mesa / Comanda</th>
                    <th class="py-2 pr-3">Producto</th>
                    <th class="py-2 pr-3">Motivo</th>
                    <th class="py-2 pr-3">Autorizó</th>
                    <th class="py-2 text-right">Monto</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-surface-container-low">
                  ${voids.map((v) => `
                    <tr>
                      <td class="py-2.5 pr-3 text-secondary font-medium">${formatTime(v.created_at)}</td>
                      <td class="py-2.5 pr-3 font-bold">Mesa ${v.table_code} · #${v.order_id}</td>
                      <td class="py-2.5 pr-3 font-semibold">${v.quantity}× ${v.product_name}</td>
                      <td class="py-2.5 pr-3 text-error font-medium">${v.void_reason || "Sin motivo"}</td>
                      <td class="py-2.5 pr-3 text-secondary">${v.voided_by_name || ""} (${v.voided_by_code || ""})</td>
                      <td class="py-2.5 text-right font-bold text-error">${money(v.unit_price * v.quantity)}</td>
                    </tr>
                  `).join("")}
                </tbody>
              </table>
            </div>
          ` : `
            <p class="text-xs text-secondary py-2">No se registraron ítems anulados en esta fecha.</p>
          `}
        </div>
      </div>
    `;

    container.querySelector("#input-report-date")?.addEventListener("change", (e) => {
      if (e.target.value) load(e.target.value);
    });

    container.querySelector("#btn-report-today")?.addEventListener("click", () => {
      load("");
    });

    container.querySelector("#btn-report-refresh")?.addEventListener("click", () => {
      load(selectedDate);
    });

    container.querySelector("#btn-report-print")?.addEventListener("click", () => {
      window.print();
    });
  }

  await load("");
}
