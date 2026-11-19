package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORSOrigenPermitidoRecibeHeaders(t *testing.T) {
	h := CORS([]string{"http://localhost:5173"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Allow-Origin esperado http://localhost:5173, llegó %q", got)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("código esperado 200, llegó %d", rec.Code)
	}
}

func TestCORSOrigenNoPermitidoNoRecibeHeaders(t *testing.T) {
	h := CORS([]string{"http://localhost:5173"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "http://malicioso.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no debía haber Allow-Origin, llegó %q", got)
	}
}

func TestCORSPreflightResponde204(t *testing.T) {
	h := CORS([]string{"http://localhost:5173"})(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight esperado 204, llegó %d", rec.Code)
	}

	// El frontend necesita estas cabeceras y métodos; si alguien los quita, esta prueba falla.
	headers := rec.Header().Get("Access-Control-Allow-Headers")
	for _, want := range []string{"Authorization", "Content-Type"} {
		if !strings.Contains(headers, want) {
			t.Errorf("Allow-Headers debía incluir %s, llegó %q", want, headers)
		}
	}

	methods := rec.Header().Get("Access-Control-Allow-Methods")
	for _, want := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(methods, want) {
			t.Errorf("Allow-Methods debía incluir %s, llegó %q", want, methods)
		}
	}
}

func TestClientIPQuitaElPuerto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.20:54321"
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // debe ignorarse

	if got := ClientIP(req); got != "192.168.1.20" {
		t.Errorf("IP esperada 192.168.1.20, llegó %q", got)
	}
}
