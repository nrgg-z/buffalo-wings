package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeClock permite mover el tiempo a mano en las pruebas.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLimiter(limit int) (*Limiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)}
	l := New(limit, time.Minute)
	l.now = clock.now
	return l, clock
}

func TestPermiteHastaElLimiteYBloqueaElSiguiente(t *testing.T) {
	l, _ := newTestLimiter(15)

	for i := 1; i <= 15; i++ {
		if ok, _ := l.Allow("10.0.0.1"); !ok {
			t.Fatalf("la petición %d debía permitirse", i)
		}
	}
	ok, retry := l.Allow("10.0.0.1")
	if ok {
		t.Fatal("la petición 16 debía bloquearse")
	}
	if retry != time.Minute {
		t.Errorf("retryAfter esperado 1m, llegó %v", retry)
	}
}

func TestLaVentanaSeReiniciaAlPasarElPeriodo(t *testing.T) {
	l, clock := newTestLimiter(2)

	l.Allow("10.0.0.1")
	l.Allow("10.0.0.1")
	if ok, _ := l.Allow("10.0.0.1"); ok {
		t.Fatal("la tercera debía bloquearse")
	}

	clock.t = clock.t.Add(time.Minute)

	if ok, _ := l.Allow("10.0.0.1"); !ok {
		t.Error("pasado un minuto debía permitirse otra vez")
	}
}

func TestIPsDistintasSonIndependientes(t *testing.T) {
	l, _ := newTestLimiter(1)

	l.Allow("10.0.0.1")
	if ok, _ := l.Allow("10.0.0.2"); !ok {
		t.Error("otra IP no debía verse afectada")
	}
}

func TestRetryAfterDisminuyeConElTiempo(t *testing.T) {
	l, clock := newTestLimiter(1)

	l.Allow("10.0.0.1")
	clock.t = clock.t.Add(40 * time.Second)

	_, retry := l.Allow("10.0.0.1")
	if retry != 20*time.Second {
		t.Errorf("retryAfter esperado 20s, llegó %v", retry)
	}
}

func TestMiddlewareResponde429ConRetryAfter(t *testing.T) {
	l, _ := newTestLimiter(1)
	h := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	hacer := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:5000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := hacer(); rec.Code != http.StatusOK {
		t.Fatalf("la primera debía pasar, llegó %d", rec.Code)
	}
	rec := hacer()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("la segunda debía dar 429, llegó %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After esperado 60, llegó %q", rec.Header().Get("Retry-After"))
	}
}
