import { api } from "../api.js";

const money = (n) => `$${Number(n || 0).toFixed(2)}`;

export async function renderCatalog(container, { user }) {
  let categories = [];
  let selectedCatId = "ALL";
  let feedback = null;

  const canManage = ["KITCHEN", "MANAGER", "ADMIN"].includes(user.role);

  async function load(keepFeedback = false) {
    if (!keepFeedback) feedback = null;
    container.innerHTML = `<div class="p-8 text-center text-sm text-secondary">Cargando catálogo...</div>`;
    try {
      const res = await api.getCatalog();
      categories = res.categories || [];
      render();
    } catch (err) {
      container.innerHTML = `<div class="bg-red-50 border border-red-200 text-error rounded-2xl p-6 text-sm font-medium">${err.message}</div>`;
    }
  }

  function render() {
    const visibleCats = selectedCatId === "ALL"
      ? categories
      : categories.filter((c) => c.id === Number(selectedCatId));

    container.innerHTML = `
      <div class="space-y-5">
        <div class="bg-surface-container-lowest rounded-2xl border border-surface-container-high p-5 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-widest text-secondary">Menú y Control de Agotados (86)</div>
            <h1 class="font-display text-2xl uppercase text-bw-ink tracking-wide mt-0.5">Catálogo de Productos y Salsas</h1>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-xs font-semibold px-3 py-1.5 rounded-full ${
              canManage ? "bg-emerald-50 text-bw-forest border border-emerald-200" : "bg-surface-container-low text-secondary"
            }">
              ${canManage ? "Modo edición activo" : "Solo lectura"}
            </span>
            <button id="btn-reload-cat" class="px-3.5 py-2 rounded-xl text-xs font-bold uppercase bg-surface-container-lowest border border-surface-container-high hover:bg-surface-container-low flex items-center gap-1.5">
              <span class="material-symbols-outlined text-base">refresh</span> Actualizar
            </button>
          </div>
        </div>

        ${feedback ? `
          <div class="p-3.5 rounded-xl border text-xs font-semibold flex items-center justify-between ${
            feedback.type === "ok" ? "bg-emerald-50 border-emerald-200 text-bw-forest" : "bg-red-50 border-red-200 text-error"
          }">
            <span>${feedback.text}</span>
            <button id="close-cat-fb" class="ml-4 opacity-70 hover:opacity-100">✕</button>
          </div>
        ` : ""}

        <div class="flex flex-wrap gap-1.5">
          <button data-cat="ALL" class="cat-tab px-3.5 py-1.5 rounded-xl text-xs font-bold uppercase ${
            selectedCatId === "ALL" ? "bg-bw-ink text-white" : "bg-surface-container-lowest border border-surface-container-high text-secondary"
          }">Todas</button>
          ${categories.map((c) => `
            <button data-cat="${c.id}" class="cat-tab px-3.5 py-1.5 rounded-xl text-xs font-bold uppercase ${
              String(selectedCatId) === String(c.id) ? "bg-bw-ink text-white" : "bg-surface-container-lowest border border-surface-container-high text-secondary"
            }">${c.name} <span class="opacity-60">(${c.station})</span></button>
          `).join("")}
        </div>

        <div class="space-y-6">
          ${visibleCats.map((cat) => `
            <section class="space-y-3">
              <div class="flex items-center gap-2">
                <h2 class="font-display text-xl uppercase text-bw-ink">${cat.name}</h2>
                <span class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-surface-container text-secondary uppercase">${cat.station}</span>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                ${(cat.products || []).map((p) => `
                  <div class="bg-surface-container-lowest rounded-2xl border p-4 shadow-sm space-y-3 ${
                    p.is_available ? "border-surface-container-high" : "border-red-200 bg-red-50/30"
                  }">
                    <div class="flex items-start justify-between gap-2">
                      <div>
                        <div class="font-bold text-sm text-on-surface">${p.name}</div>
                        <div class="text-xs font-bold text-bw-orange mt-0.5">${money(p.base_price)}</div>
                      </div>
                      ${canManage ? `
                        <button data-toggle-prod="${p.id}" data-next="${!p.is_available}" class="btn-toggle-prod px-2.5 py-1 rounded-lg text-[11px] font-bold uppercase border transition ${
                          p.is_available
                            ? "bg-emerald-50 text-bw-forest border-emerald-200 hover:bg-red-50 hover:text-error hover:border-red-200"
                            : "bg-red-100 text-error border-red-200 hover:bg-emerald-50 hover:text-bw-forest"
                        }">
                          ${p.is_available ? "Disponible" : "Agotado"}
                        </button>
                      ` : `
                        <span class="px-2.5 py-1 rounded-lg text-[11px] font-bold uppercase ${
                          p.is_available ? "bg-emerald-50 text-bw-forest" : "bg-red-100 text-error"
                        }">${p.is_available ? "Disponible" : "Agotado"}</span>
                      `}
                    </div>

                    ${(p.modifier_groups || []).length ? `
                      <div class="pt-2 border-t border-surface-container-high space-y-2">
                        ${p.modifier_groups.map((g) => `
                          <div>
                            <div class="text-[10px] font-bold uppercase text-secondary mb-1">
                              ${g.name} ${g.is_required ? "· Obligatorio" : ""} (${g.min_selections}-${g.max_selections})
                            </div>
                            <div class="flex flex-wrap gap-1.5">
                              ${(g.options || []).map((o) => canManage ? `
                                <button data-toggle-mod="${o.id}" data-next="${!o.is_available}" class="btn-toggle-mod px-2 py-0.5 rounded-md text-[11px] font-semibold border transition ${
                                  o.is_available
                                    ? "bg-surface-container-low border-surface-container-high text-on-surface hover:border-error"
                                    : "bg-red-50 border-red-200 text-error line-through"
                                }">
                                  ${o.name}${o.extra_price > 0 ? ` (+${money(o.extra_price)})` : ""}
                                </button>
                              ` : `
                                <span class="px-2 py-0.5 rounded-md text-[11px] font-semibold ${
                                  o.is_available ? "bg-surface-container-low text-on-surface" : "bg-red-50 text-error line-through"
                                }">
                                  ${o.name}${o.extra_price > 0 ? ` (+${money(o.extra_price)})` : ""}
                                </span>
                              `).join("")}
                            </div>
                          </div>
                        `).join("")}
                      </div>
                    ` : ""}
                  </div>
                `).join("")}
              </div>
            </section>
          `).join("")}
        </div>
      </div>
    `;

    container.querySelector("#close-cat-fb")?.addEventListener("click", () => {
      feedback = null;
      render();
    });
    container.querySelector("#btn-reload-cat")?.addEventListener("click", () => load(false));
    container.querySelectorAll(".cat-tab").forEach((btn) => {
      btn.addEventListener("click", () => {
        selectedCatId = btn.dataset.cat;
        render();
      });
    });

    container.querySelectorAll(".btn-toggle-prod").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const id = Number(btn.dataset.toggleProd);
        const next = btn.dataset.next === "true";
        try {
          const updated = await api.setProductAvailability(id, next);
          feedback = { type: "ok", text: `${updated.name}: marcado como ${updated.is_available ? "DISPONIBLE" : "AGOTADO"}.` };
          await load(true);
        } catch (err) {
          feedback = { type: "err", text: err.message };
          render();
        }
      });
    });

    container.querySelectorAll(".btn-toggle-mod").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const id = Number(btn.dataset.toggleMod);
        const next = btn.dataset.next === "true";
        try {
          const updated = await api.setModifierAvailability(id, next);
          feedback = { type: "ok", text: `Modificador "${updated.name}": marcado como ${updated.is_available ? "DISPONIBLE" : "AGOTADO"}.` };
          await load(true);
        } catch (err) {
          feedback = { type: "err", text: err.message };
          render();
        }
      });
    });
  }

  await load(false);
}