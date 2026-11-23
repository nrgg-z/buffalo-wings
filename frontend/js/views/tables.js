import { api } from "../api.js";

const STATUS_META = {
  LIBRE: { label: "Libre", dot: "bg-bw-forest", badge: "bg-emerald-50 text-bw-forest border-emerald-200", ring: "hover:border-bw-forest" },
  OCUPADA: { label: "Ocupada", dot: "bg-error", badge: "bg-red-50 text-error border-red-200", ring: "hover:border-error" },
  POR_LIMPIAR: { label: "Por limpiar", dot: "bg-bw-yellow", badge: "bg-amber-50 text-amber-800 border-amber-200", ring: "hover:border-bw-yellow" },
  RESERVADA: { label: "Reservada", dot: "bg-blue-600", badge: "bg-blue-50 text-blue-700 border-blue-200", ring: "hover:border-blue-500" },
};

const ERROR_MESSAGES = {
  version_conflict: "Otro usuario modificó esta mesa. El mapa se actualizó; vuelve a intentarlo.",
  over_capacity: "La cantidad de personas supera la capacidad efectiva de la mesa.",
  invalid_guests: "Ingresa un número válido de personas (mínimo 1).",
  table_not_free: "La mesa ya no está libre.",
  table_merged: "Esta mesa está combinada con otra principal.",
  merge_not_allowed: "Solo puedes combinar mesas LIBRES de la misma zona con códigos consecutivos (ej. M-04 y M-05).",
  invalid_merge: "Selecciona al menos 2 mesas diferentes para combinar.",
  table_not_cleaning: "Solo se pueden limpiar mesas en estado POR_LIMPIAR.",
};

