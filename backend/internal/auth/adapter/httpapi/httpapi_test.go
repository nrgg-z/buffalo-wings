package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/service"
)

// fakeAuth imita el servicio: acepta solo los tokens que tiene en su mapa.
type fakeAuth struct {
	tokens   map[string]domain.Principal
	users    map[int64]*domain.User
	loginErr error
	lastIP   string
}

func (f *fakeAuth) VerifyToken(token string) (domain.Principal, error) {
	p, ok := f.tokens[token]
	if !ok {
		return domain.Principal{}, domain.ErrInvalidToken
	}
	return p, nil
}

func (f *fakeAuth) Login(ctx context.Context, code, pin, ip string) (*service.LoginResult, error) {
	f.lastIP = ip
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return &service.LoginResult{Token: "tok", UserID: 3, EmployeeCode: code, Role: domain.RoleWaiter}, nil
}

func (f *fakeAuth) CurrentUser(ctx context.Context, id int64) (*domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, domain.ErrInvalidToken
	}
	if !u.Active {
		return nil, domain.ErrUserInactive
	}
	return u, nil
}

var expiresAt = time.Date(2026, 11, 18, 21, 0, 0, 0, time.UTC)

func newFakeAuth() *fakeAuth {
	return &fakeAuth{
		tokens: map[string]domain.Principal{
			"token-mesero": {UserID: 3, EmployeeCode: "EMP-03", Role: domain.RoleWaiter, ExpiresAt: expiresAt},
			"token-admin":  {UserID: 9, EmployeeCode: "ADM-01", Role: domain.RoleAdmin, ExpiresAt: expiresAt},
			"token-cocina": {UserID: 4, EmployeeCode: "EMP-04", Role: domain.RoleKitchen, ExpiresAt: expiresAt},
		},
		users: map[int64]*domain.User{
			3: {ID: 3, EmployeeCode: "EMP-03", FullName: "Carlos R. (Mesero)", Role: domain.RoleWaiter, Active: true},
			4: {ID: 4, EmployeeCode: "EMP-04", FullName: "Roberto G. (Cocina)", Role: domain.RoleKitchen, Active: false},
		},
	}
}

// newMux arma el router como en main.go, más una ruta de ejemplo solo para WAITER
// que existe únicamente en esta prueba (para probar RequireRoles).
func newMux(auth *fakeAuth) *http.ServeMux {
	mux := http.NewServeMux()
	NewAuthHandler(auth).Register(mux, nil)

	mux.Handle("GET /test/solo-meseros", Protected(auth, domain.RoleWaiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, _ := PrincipalFrom(r.Context())
			w.Write([]byte(p.EmployeeCode))
		})))
	return mux
}

func do(mux http.Handler, method, path, authHeader, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.RemoteAddr = "10.1.2.3:4567"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta no es JSON: %v", err)
	}
	return body.Error
}

// ---------- RequireAuth ----------

func TestRequireAuthRechazaPeticionesSinTokenValido(t *testing.T) {
	mux := newMux(newFakeAuth())

	casos := map[string]struct {
		header string
		code   string
	}{
		"sin header":        {"", "unauthorized"},
		"esquema Basic":     {"Basic dXNlcjpwYXNz", "unauthorized"},
		"Bearer sin token":  {"Bearer ", "unauthorized"},
		"token desconocido": {"Bearer token-falso", "invalid_token"},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			rec := do(mux, http.MethodGet, "/api/v1/auth/me", c.header, "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("esperaba 401, llegó %d", rec.Code)
			}
			if got := errorCode(t, rec); got != c.code {
				t.Errorf("código de error esperado %s, llegó %s", c.code, got)
			}
		})
	}
}

func TestRequireAuthAceptaBearerSinImportarMayusculas(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/test/solo-meseros", "bearer token-mesero", "")
	if rec.Code != http.StatusOK {
		t.Errorf("esperaba 200, llegó %d", rec.Code)
	}
}

// ---------- RequireRoles ----------

