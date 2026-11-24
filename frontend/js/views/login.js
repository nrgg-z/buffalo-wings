import { api, setSession } from "../api.js";

const DEV_USERS = [
  { code: "EMP-01", label: "EMP-01 · Gabriela T. (Gerente)" },
  { code: "EMP-02", label: "EMP-02 · Andrea M. (Hostess)" },
  { code: "EMP-03", label: "EMP-03 · Carlos R. (Mesero)" },
  { code: "EMP-04", label: "EMP-04 · Roberto G. (Cocina)" },
  { code: "EMP-05", label: "EMP-05 · Luis F. (Cajero)" },
];

export function renderLogin(container, { onSuccess, notice = "" }) {
  let pin = "";
  let loading = false;

  container.innerHTML = `
    <div class="min-h-screen flex items-center justify-center p-4 bg-surface">
      <div class="w-full max-w-sm bg-surface-container-lowest rounded-2xl shadow-lg border border-surface-container-high p-5 sm:p-6">
        <div class="text-center mb-6">
          <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-bw-orange text-white font-display text-2xl mb-3 shadow-sm">BW</div>
          <h1 class="font-display text-3xl tracking-wide text-bw-ink uppercase">Buffalo Wings</h1>
          <p class="text-xs font-semibold tracking-widest text-secondary uppercase mt-1">El Salvador · POS & KDS</p>
        </div>

        ${notice ? `<div class="mb-4 p-3 rounded-xl bg-amber-50 border border-amber-200 text-amber-800 text-xs font-medium">${notice}</div>` : ""}
        <div id="login-error" class="hidden mb-4 p-3 rounded-xl bg-red-50 border border-red-200 text-error text-xs font-semibold"></div>

        <label class="block text-xs font-bold uppercase tracking-wider text-secondary mb-1.5">Código de Empleado</label>
        <div class="flex gap-2 mb-4">
          <input id="emp-code" type="text" placeholder="Ej. EMP-03"
            class="w-1/3 px-3 py-2.5 rounded-xl bg-surface-container-low border border-surface-container-high font-semibold text-sm uppercase focus:outline-none focus:ring-2 focus:ring-bw-orange" />
          <select id="emp-quick"
            class="w-2/3 px-3 py-2.5 rounded-xl bg-surface-container-low border border-surface-container-high text-xs font-medium focus:outline-none focus:ring-2 focus:ring-bw-orange">
            <option value="" selected>Seleccionar empleado...</option>
            ${DEV_USERS.map((u) => `<option value="${u.code}">${u.label}</option>`).join("")}
          </select>
        </div>

        <label class="block text-xs font-bold uppercase tracking-wider text-secondary mb-2">PIN de Acceso (4 dígitos)</label>
        <div id="pin-dots" class="grid grid-cols-4 gap-3 mb-5">
          ${[0, 1, 2, 3].map(() => `<div class="pin-dot h-12 rounded-xl bg-surface-container-low border-2 border-surface-container-high flex items-center justify-center text-2xl font-bold text-bw-ink"></div>`).join("")}
        </div>

        <div class="grid grid-cols-3 gap-2.5 mb-4">
          ${[1, 2, 3, 4, 5, 6, 7, 8, 9].map((n) => `
            <button type="button" data-num="${n}" class="keypad-btn h-12 rounded-xl bg-surface-container-low hover:bg-surface-container font-bold text-xl text-bw-ink active:scale-95 transition">${n}</button>
          `).join("")}
          <button type="button" id="pin-clear" class="h-12 rounded-xl bg-surface-container text-secondary font-semibold text-xs uppercase hover:bg-surface-container-high active:scale-95 transition">Limpiar</button>
          <button type="button" data-num="0" class="keypad-btn h-12 rounded-xl bg-surface-container-low hover:bg-surface-container font-bold text-xl text-bw-ink active:scale-95 transition">0</button>
          <button type="button" id="pin-back" class="h-12 rounded-xl bg-surface-container text-secondary flex items-center justify-center hover:bg-surface-container-high active:scale-95 transition">
            <span class="material-symbols-outlined">backspace</span>
          </button>
        </div>

        <button id="btn-login" type="button"
          class="w-full h-12 rounded-xl bg-bw-orange hover:bg-bw-orange-deep text-white font-bold uppercase tracking-wider text-sm shadow-sm disabled:opacity-50 transition">
          Iniciar Turno
        </button>

      </div>
    </div>
  `;

  const codeInput = container.querySelector("#emp-code");
  const quickSelect = container.querySelector("#emp-quick");
  const dots = container.querySelectorAll(".pin-dot");
  const errorBox = container.querySelector("#login-error");
  const btnLogin = container.querySelector("#btn-login");

  quickSelect.addEventListener("change", () => {
    codeInput.value = quickSelect.value;
    pin = "";
    renderDots();
  });

  function showError(msg) { errorBox.textContent = msg; errorBox.classList.remove("hidden"); }
  function clearError() { errorBox.classList.add("hidden"); }

  function renderDots() {
    dots.forEach((d, idx) => {
      const filled = idx < pin.length;
      d.textContent = filled ? "●" : "";
      d.classList.toggle("border-bw-orange", filled);
      d.classList.toggle("border-surface-container-high", !filled);
    });
  }

  async function submit() {
    if (loading) return;
    const employeeCode = codeInput.value.trim().toUpperCase();
    if (!employeeCode || pin.length !== 4) {
      showError("Ingresa tu código de empleado y los 4 dígitos de tu PIN.");
      return;
    }
    clearError();
    loading = true;
    btnLogin.disabled = true;
    btnLogin.textContent = "Verificando...";

    try {
      const res = await api.login(employeeCode, pin);
      setSession({ token: res.token, user: res.user });
      window.removeEventListener("keydown", onKeyDown);
      onSuccess(res.user);
    } catch (err) {
      pin = "";
      renderDots();
      if (err.status === 401) showError("Código o PIN incorrecto.");
      else if (err.status === 423) showError("Usuario bloqueado 5 minutos por 3 intentos fallidos.");
      else if (err.status === 403) showError("Este empleado está desactivado.");
      else if (err.status === 429) showError(`Demasiados intentos. Espera ${err.retryAfter || 60}s.`);
      else showError(err.message || "Error al iniciar sesión.");
    } finally {
      loading = false;
      btnLogin.disabled = false;
      btnLogin.textContent = "Iniciar Turno";
    }
  }

  function appendDigit(d) {
    if (pin.length >= 4 || loading) return;
    clearError();
    pin += d;
    renderDots();
    if (pin.length === 4) submit();
  }

  container.querySelectorAll(".keypad-btn").forEach((btn) => btn.addEventListener("click", () => appendDigit(btn.dataset.num)));
  container.querySelector("#pin-clear").addEventListener("click", () => { pin = ""; clearError(); renderDots(); });
  container.querySelector("#pin-back").addEventListener("click", () => { pin = pin.slice(0, -1); renderDots(); });
  btnLogin.addEventListener("click", submit);

  function onKeyDown(e) {
    if (document.activeElement === codeInput) return;
    if (/^[0-9]$/.test(e.key)) appendDigit(e.key);
    else if (e.key === "Backspace") { pin = pin.slice(0, -1); renderDots(); }
    else if (e.key === "Enter") submit();
  }
  window.addEventListener("keydown", onKeyDown);
}