export async function renderTables(container, { user, onOpenOrder }) {
  let tables = [];
  let selectedZone = "TODAS";
  let mergeMode = false;
  let mergeSelection = []; // [{ id, version, code }]
  let feedback = null; // { type: 'ok'|'err', text }
  let assignModalTable = null;

  const canAssignOrMerge = ["HOSTESS", "MANAGER", "ADMIN"].includes(user.role);
  const canClean = ["WAITER", "HOSTESS", "MANAGER", "ADMIN"].includes(user.role);

  function setFeedback(type, text) {
    feedback = { type, text };
  }

  async function loadTables(keepFeedback = false) {
    if (!keepFeedback) feedback = null;
    container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Cargando mapa de mesas...</div>`;
    try {
      const res = await api.getTables();
      tables = res.tables || [];
      mergeSelection = [];
      render();
    } catch (err) {
      const msg =
        err.status === 403
          ? `Tu rol (${user.role}) no tiene permiso para ver el mapa de mesas. Inicia sesión como HOSTESS, WAITER o MANAGER.`
          : err.message || "Error al cargar mesas.";
      container.innerHTML = `
        <div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-medium">
          ${msg}
        </div>
      `;
    }
  }

  function render() {
    const counts = { LIBRE: 0, OCUPADA: 0, POR_LIMPIAR: 0, RESERVADA: 0 };
    tables.forEach((t) => {
      if (counts[t.status] !== undefined) counts[t.status]++;
    });

    const zones = ["TODAS", ...new Set(tables.map((t) => t.zone).filter(Boolean))];
    const visibleTables = selectedZone === "TODAS" ? tables : tables.filter((t) => t.zone === selectedZone);

    container.innerHTML = `
      <div class="space-y-5">
        <!-- Barra superior de salón y contadores -->
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm flex flex-col lg:flex-row lg:items-center justify-between gap-4">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-widest text-secondary">Restaurante Multiplaza · Control de Salón</div>
            <h1 class="font-display text-2xl uppercase text-bw-ink tracking-wide mt-0.5">Mapa de Mesas</h1>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            ${Object.entries(STATUS_META).map(([key, meta]) => `
              <div class="flex items-center gap-2 bg-surface-container-low px-3.5 py-1.5 rounded-full text-xs font-semibold">
                <span class="w-2.5 h-2.5 rounded-full ${meta.dot}"></span>
                <span>${meta.label}: ${counts[key]}</span>
              </div>
            `).join("")}
          </div>
        </div>

        ${feedback ? `
          <div class="p-3.5 rounded-xl border text-xs font-semibold flex items-center justify-between ${
            feedback.type === "ok"
              ? "bg-emerald-50 border-emerald-200 text-bw-forest"
              : "bg-red-50 border-red-200 text-error"
          }">
            <span>${feedback.text}</span>
            <button id="close-feedback" class="ml-4 opacity-70 hover:opacity-100">✕</button>
          </div>
        ` : ""}

        <!-- Filtros y acciones -->
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-1.5">
            ${zones.map((z) => `
              <button data-zone="${z}" class="zone-btn px-3.5 py-1.5 rounded-xl text-xs font-bold uppercase tracking-wider transition ${
                selectedZone === z
                  ? "bg-bw-ink text-white"
                  : "bg-surface-container-lowest border border-surface-container-high text-secondary hover:text-on-surface"
              }">${z.replaceAll("_", " ")}</button>
            `).join("")}
          </div>

          <div class="flex items-center gap-2">
            ${canAssignOrMerge ? `
              <button id="btn-toggle-merge" class="px-4 py-2 rounded-xl text-xs font-bold uppercase tracking-wider border transition ${
                mergeMode
                  ? "bg-bw-orange text-white border-bw-orange"
                  : "bg-surface-container-lowest border-surface-container-high text-bw-ink hover:bg-surface-container-low"
              }">
                ${mergeMode ? `Cancelar combinación (${mergeSelection.length})` : "Combinar Mesas"}
              </button>
              ${mergeMode && mergeSelection.length >= 2 ? `
                <button id="btn-confirm-merge" class="px-4 py-2 rounded-xl text-xs font-bold uppercase tracking-wider bg-bw-forest text-white hover:opacity-90 transition">
                  Confirmar (${mergeSelection.map((m) => m.code).join(" + ")})
                </button>
              ` : ""}
            ` : ""}
            <button id="btn-refresh-tables" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase tracking-wider bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">refresh</span>
              Actualizar
            </button>
          </div>
        </div>

        ${mergeMode ? `
          <div class="p-3 rounded-xl bg-amber-50 border border-amber-200 text-amber-900 text-xs font-medium">
            Modo combinación activo: toca 2 o más mesas <strong>LIBRES</strong> de la misma zona con códigos consecutivos (por ejemplo <strong>M-04</strong> y <strong>M-05</strong>) y luego presiona <strong>Confirmar</strong>.
          </div>
        ` : ""}

        <!-- Grid de mesas -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          ${visibleTables.map((t) => {
            const meta = STATUS_META[t.status] || STATUS_META.LIBRE;
            const isMergedChild = t.merged_into_id !== null && t.merged_into_id !== undefined;
            const isSelectedForMerge = mergeSelection.some((m) => m.id === t.id);
            const selectableMerge = mergeMode && t.status === "LIBRE" && !isMergedChild;

            return `
              <div data-table-id="${t.id}" class="table-card bg-surface-container-lowest rounded-2xl border-2 p-4 shadow-sm flex flex-col justify-between gap-4 transition ${
                isSelectedForMerge
                  ? "border-bw-orange ring-2 ring-bw-orange/30"
                  : `border-surface-container-high ${selectableMerge ? "cursor-pointer hover:border-bw-orange" : meta.ring}`
              }">
                <div class="flex items-start justify-between gap-2">
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-display text-2xl text-bw-ink">${t.code}</span>
                      ${isMergedChild ? `<span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-surface-container text-secondary uppercase">Unida</span>` : ""}
                      ${t.effective_capacity > t.capacity ? `<span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-orange-100 text-primary uppercase">Combinada</span>` : ""}
                    </div>
                    <div class="text-[11px] font-semibold text-secondary uppercase mt-0.5">${(t.zone || "").replaceAll("_", " ")}</div>
                  </div>
                  <span class="px-2.5 py-1 rounded-full text-[11px] font-bold border ${meta.badge}">
                    ${meta.label}
                  </span>
                </div>

                <div class="grid grid-cols-2 gap-2 text-xs bg-surface-container-low rounded-xl p-2.5">
                  <div>
                    <div class="text-[10px] uppercase text-secondary font-bold">Capacidad</div>
                    <div class="font-bold text-on-surface">${t.effective_capacity} pers.</div>
                  </div>
                  <div>
                    <div class="text-[10px] uppercase text-secondary font-bold">Personas</div>
                    <div class="font-bold text-on-surface">${t.current_guests || 0}</div>
                  </div>
                </div>

                <div class="flex items-center justify-between gap-2 pt-1">
                  <span class="text-[10px] text-secondary font-mono">v${t.version} · #${t.id}</span>
                  <div class="flex gap-2">
                    ${!mergeMode && t.status === "LIBRE" && !isMergedChild && canAssignOrMerge ? `
                      <button data-assign="${t.id}" class="btn-assign px-3 py-1.5 rounded-lg bg-bw-orange hover:bg-bw-orange-deep text-white text-xs font-bold uppercase">
                        Sentar
                      </button>
                    ` : ""}
                    ${!mergeMode && t.status === "OCUPADA" && !isMergedChild ? `
                      <button data-order="${t.id}" class="btn-open-order px-3 py-1.5 rounded-lg bg-bw-ink hover:opacity-90 text-white text-xs font-bold uppercase">
                        Ver Comanda
                      </button>
                    ` : ""}
                    ${!mergeMode && t.status === "POR_LIMPIAR" && !isMergedChild && canClean ? `
                      <button data-clean="${t.id}" class="btn-clean px-3 py-1.5 rounded-lg bg-bw-forest hover:opacity-90 text-white text-xs font-bold uppercase">
                        Marcar Limpia
                      </button>
                    ` : ""}
                    ${selectableMerge ? `
                      <span class="text-xs font-bold ${isSelectedForMerge ? "text-bw-orange" : "text-secondary"}">
                        ${isSelectedForMerge ? "✓ Seleccionada" : "Tocar para unir"}
                      </span>
                    ` : ""}
                  </div>
                </div>
              </div>
            `;
          }).join("")}
        </div>
      </div>

      ${assignModalTable ? `
        <div id="assign-modal" class="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high shadow-xl w-full max-w-sm p-6 space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="font-display text-xl uppercase text-bw-ink">Sentar en ${assignModalTable.code}</h3>
              <button id="cancel-assign" class="text-secondary hover:text-on-surface">✕</button>
            </div>
            <p class="text-xs text-secondary">
              Zona: <strong>${assignModalTable.zone.replaceAll("_", " ")}</strong> · Capacidad máxima: <strong>${assignModalTable.effective_capacity} personas</strong>.
            </p>
            <div>
              <label class="block text-xs font-bold uppercase text-secondary mb-1.5">Número de personas</label>
              <input id="assign-guests" type="number" min="1" max="${assignModalTable.effective_capacity}" value="2"
                class="w-full px-3.5 py-2.5 rounded-xl bg-surface-container-low border border-surface-container-high font-bold text-lg focus:outline-none focus:ring-2 focus:ring-bw-orange" />
            </div>
            <div class="flex gap-2 pt-2">
              <button id="cancel-assign-2" class="w-1/2 py-2.5 rounded-xl bg-surface-container text-secondary font-bold text-xs uppercase">Cancelar</button>
              <button id="confirm-assign" class="w-1/2 py-2.5 rounded-xl bg-bw-orange hover:bg-bw-orange-deep text-white font-bold text-xs uppercase">Confirmar</button>
            </div>
          </div>
        </div>
      ` : ""}
    `;

    // Eventos
    container.querySelector("#close-feedback")?.addEventListener("click", () => {
      feedback = null;
      render();
    });

    container.querySelector("#btn-refresh-tables")?.addEventListener("click", () => loadTables(false));

    container.querySelectorAll(".zone-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        selectedZone = btn.dataset.zone;
        render();
      });
    });

    container.querySelector("#btn-toggle-merge")?.addEventListener("click", () => {
      mergeMode = !mergeMode;
      mergeSelection = [];
      render();
    });

    if (mergeMode) {
      container.querySelectorAll(".table-card").forEach((card) => {
        const id = Number(card.dataset.tableId);
        const t = tables.find((x) => x.id === id);
        if (!t || t.status !== "LIBRE" || t.merged_into_id) return;
        card.addEventListener("click", () => {
          const exists = mergeSelection.some((m) => m.id === id);
          if (exists) mergeSelection = mergeSelection.filter((m) => m.id !== id);
          else mergeSelection.push({ id: t.id, version: t.version, code: t.code });
          render();
        });
      });
    }

    container.querySelector("#btn-confirm-merge")?.addEventListener("click", async () => {
      try {
        const payload = mergeSelection.map(({ id, version }) => ({ id, version }));
        const mainTable = await api.mergeTables(payload);
        mergeMode = false;
       setFeedback("ok", `Mesas combinadas en ${mainTable.code} (nueva capacidad: ${mainTable.effective_capacity} personas).`);
          await loadTables(true);
        } catch (err) {
          setFeedback("err", ERROR_MESSAGES[err.code] || err.message);
          if (err.code === "version_conflict") await loadTables(true);
          else render();
        }
      });
      container.querySelectorAll(".btn-assign").forEach((btn) => {
      btn.addEventListener("click", () => {
        assignModalTable = tables.find((t) => t.id === Number(btn.dataset.assign));
        render();
      });
    });

    container.querySelectorAll(".btn-clean").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const t = tables.find((x) => x.id === Number(btn.dataset.clean));
        if (!t) return;
        try {
          await api.cleanTable(t.id, t.version);
          setFeedback("ok", `Mesa ${t.code} marcada como LIBRE.`);
          await loadTables(true);
        } catch (err) {
          setFeedback("err", ERROR_MESSAGES[err.code] || err.message);
          if (err.code === "version_conflict") await loadTables(true);
          else render();
        }
      });
    });

    container.querySelectorAll(".btn-open-order").forEach((btn) => {
      btn.addEventListener("click", () => {
        const t = tables.find((x) => x.id === Number(btn.dataset.order));
        if (t && onOpenOrder) onOpenOrder(t);
      });
    });

    if (assignModalTable) {
      const closeModal = () => {
        assignModalTable = null;
        render();
      };
      container.querySelector("#cancel-assign")?.addEventListener("click", closeModal);
      container.querySelector("#cancel-assign-2")?.addEventListener("click", closeModal);
      container.querySelector("#confirm-assign")?.addEventListener("click", async () => {
        const guests = parseInt(container.querySelector("#assign-guests").value, 10);
        const target = assignModalTable;
        assignModalTable = null;
        try {
          const updated = await api.assignTable(target.id, guests, target.version);
        setFeedback("ok", `Mesa ${updated.code} ocupada con ${updated.current_guests} ${updated.current_guests === 1 ? "persona" : "personas"}.`);
          await loadTables(true);
        } catch (err) {
          setFeedback("err", ERROR_MESSAGES[err.code] || err.message);
          if (err.code === "version_conflict") await loadTables(true);
          else render();
        }
      });
    }
  }

  await loadTables(false);
}
