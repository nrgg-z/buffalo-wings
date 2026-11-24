import { api } from "../api.js";

const money = (n) => `$${Number(n || 0).toFixed(2)}`;
const round2 = (n) => Math.round((Number(n) || 0) * 100) / 100;

const METHOD_LABELS = {
  CASH: "Efectivo",
  CARD: "Tarjeta",
  QR: "Pago QR",
};

export async function renderBilling(container, { user, initialTable, initialOrder, onGoToTables }) {
  const canPay = ["CASHIER", "MANAGER", "ADMIN"].includes(user.role);

  let openAccounts = []; // [{ table, order }]
  let selectedOrderId = initialOrder?.id || null;
  let precheck = null;
  let activeOrder = null;
  let receipt = null;
  let feedback = null;

  // Estado del formulario de pago
  let paymentMethod = "CASH";
  let tipMode = "SUGGESTED"; // "NONE" | "SUGGESTED" | "CUSTOM"
  let customTip = 0;
  let amountReceived = "";
  let reference = "";

  async function init() {
    container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Cargando cuentas abiertas...</div>`;
    try {
      const tablesRes = await api.getTables();
      const occupied = (tablesRes.tables || []).filter((t) => t.status === "OCUPADA" && !t.merged_into_id);

      const results = await Promise.all(
        occupied.map(async (table) => {
          try {
            const order = await api.getTableOrder(table.id);
            return { table, order };
          } catch {
            return null;
          }
        })
      );

      openAccounts = results.filter((x) => x && x.order && x.order.status === "OPEN");

      if (!selectedOrderId && initialTable) {
        const match = openAccounts.find((a) => a.table.id === initialTable.id);
        if (match) selectedOrderId = match.order.id;
      }

      if (!openAccounts.some((a) => a.order.id === selectedOrderId)) {
        selectedOrderId = openAccounts[0]?.order.id || null;
      }

      if (selectedOrderId) {
        await loadPrecheck(selectedOrderId);
      } else {
        precheck = null;
        activeOrder = null;
      }
      render();
    } catch (err) {
      container.innerHTML = `<div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-medium">${err.message}</div>`;
    }
  }

  async function loadPrecheck(orderId) {
    const acc = openAccounts.find((a) => a.order.id === orderId);
    activeOrder = acc?.order || null;
    try {
      precheck = await api.getPrecheck(orderId);
      const methods = precheck.enabled_payment_methods || ["CASH", "CARD", "QR"];
      if (!methods.includes(paymentMethod)) paymentMethod = methods[0] || "CASH";
      tipMode = "SUGGESTED";
      customTip = precheck.suggested_tip_amount || 0;
      amountReceived = "";
      reference = "";
    } catch (err) {
      precheck = null;
      feedback = { type: "err", text: err.message };
    }
  }

  function currentTip() {
    if (!precheck) return 0;
    if (tipMode === "NONE") return 0;
    if (tipMode === "SUGGESTED") return round2(precheck.suggested_tip_amount);
    return Math.max(0, round2(customTip));
  }

  function currentTotal() {
    if (!precheck) return 0;
    return round2(precheck.total_without_tip + currentTip());
  }

  function render(focusSelector = null) {
    const total = currentTotal();
    const tip = currentTip();
    const receivedNum = paymentMethod === "CASH" ? Number(amountReceived || 0) : total;
    const change = paymentMethod === "CASH" ? round2(receivedNum - total) : 0;

    container.innerHTML = `
      <div class="space-y-5">
        <!-- Encabezado -->
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-widest text-secondary">Facturación y Emisión DTE</div>
            <h1 class="font-display text-2xl uppercase text-bw-ink tracking-wide mt-0.5">Caja POS</h1>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-1.5">
              <input id="input-lookup-receipt" type="number" min="1" placeholder="Comanda #"
                class="w-28 px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-bold" />
              <button id="btn-lookup-receipt" class="px-3 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container text-on-surface hover:bg-surface-container-high">
                Ver Recibo
              </button>
            </div>
            <button id="btn-reload-billing" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">refresh</span> Actualizar
            </button>
          </div>
        </div>

        ${feedback ? `
          <div class="p-3.5 rounded-xl border text-xs font-semibold flex items-center justify-between ${
            feedback.type === "ok" ? "bg-emerald-50 border-emerald-200 text-bw-forest" : "bg-red-50 border-red-200 text-error"
          }">
            <span>${feedback.text}</span>
            <button id="close-bill-fb" class="ml-4 opacity-70 hover:opacity-100">✕</button>
          </div>
        ` : ""}

        <!-- Recibo DTE emitido o consultado -->
        ${receipt ? `
          <div class="bg-emerald-50/70 border-2 border-emerald-300 rounded-2xl p-6 shadow-sm space-y-4">
            <div class="flex flex-wrap items-start justify-between gap-3 border-b border-emerald-200 pb-4">
              <div>
                <div class="text-[11px] font-bold uppercase tracking-widest text-bw-forest">Pago Registrado · Comprobante DTE</div>
                <h2 class="font-display text-2xl uppercase text-bw-ink mt-0.5">Mesa ${receipt.table_code} · Comanda #${receipt.order_id}</h2>
                <div class="text-xs font-mono font-bold text-bw-forest mt-1">${receipt.dte_code}</div>
              </div>
              <div class="flex gap-2">
                <button id="btn-print-receipt" class="px-3.5 py-2 rounded-xl bg-white border border-emerald-300 text-xs font-bold uppercase text-bw-ink hover:bg-emerald-50">
                  Imprimir
                </button>
                <button id="btn-go-tables" class="px-3.5 py-2 rounded-xl bg-bw-ink text-white text-xs font-bold uppercase hover:opacity-90">
                  Ir a Mesas
                </button>
                <button id="btn-close-receipt" class="px-3 py-2 rounded-xl text-xs font-bold text-secondary hover:text-on-surface">
                  ✕
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div class="bg-white rounded-xl p-3 border border-emerald-200">
                <div class="text-secondary uppercase font-bold text-[10px]">Método</div>
                <div class="font-bold text-sm mt-0.5">${METHOD_LABELS[receipt.payment_method] || receipt.payment_method}</div>
                ${receipt.reference ? `<div class="text-[11px] text-secondary">Ref: ${receipt.reference}</div>` : ""}
              </div>
              <div class="bg-white rounded-xl p-3 border border-emerald-200">
                <div class="text-secondary uppercase font-bold text-[10px]">Subtotal + IVA</div>
                <div class="font-bold text-sm mt-0.5">${money(receipt.subtotal)} + ${money(receipt.tax_amount)}</div>
                <div class="text-[11px] text-secondary">Propina: ${money(receipt.tip_amount)}</div>
              </div>
              <div class="bg-white rounded-xl p-3 border border-emerald-200">
                <div class="text-secondary uppercase font-bold text-[10px]">Total Pagado</div>
                <div class="font-display text-xl text-bw-forest mt-0.5">${money(receipt.total_amount)}</div>
              </div>
              <div class="bg-white rounded-xl p-3 border border-emerald-200">
                <div class="text-secondary uppercase font-bold text-[10px]">Recibido / Vuelto</div>
                <div class="font-bold text-sm mt-0.5">${money(receipt.amount_received)}</div>
                <div class="text-[11px] font-bold text-bw-orange">Vuelto: ${money(receipt.change_amount)}</div>
              </div>
            </div>
          </div>
        ` : ""}

        ${!openAccounts.length ? `
          <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-8 text-center space-y-2 shadow-sm">
            <h2 class="font-display text-2xl uppercase text-bw-ink">No hay cuentas pendientes de cobro</h2>
            <p class="text-sm text-secondary">Todas las mesas ocupadas están sin comanda o ya fueron cobradas.</p>
          </div>
        ` : `
          <div class="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
            <!-- IZQUIERDA: Cuentas abiertas + Precuenta -->
            <div class="lg:col-span-6 space-y-4">
              <div class="flex flex-wrap gap-2">
                ${openAccounts.map(({ table, order }) => `
                  <button data-pick-order="${order.id}" class="btn-pick-account px-3.5 py-2 rounded-xl text-xs font-bold uppercase border transition ${
                    order.id === selectedOrderId
                      ? "bg-bw-orange text-white border-bw-orange"
                      : "bg-surface-container-lowest border-surface-container-high text-on-surface hover:border-bw-orange"
                  }">
                    Mesa ${table.code} · #${order.id} (${money(order.subtotal)})
                  </button>
                `).join("")}
              </div>

              ${precheck && activeOrder ? `
                <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm space-y-4">
                  <div class="flex items-center justify-between border-b border-surface-container-high pb-3">
                    <div>
                      <div class="text-[10px] font-bold uppercase tracking-widest text-secondary">Precuenta Oficial</div>
                      <h2 class="font-display text-2xl uppercase text-bw-ink">Mesa ${precheck.table_code} · Comanda #${precheck.order_id}</h2>
                    </div>
                    <span class="px-2.5 py-1 rounded-full text-[11px] font-bold uppercase bg-amber-50 text-amber-800 border border-amber-200">
                      Por cobrar
                    </span>
                  </div>

                  <div class="space-y-2 max-h-64 overflow-y-auto pr-1">
                    ${(activeOrder.items || []).map((it) => `
                      <div class="flex items-start justify-between gap-2 text-xs py-1.5 border-b border-surface-container-low ${
                        it.status === "VOIDED" ? "opacity-45 line-through" : ""
                      }">
                        <div>
                          <div class="font-bold">${it.quantity}× ${it.product_name}</div>
                          ${(it.modifiers || []).length ? `<div class="text-[11px] text-secondary">${it.modifiers.map((m) => m.name).join(", ")}</div>` : ""}
                        </div>
                        <div class="font-bold">${money(it.unit_price * it.quantity)}</div>
                      </div>
                    `).join("")}
                  </div>

                  <div class="space-y-1.5 pt-2 border-t border-surface-container-high text-sm">
                    <div class="flex justify-between text-secondary">
                      <span>Subtotal</span>
                      <span class="font-semibold text-on-surface">${money(precheck.subtotal)}</span>
                    </div>
                    <div class="flex justify-between text-secondary">
                      <span>IVA (${Math.round((precheck.tax_rate || 0.13) * 100)}%)</span>
                      <span class="font-semibold text-on-surface">${money(precheck.tax_amount)}</span>
                    </div>
                    <div class="flex justify-between text-secondary">
                      <span>Propina (${tipMode === "SUGGESTED" ? `${Math.round((precheck.suggested_tip_rate || 0.1) * 100)}%` : tipMode === "NONE" ? "0%" : "Personalizada"})</span>
                      <span class="font-semibold text-on-surface">${money(tip)}</span>
                    </div>
                    <div class="flex justify-between pt-2 border-t border-surface-container-high font-display text-2xl text-bw-ink">
                      <span>TOTAL A PAGAR</span>
                      <span class="text-bw-orange">${money(total)}</span>
                    </div>
                  </div>
                </div>
              ` : ""}
            </div>

            <!-- DERECHA: Panel de Cobro -->
            <div class="lg:col-span-6">
              ${precheck ? `
                <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm space-y-5">
                  <h2 class="font-display text-xl uppercase text-bw-ink">Procesar Cobro</h2>

                  ${!canPay ? `
                    <div class="p-4 rounded-xl bg-amber-50 border border-amber-200 text-amber-800 text-xs font-semibold">
                      Tu rol (${user.role}) solo puede consultar precuentas y recibos. El cobro requiere rol <strong>CASHIER</strong> o <strong>MANAGER</strong>.
                    </div>
                  ` : `
                    <!-- Propina -->
                    <div class="space-y-2">
                      <label class="block text-xs font-bold uppercase text-secondary">Propina</label>
                      <div class="grid grid-cols-3 gap-2">
                        <button data-tip="NONE" class="btn-tip-mode py-2 rounded-xl text-xs font-bold uppercase border ${
                          tipMode === "NONE" ? "bg-bw-ink text-white border-bw-ink" : "bg-surface-container-low border-surface-container-high text-secondary"
                        }">Sin propina ($0)</button>
                        <button data-tip="SUGGESTED" class="btn-tip-mode py-2 rounded-xl text-xs font-bold uppercase border ${
                          tipMode === "SUGGESTED" ? "bg-bw-ink text-white border-bw-ink" : "bg-surface-container-low border-surface-container-high text-secondary"
                        }">10% (${money(precheck.suggested_tip_amount)})</button>
                        <button data-tip="CUSTOM" class="btn-tip-mode py-2 rounded-xl text-xs font-bold uppercase border ${
                          tipMode === "CUSTOM" ? "bg-bw-ink text-white border-bw-ink" : "bg-surface-container-low border-surface-container-high text-secondary"
                        }">Otro monto</button>
                      </div>
                      ${tipMode === "CUSTOM" ? `
                        <input id="input-custom-tip" type="number" step="0.01" min="0" value="${customTip}"
                          class="w-full px-3 py-2 rounded-xl bg-surface-container-low border border-surface-container-high text-sm font-bold" />
                      ` : ""}
                    </div>

                    <!-- Método de pago -->
                    <div class="space-y-2">
                      <label class="block text-xs font-bold uppercase text-secondary">Método de Pago</label>
                      <div class="grid grid-cols-3 gap-2">
                        ${(precheck.enabled_payment_methods || ["CASH", "CARD", "QR"]).map((m) => `
                          <button data-method="${m}" class="btn-pay-method py-2.5 rounded-xl text-xs font-bold uppercase border transition ${
                            paymentMethod === m
                              ? "bg-bw-orange text-white border-bw-orange"
                              : "bg-surface-container-low border-surface-container-high text-on-surface"
                          }">${METHOD_LABELS[m] || m}</button>
                        `).join("")}
                      </div>
                    </div>

                    <!-- Monto recibido / Referencia -->
                    ${paymentMethod === "CASH" ? `
                      <div class="space-y-2">
                        <label class="block text-xs font-bold uppercase text-secondary">Efectivo Recibido ($)</label>
                        <input id="input-received" type="text" inputmode="decimal" autocomplete="off" placeholder="${total.toFixed(2)}" value="${amountReceived}"
                          class="w-full px-3.5 py-2.5 rounded-xl bg-surface-container-low border border-surface-container-high text-lg font-bold" />
                        <div class="flex flex-wrap gap-1.5">
                          <button data-quick-cash="${total.toFixed(2)}" class="btn-quick-cash px-3 py-1 rounded-lg bg-surface-container text-xs font-bold hover:bg-surface-container-high">Exacto (${money(total)})</button>
                          <button data-quick-cash="${Math.ceil(total / 5) * 5}" class="btn-quick-cash px-3 py-1 rounded-lg bg-surface-container text-xs font-bold hover:bg-surface-container-high">${money(Math.ceil(total / 5) * 5)}</button>
                          <button data-quick-cash="${Math.ceil(total / 10) * 10}" class="btn-quick-cash px-3 py-1 rounded-lg bg-surface-container text-xs font-bold hover:bg-surface-container-high">${money(Math.ceil(total / 10) * 10)}</button>
                          <button data-quick-cash="100" class="btn-quick-cash px-3 py-1 rounded-lg bg-surface-container text-xs font-bold hover:bg-surface-container-high">$100.00</button>
                        </div>
                        <div class="p-3 rounded-xl flex items-center justify-between text-sm font-bold ${
                          amountReceived !== "" && change < 0 ? "bg-red-50 text-error" : "bg-surface-container-low text-bw-ink"
                        }">
                          <span>${change < 0 && amountReceived !== "" ? "Falta por cubrir" : "Vuelto estimado"}</span>
                          <span>${amountReceived === "" ? "$0.00" : money(Math.abs(change))}</span>
                        </div>
                      </div>
                    ` : `
                      ${paymentMethod === "QR" ? `
                        <div class="p-4 rounded-2xl bg-surface-container-low border border-surface-container-high flex flex-col items-center text-center space-y-2.5">
                          <span class="text-[10px] font-bold uppercase tracking-widest text-bw-orange">Transfer365 / QuickPay · Escanea para pagar</span>
                          <div class="p-2.5 bg-white rounded-xl border border-surface-container-high shadow-xs">
                            <img
                              src="https://api.qrserver.com/v1/create-qr-code/?size=180x180&margin=6&data=${encodeURIComponent(`BUFFALO WINGS SV | MESA ${precheck.table_code} | COMANDA #${precheck.order_id} | TOTAL $${total.toFixed(2)}`)}"
                              alt="Código QR de pago"
                              class="w-44 h-44 rounded-lg"
                            />
                          </div>
                          <div class="text-xs font-bold text-bw-ink">
                            Mesa ${precheck.table_code} · Comanda #${precheck.order_id} · <span class="text-bw-orange">${money(total)}</span>
                          </div>
                          <p class="text-[11px] text-secondary">Pide al cliente escanear el código y confirma el comprobante abajo.</p>
                        </div>
                      ` : ""}

                      <div class="space-y-2">
                        <div class="flex items-center justify-between">
                          <label class="block text-xs font-bold uppercase text-secondary">Referencia / Autorización (${METHOD_LABELS[paymentMethod]})</label>
                          ${paymentMethod === "QR" ? `
                            <button id="btn-sim-qr-ref" type="button" class="text-[11px] font-bold text-bw-orange hover:underline uppercase">
                              Simular Ref.
                            </button>
                          ` : ""}
                        </div>
                        <input id="input-reference" type="text" placeholder="${paymentMethod === "QR" ? "Ej: QR-849201" : "Ej: AUTH-482910 o últimos 4 dígitos"}" value="${reference}"
                          class="w-full px-3.5 py-2.5 rounded-xl bg-surface-container-low border border-surface-container-high text-sm font-medium" />
                        <p class="text-[11px] text-secondary">Se cobrará el monto exacto de <strong>${money(total)}</strong>.</p>
                      </div>
                    `}

                    <button id="btn-submit-pay" class="w-full py-3.5 rounded-xl bg-bw-forest hover:opacity-90 text-white font-display text-lg uppercase tracking-wider shadow">
                      Cobrar ${money(total)} y Emitir DTE
                    </button>
                  `}
                </div>
              ` : ""}
            </div>
          </div>
        `}
      </div>
    `;

    // Eventos
    container.querySelector("#close-bill-fb")?.addEventListener("click", () => {
      feedback = null;
      render();
    });

    container.querySelector("#btn-reload-billing")?.addEventListener("click", () => {
      feedback = null;
      init();
    });

    container.querySelector("#btn-lookup-receipt")?.addEventListener("click", async () => {
      const val = Number(container.querySelector("#input-lookup-receipt")?.value);
      if (!val || val <= 0) return;
      try {
        receipt = await api.getReceipt(val);
        feedback = { type: "ok", text: `Recibo de Comanda #${val} cargado.` };
        render();
      } catch (err) {
        feedback = { type: "err", text: err.message };
        render();
      }
    });

    container.querySelector("#btn-print-receipt")?.addEventListener("click", () => window.print());
    container.querySelector("#btn-close-receipt")?.addEventListener("click", () => {
      receipt = null;
      render();
    });
    container.querySelector("#btn-go-tables")?.addEventListener("click", () => {
      if (onGoToTables) onGoToTables();
    });

    container.querySelectorAll(".btn-pick-account").forEach((btn) => {
      btn.addEventListener("click", async () => {
        selectedOrderId = Number(btn.dataset.pickOrder);
        feedback = null;
        await loadPrecheck(selectedOrderId);
        render();
      });
    });

    container.querySelectorAll(".btn-tip-mode").forEach((btn) => {
      btn.addEventListener("click", () => {
        tipMode = btn.dataset.tip;
        if (tipMode === "SUGGESTED" && precheck) customTip = precheck.suggested_tip_amount;
        render();
      });
    });

    container.querySelector("#input-custom-tip")?.addEventListener("input", (e) => {
      customTip = e.target.value;
      render("#input-custom-tip");
    });

    container.querySelectorAll(".btn-pay-method").forEach((btn) => {
      btn.addEventListener("click", () => {
        paymentMethod = btn.dataset.method;
        render();
      });
    });

    container.querySelector("#input-received")?.addEventListener("input", (e) => {
      amountReceived = e.target.value;
      const input = e.target;
      const start = input.selectionStart;
      const end = input.selectionEnd;
      render();
      const restored = container.querySelector("#input-received");
      if (restored) {
        restored.focus();
        restored.setSelectionRange(start, end);
      }
    });

    container.querySelectorAll(".btn-quick-cash").forEach((btn) => {
      btn.addEventListener("click", () => {
        amountReceived = String(btn.dataset.quickCash);
        render();
      });
    });

    container.querySelector("#btn-sim-qr-ref")?.addEventListener("click", () => {
      reference = `QR-${Math.floor(100000 + Math.random() * 900000)}`;
      const input = container.querySelector("#input-reference");
      if (input) input.value = reference;
    });

    container.querySelector("#btn-submit-pay")?.addEventListener("click", async () => {
      if (!precheck) return;
      const finalTip = currentTip();
      const finalTotal = currentTotal();
      const finalReceived = paymentMethod === "CASH" ? Number(amountReceived || 0) : finalTotal;

      if (paymentMethod === "CASH" && finalReceived < finalTotal) {
        feedback = { type: "err", text: `El efectivo recibido (${money(finalReceived)}) es menor al total (${money(finalTotal)}).` };
        render();
        return;
      }

      const btn = container.querySelector("#btn-submit-pay");
      btn.disabled = true;
      btn.textContent = "Procesando cobro...";

      try {
        receipt = await api.payOrder(precheck.order_id, {
          payment_method: paymentMethod,
          tip_amount: finalTip,
          amount_received: finalReceived,
          reference: paymentMethod === "CASH" ? "" : reference.trim(),
        });
        feedback = {
          type: "ok",
          text: `Comanda #${receipt.order_id} (Mesa ${receipt.table_code}) cobrada. DTE: ${receipt.dte_code}. La mesa pasó a "Por limpiar".`,
        };
        selectedOrderId = null;
        await init();
      } catch (err) {
        feedback = { type: "err", text: err.message };
        render();
      }
    });

    if (focusSelector) container.querySelector(focusSelector)?.focus({ preventScroll: true });
  }

  await init();
}