func TestRequireRolesDejaPasarRolCorrectoConPrincipal(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/test/solo-meseros", "Bearer token-mesero", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, llegó %d", rec.Code)
	}
	if rec.Body.String() != "EMP-03" {
		t.Errorf("el handler debía ver al principal EMP-03, vio %q", rec.Body.String())
	}
}

func TestRequireRolesRechazaOtroRolCon403(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/test/solo-meseros", "Bearer token-cocina", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperaba 403, llegó %d", rec.Code)
	}
	if got := errorCode(t, rec); got != "forbidden" {
		t.Errorf("código esperado forbidden, llegó %s", got)
	}
}

func TestRequireRolesAdminSiemprePasa(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/test/solo-meseros", "Bearer token-admin", "")
	if rec.Code != http.StatusOK {
		t.Errorf("ADMIN debía pasar, llegó %d", rec.Code)
	}
}

func TestRequireRolesSinRequireAuthDa401(t *testing.T) {
	h := RequireRoles(domain.RoleWaiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no debía llegar al handler")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperaba 401, llegó %d", rec.Code)
	}
}

// ---------- GET /auth/me ----------

func TestMeDevuelveUsuarioYExpiracion(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/api/v1/auth/me", "Bearer token-mesero", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, llegó %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("la respuesta debía llevar Cache-Control: no-store")
	}

	var body meResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.User.FullName != "Carlos R. (Mesero)" || body.User.Role != "WAITER" {
		t.Errorf("usuario inesperado: %+v", body.User)
	}
	if !body.TokenExpiresAt.Equal(expiresAt) {
		t.Errorf("token_expires_at esperado %v, llegó %v", expiresAt, body.TokenExpiresAt)
	}
}

func TestMeUsuarioInactivoDa403(t *testing.T) {
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/api/v1/auth/me", "Bearer token-cocina", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("esperaba 403, llegó %d", rec.Code)
	}
}

func TestMeUsuarioBorradoDa401(t *testing.T) {
	// token-admin es válido pero su usuario (id 9) no existe en la "BD".
	rec := do(newMux(newFakeAuth()), http.MethodGet, "/api/v1/auth/me", "Bearer token-admin", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperaba 401, llegó %d", rec.Code)
	}
}

// ---------- POST /auth/login ----------

func TestLoginPasaLaIPDelClienteAlServicio(t *testing.T) {
	auth := newFakeAuth()
	rec := do(newMux(auth), http.MethodPost, "/api/v1/auth/login", "", `{"employee_code":"EMP-03","pin":"1234"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, llegó %d", rec.Code)
	}
	if auth.lastIP != "10.1.2.3" {
		t.Errorf("IP esperada 10.1.2.3, llegó %q", auth.lastIP)
	}
}

func TestLoginMapeaErroresACodigosHTTP(t *testing.T) {
	casos := map[string]struct {
		err    error
		status int
	}{
		"credenciales": {domain.ErrInvalidCredentials, http.StatusUnauthorized},
		"bloqueado":    {domain.ErrUserLocked, http.StatusLocked},
		"inactivo":     {domain.ErrUserInactive, http.StatusForbidden},
		"interno":      {errors.New("bitácora caída"), http.StatusInternalServerError},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			auth := newFakeAuth()
			auth.loginErr = c.err
			rec := do(newMux(auth), http.MethodPost, "/api/v1/auth/login", "", `{"employee_code":"EMP-03","pin":"1234"}`)
			if rec.Code != c.status {
				t.Errorf("esperaba %d, llegó %d", c.status, rec.Code)
			}
			if strings.Contains(rec.Body.String(), "bitácora caída") {
				t.Error("el detalle del error interno no debe llegar al cliente")
			}
		})
	}
}

func TestLoginRechazaCuerpoInvalido(t *testing.T) {
	mux := newMux(newFakeAuth())
	for _, body := range []string{`no-json`, `{"employee_code":"EMP-03"}`, `{"employee_code":"EMP-03","pin":"1234","extra":1}`} {
		rec := do(mux, http.MethodPost, "/api/v1/auth/login", "", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("cuerpo %s: esperaba 400, llegó %d", body, rec.Code)
		}
	}
}
