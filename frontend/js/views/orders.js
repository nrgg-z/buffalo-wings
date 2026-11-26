import { api } from "../api.js";

const money = (n) => `$${Number(n || 0).toFixed(2)}`;

const ITEM_STATUS = {
  PENDING: { label: "Pendiente", cls: "bg-amber-50 text-amber-800 border-amber-200" },
  IN_PROGRESS: { label: "En preparación", cls: "bg-blue-50 text-blue-700 border-blue-200" },
  READY: { label: "Listo", cls: "bg-emerald-50 text-bw-forest border-emerald-200" },
  DELIVERED: { label: "Entregado", cls: "bg-surface-container text-secondary border-surface-container-high" },
  VOIDED: { label: "Anulado", cls: "bg-red-50 text-error border-red-200 line-through" },
};

export async function renderOrders(container, { user, initialTable, onSelectTable, onGoToBilling }) {
  let occupiedTables = [];
  let allTables = [];
  let selectedTable = initialTable || null;
  let categories = [];
  let activeCatId = null;
  let currentOrder = null;
  let draftItems = []; // [{ product, quantity, notes, modifiers: [ModifierOption] }]
  let modalProduct = null;
  let voidTargetItem = null;
  let feedback = null;
  let sendingOrder = false;

  const canTakeOrder = ["WAITER", "MANAGER", "ADMIN"].includes(user.role);

  async function init() {
    container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Cargando comandas y menú...</div>`;
    try {
      const [tablesRes, catRes] = await Promise.all([api.getTables(), api.getCatalog()]);
      allTables = tablesRes.tables || [];
      occupiedTables = allTables.filter((t) => t.status === "OCUPADA" && !t.merged_into_id);
      categories = catRes.categories || [];
      if (categories.length && !activeCatId) activeCatId = categories[0].id;

      if (selectedTable) {
        selectedTable = occupiedTables.find((t) => t.id === selectedTable.id) || occupiedTables[0] || null;
      } else {
        selectedTable = occupiedTables[0] || null;
      }

      if (selectedTable) {
        await loadTableOrder();
      }
      render();
    } catch (err) {
      container.innerHTML = `<div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-medium">${err.message}</div>`;
    }
  }

  function groupDetails(table) {
    const children = allTables.filter((t) => t.merged_into_id === table.id);
    const guestCount = table.current_guests + children.reduce((sum, t) => sum + t.current_guests, 0);
    return {
      label: children.length ? `${table.code} (+ ${children.map((t) => t.code).join(", ")})` : table.code,
      guests: guestCount,
    };
  }

  async function loadTableOrder() {
    if (!selectedTable) {
      currentOrder = null;
      return;
    }
    try {
      currentOrder = await api.getTableOrder(selectedTable.id);
    } catch (err) {
      if (err.status === 404) {
        currentOrder = null; // Mesa ocupada aún sin comanda abierta
      } else {
        feedback = { type: "err", text: err.message };
        currentOrder = null;
      }
    }
  }

  function draftSubtotal() {
    return draftItems.reduce((acc, it) => {
      const extras = it.modifiers.reduce((s, m) => s + Number(m.extra_price || 0), 0);
      return acc + (Number(it.product.base_price) + extras) * it.quantity;
    }, 0);
  }

  function render() {
    if (!occupiedTables.length) {
      container.innerHTML = `
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-8 text-center space-y-2 shadow-sm">
          <h2 class="font-display text-2xl uppercase text-bw-ink">No hay mesas ocupadas</h2>
          <p class="text-sm text-secondary">Ve a la pestaña <strong>MESAS</strong> y sienta comensales en una mesa libre para abrir una comanda.</p>
        </div>
      `;
      return;
    }

    const activeCat = categories.find((c) => c.id === activeCatId) || categories[0];
    const selectedGroup = selectedTable ? groupDetails(selectedTable) : null;

    container.innerHTML = `
      <div class="space-y-4">
        <!-- Barra superior de selección de mesa -->
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-4 shadow-sm flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div>
              <div class="text-[10px] font-bold uppercase tracking-widest text-secondary">Toma de Comandas</div>
              <h1 class="font-display text-2xl uppercase text-bw-ink leading-none mt-0.5">
                Mesa ${selectedGroup?.label || selectedTable.code} ${selectedGroup ? `· ${selectedGroup.guests} pers.` : ""} ${currentOrder ? `· Comanda #${currentOrder.id}` : "· Sin comanda"}
              </h1>
            </div>
            <select id="select-order-table" class="px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-bold">
              ${occupiedTables.map((t) => `
                <option value="${t.id}" ${t.id === selectedTable.id ? "selected" : ""}>
                  ${groupDetails(t).label} (${groupDetails(t).guests} pers. · ${t.zone.replaceAll("_", " ")})
                </option>
              `).join("")}
            </select>
          </div>

          <div class="flex items-center gap-2">
            ${currentOrder ? `
              <button id="btn-go-billing" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-bw-ink text-white hover:opacity-90">
                Ir a Caja (${money(currentOrder.subtotal)})
              </button>
            ` : ""}
            <button id="btn-reload-order" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low">
              Actualizar
            </button>
          </div>
        </div>

        ${feedback ? `
          <div class="p-3.5 rounded-xl border text-xs font-semibold flex items-center justify-between ${
            feedback.type === "ok" ? "bg-emerald-50 border-emerald-200 text-bw-forest" : "bg-red-50 border-red-200 text-error"
          }">
            <span>${feedback.text}</span>
            <button id="close-order-fb" class="ml-4 opacity-70 hover:opacity-100">✕</button>
          </div>
        ` : ""}

        <div class="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
          <!-- IZQUIERDA: MENÚ -->
          <div class="lg:col-span-7 space-y-4">
            <div class="flex flex-wrap gap-1.5">
              ${categories.map((c) => `
                <button data-cat="${c.id}" class="order-cat-tab px-3.5 py-2 rounded-xl text-xs font-bold uppercase transition ${
                  activeCat && c.id === activeCat.id
                    ? "bg-bw-orange text-white"
                    : "bg-surface-container-lowest border border-surface-container-high text-secondary hover:text-on-surface"
                }">${c.name}</button>
              `).join("")}
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              ${(activeCat?.products || []).map((p) => `
                <button data-add-prod="${p.id}" ${!p.is_available || !canTakeOrder ? "disabled" : ""}
                  class="btn-pick-prod text-left bg-surface-container-lowest rounded-2xl border p-4 shadow-sm transition flex flex-col justify-between gap-2 ${
                    p.is_available && canTakeOrder
                      ? "border-surface-container-high hover:border-bw-orange cursor-pointer"
                      : "border-red-200 opacity-50 cursor-not-allowed"
                  }">
                  <div class="flex items-start justify-between gap-2 w-full">
                    <span class="font-bold text-sm text-on-surface">${p.name}</span>
                    <span class="font-display text-lg text-bw-orange">${money(p.base_price)}</span>
                  </div>
                  <div class="flex items-center justify-between w-full text-[11px]">
                    <span class="text-secondary">${(p.modifier_groups || []).length ? "Personalizable (salsas/extras)" : "Directo"}</span>
                    ${!p.is_available ? `<span class="text-error font-bold uppercase">Agotado</span>` : `<span class="text-bw-forest font-bold">+ Agregar</span>`}
                  </div>
                </button>
              `).join("")}
            </div>
          </div>

          <!-- DERECHA: BORRADOR + COMANDA ACTIVA -->
          <div class="lg:col-span-5 space-y-4">
            <!-- Carrito por enviar -->
            <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-4 shadow-sm space-y-3">
              <div class="flex items-center justify-between">
                <h2 class="font-display text-lg uppercase text-bw-ink">Por enviar a cocina (${draftItems.length})</h2>
                ${draftItems.length ? `<button id="btn-clear-draft" class="text-[11px] font-bold text-error uppercase">Vaciar</button>` : ""}
              </div>

              ${draftItems.length ? `
                <div class="space-y-2 max-h-60 overflow-y-auto pr-1">
                  ${draftItems.map((it, idx) => {
                    const extras = it.modifiers.reduce((s, m) => s + Number(m.extra_price || 0), 0);
                    const lineTotal = (Number(it.product.base_price) + extras) * it.quantity;
                    return `
                      <div class="p-2.5 rounded-xl bg-surface-container-low flex items-start justify-between gap-2 text-xs">
                        <div>
                          <div class="font-bold">${it.quantity}x ${it.product.name}</div>
                          ${it.modifiers.length ? `<div class="text-[11px] text-secondary">${it.modifiers.map((m) => m.name).join(", ")}</div>` : ""}
                          ${it.notes ? `<div class="text-[11px] italic text-bw-orange">Nota: ${it.notes}</div>` : ""}
                        </div>
                        <div class="flex items-center gap-2">
                          <span class="font-bold">${money(lineTotal)}</span>
                          <button data-rm-draft="${idx}" class="btn-rm-draft text-error font-bold px-1">✕</button>
                        </div>
                      </div>
                    `;
                  }).join("")}
                </div>
                <div class="pt-2 border-t border-surface-container-high flex items-center justify-between text-sm font-bold">
                  <span>Subtotal nuevo</span>
                  <span>${money(draftSubtotal())}</span>
                </div>
                <button id="btn-send-order" ${sendingOrder ? "disabled" : ""} class="w-full py-3 rounded-xl bg-bw-orange hover:bg-bw-orange-deep text-white font-display text-base uppercase tracking-wider shadow disabled:opacity-50">
                  ${sendingOrder ? "Enviando..." : "Enviar a Cocina / Barra"}
                </button>
              ` : `
                <p class="text-xs text-secondary">Selecciona productos del menú para agregarlos a la comanda de <strong>${selectedTable.code}</strong>.</p>
              `}
            </div>

            <!-- Comanda ya enviada -->
            <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-4 shadow-sm space-y-3">
              <div class="flex items-center justify-between">
                <h2 class="font-display text-lg uppercase text-bw-ink">Comanda activa</h2>
                ${currentOrder ? `<span class="text-xs font-bold text-bw-forest">Subtotal: ${money(currentOrder.subtotal)}</span>` : ""}
              </div>

              ${currentOrder && (currentOrder.items || []).length ? `
                <div class="space-y-2 max-h-80 overflow-y-auto pr-1">
                  ${currentOrder.items.map((it) => {
                    const st = ITEM_STATUS[it.status] || ITEM_STATUS.PENDING;
                    return `
                      <div class="p-3 rounded-xl border border-surface-container-high bg-surface-container-low/50 space-y-1.5 text-xs">
                        <div class="flex items-start justify-between gap-2">
                          <div>
                            <div class="font-bold ${it.status === "VOIDED" ? "line-through text-secondary" : ""}">
                              ${it.quantity}x ${it.product_name}
                            </div>
                            ${(it.modifiers || []).length ? `<div class="text-[11px] text-secondary">${it.modifiers.map((m) => m.name).join(", ")}</div>` : ""}
                            ${it.notes ? `<div class="text-[11px] italic text-bw-orange">Nota: ${it.notes}</div>` : ""}
                            ${it.void_reason ? `<div class="text-[11px] text-error">Anulado: ${it.void_reason}</div>` : ""}
                          </div>
                          <div class="text-right space-y-1">
                            <span class="inline-block px-2 py-0.5 rounded-full text-[10px] font-bold border ${st.cls}">${st.label}</span>
                            <div class="font-bold">${money(it.unit_price * it.quantity)}</div>
                          </div>
                        </div>
                        <div class="flex items-center justify-between pt-1">
                          <span class="text-[10px] uppercase text-secondary font-bold">${it.station}</span>
                          ${canTakeOrder && it.status !== "VOIDED" ? `
                            <button data-void-item="${it.id}" class="btn-void-item text-[11px] font-bold text-error hover:underline uppercase">
                              Anular ítem
                            </button>
                          ` : ""}
                        </div>
                      </div>
                    `;
                  }).join("")}
                </div>
              ` : `
                <p class="text-xs text-secondary">Todavía no hay productos enviados para esta mesa.</p>
              `}
            </div>
          </div>
        </div>
      </div>

      <!-- Modal de modificadores / producto -->
      ${modalProduct ? `
        <div class="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high shadow-xl w-full max-w-md p-5 space-y-4 max-h-[90vh] overflow-y-auto">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="font-display text-xl uppercase text-bw-ink">${modalProduct.name}</h3>
                <div class="text-xs font-bold text-bw-orange">${money(modalProduct.base_price)}</div>
              </div>
              <button id="close-prod-modal" class="text-secondary hover:text-on-surface">✕</button>
            </div>

            <div class="flex items-center justify-between bg-surface-container-low rounded-xl p-3">
              <span class="text-xs font-bold uppercase text-secondary">Cantidad</span>
              <input id="modal-qty" type="number" min="1" value="1" class="w-20 text-center px-2 py-1 rounded-lg bg-white border border-surface-container-high font-bold" />
            </div>

            ${(modalProduct.modifier_groups || []).map((g) => `
              <div class="space-y-2 border-t border-surface-container-high pt-3">
                <div class="flex items-center justify-between">
                  <span class="text-xs font-bold uppercase text-bw-ink">${g.name}</span>
                  <span class="text-[10px] font-bold uppercase ${g.is_required ? "text-bw-orange" : "text-secondary"}">
                    ${g.is_required ? "Obligatorio" : "Opcional"} · ${g.min_selections} a ${g.max_selections}
                  </span>
                </div>
                <div class="grid grid-cols-1 gap-1.5">
                  ${(g.options || []).map((o) => `
                    <label class="flex items-center justify-between p-2 rounded-xl border text-xs ${
                      o.is_available ? "border-surface-container-high bg-surface-container-low/50 cursor-pointer" : "border-red-200 opacity-50 cursor-not-allowed"
                    }">
                      <span class="flex items-center gap-2 font-medium">
                        <input type="${g.max_selections === 1 ? "radio" : "checkbox"}" name="mod_group_${g.id}" value="${o.id}" ${!o.is_available ? "disabled" : ""} />
                        ${o.name} ${!o.is_available ? "(Agotado)" : ""}
                      </span>
                      <span class="font-bold text-secondary">${o.extra_price > 0 ? `+${money(o.extra_price)}` : "Incluido"}</span>
                    </label>
                  `).join("")}
                </div>
              </div>
            `).join("")}

            <div>
              <label class="block text-xs font-bold uppercase text-secondary mb-1">Notas de cocina (opcional)</label>
              <input id="modal-notes" type="text" placeholder="Ej: salsa aparte, sin cebolla..."
                class="w-full px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs" />
            </div>

            <div id="modal-prod-err" class="hidden text-xs font-semibold text-error"></div>

            <div class="flex gap-2 pt-1">
              <button id="cancel-prod-modal" class="w-1/2 py-2.5 rounded-xl bg-surface-container text-secondary font-bold text-xs uppercase">Cancelar</button>
              <button id="confirm-prod-modal" class="w-1/2 py-2.5 rounded-xl bg-bw-orange hover:bg-bw-orange-deep text-white font-bold text-xs uppercase">Agregar</button>
            </div>
          </div>
        </div>
      ` : ""}

      <!-- Modal de anulación con PIN de gerente -->
      ${voidTargetItem ? `
        <div class="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high shadow-xl w-full max-w-sm p-5 space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="font-display text-xl uppercase text-error">Anular ítem</h3>
              <button id="close-void-modal" class="text-secondary">✕</button>
            </div>
            <p class="text-xs text-secondary">Requiere autorización de gerente para anular <strong>${voidTargetItem.quantity}x ${voidTargetItem.product_name}</strong>.</p>
            <div>
              <label class="block text-[11px] font-bold uppercase text-secondary mb-1">Código de Gerente</label>
              <input id="void-mgr-code" type="text" value="EMP-01" class="w-full px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-bold" />
            </div>
            <div>
              <label class="block text-[11px] font-bold uppercase text-secondary mb-1">PIN de Gerente</label>
              <input id="void-mgr-pin" type="password" maxlength="4" placeholder="1111" class="w-full px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-bold" />
            </div>
            <div>
              <label class="block text-[11px] font-bold uppercase text-secondary mb-1">Motivo de anulación</label>
              <input id="void-reason" type="text" placeholder="Ej: cliente cambió de opinión" class="w-full px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs" />
            </div>
            <div class="flex gap-2 pt-2">
              <button id="cancel-void-modal" class="w-1/2 py-2.5 rounded-xl bg-surface-container text-secondary font-bold text-xs uppercase">Cancelar</button>
              <button id="confirm-void-modal" class="w-1/2 py-2.5 rounded-xl bg-error text-white font-bold text-xs uppercase">Confirmar Anulación</button>
            </div>
          </div>
        </div>
      ` : ""}
    `;

    // Eventos
    container.querySelector("#close-order-fb")?.addEventListener("click", () => {
      feedback = null;
      render();
    });

    container.querySelector("#select-order-table")?.addEventListener("change", async (e) => {
      selectedTable = occupiedTables.find((t) => t.id === Number(e.target.value)) || null;
      draftItems = [];
      feedback = null;
      if (onSelectTable) onSelectTable(selectedTable);
      await loadTableOrder();
      render();
    });

    container.querySelector("#btn-reload-order")?.addEventListener("click", async () => {
      await loadTableOrder();
      render();
    });

    container.querySelector("#btn-go-billing")?.addEventListener("click", () => {
      if (onGoToBilling && currentOrder) onGoToBilling(selectedTable, currentOrder);
    });

    container.querySelectorAll(".order-cat-tab").forEach((btn) => {
      btn.addEventListener("click", () => {
        activeCatId = Number(btn.dataset.cat);
        render();
      });
    });

    container.querySelectorAll(".btn-pick-prod").forEach((btn) => {
      btn.addEventListener("click", () => {
        const prodId = Number(btn.dataset.addProd);
        const prod = (activeCat?.products || []).find((p) => p.id === prodId);
        if (!prod) return;
        modalProduct = prod;
        render();
      });
    });

    container.querySelector("#btn-clear-draft")?.addEventListener("click", () => {
      draftItems = [];
      render();
    });

    container.querySelectorAll(".btn-rm-draft").forEach((btn) => {
      btn.addEventListener("click", () => {
        draftItems.splice(Number(btn.dataset.rmDraft), 1);
        render();
      });
    });

    container.querySelector("#btn-send-order")?.addEventListener("click", async () => {
      if (sendingOrder || !draftItems.length || !selectedTable) return;
      sendingOrder = true;
      const sendButton = container.querySelector("#btn-send-order");
      if (sendButton) {
        sendButton.disabled = true;
        sendButton.textContent = "Enviando...";
      }
      const payloadItems = draftItems.map((it) => ({
        product_id: it.product.id,
        quantity: it.quantity,
        notes: it.notes,
        modifier_option_ids: it.modifiers.map((m) => m.id),
      }));
      try {
        currentOrder = await api.createOrAppendOrder(selectedTable.id, payloadItems);
        sendingOrder = false;
        draftItems = [];
        feedback = { type: "ok", text: `Comanda #${currentOrder.id} enviada a Cocina/Barra.` };
        render();
      } catch (err) {
        sendingOrder = false;
        feedback = { type: "err", text: err.message };
        render();
      }
    });

    container.querySelectorAll(".btn-void-item").forEach((btn) => {
      btn.addEventListener("click", () => {
        const id = Number(btn.dataset.voidItem);
        voidTargetItem = (currentOrder?.items || []).find((i) => i.id === id) || null;
        render();
      });
    });

    if (modalProduct) {
      const closeProd = () => {
        modalProduct = null;
        render();
      };
      container.querySelector("#close-prod-modal")?.addEventListener("click", closeProd);
      container.querySelector("#cancel-prod-modal")?.addEventListener("click", closeProd);
      container.querySelector("#confirm-prod-modal")?.addEventListener("click", () => {
        const qty = Math.max(1, parseInt(container.querySelector("#modal-qty").value, 10) || 1);
        const notes = container.querySelector("#modal-notes").value.trim();
        const errEl = container.querySelector("#modal-prod-err");
        const chosenMods = [];

        for (const g of modalProduct.modifier_groups || []) {
          const checked = [...container.querySelectorAll(`input[name="mod_group_${g.id}"]:checked`)].map((i) => Number(i.value));
          if (g.is_required && checked.length < g.min_selections) {
            errEl.textContent = `Selecciona al menos ${g.min_selections} opción(es) en "${g.name}".`;
            errEl.classList.remove("hidden");
            return;
          }
          if (g.max_selections > 0 && checked.length > g.max_selections) {
            errEl.textContent = `Máximo ${g.max_selections} opción(es) en "${g.name}".`;
            errEl.classList.remove("hidden");
            return;
          }
          for (const optId of checked) {
            const opt = (g.options || []).find((o) => o.id === optId);
            if (opt) chosenMods.push(opt);
          }
        }

        draftItems.push({ product: modalProduct, quantity: qty, notes, modifiers: chosenMods });
        modalProduct = null;
        render();
      });
    }

    if (voidTargetItem) {
      const closeVoid = () => {
        voidTargetItem = null;
        render();
      };
      container.querySelector("#close-void-modal")?.addEventListener("click", closeVoid);
      container.querySelector("#cancel-void-modal")?.addEventListener("click", closeVoid);
      container.querySelector("#confirm-void-modal")?.addEventListener("click", async () => {
        const manager_code = container.querySelector("#void-mgr-code").value.trim();
        const manager_pin = container.querySelector("#void-mgr-pin").value.trim();
        const reason = container.querySelector("#void-reason").value.trim();
        const target = voidTargetItem;
        voidTargetItem = null;
        try {
          currentOrder = await api.voidOrderItem(currentOrder.id, target.id, { manager_code, manager_pin, reason });
          feedback = { type: "ok", text: `Ítem "${target.product_name}" anulado correctamente.` };
          render();
        } catch (err) {
          feedback = { type: "err", text: err.message };
          render();
        }
      });
    }
  }

  await init();
}
