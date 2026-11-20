// Package ratelimit implementa un límite de peticiones por clave (ej. IP) en memoria.
package ratelimit

import (
	"sync"
	"time"
)

// cleanupEvery indica cada cuántas llamadas a Allow se borran ventanas vencidas,
// para que el mapa no crezca sin límite si llegan muchas IPs distintas.
const cleanupEvery = 1000

// window guarda cuántas peticiones lleva una clave en la ventana actual.
type window struct {
	start time.Time
	count int
}

// Limiter aplica una "ventana fija": cada clave puede hacer como máximo `limit`
// peticiones por cada `period`. Al terminar el periodo, el contador vuelve a cero.
//
// Es seguro para usarse desde varias goroutines (cada petición HTTP corre en la
// suya), por eso el mapa está protegido con un Mutex.
type Limiter struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	windows map[string]*window
	calls   int
	now     func() time.Time
}

// New crea un limitador de `limit` peticiones por `period`.
func New(limit int, period time.Duration) *Limiter {
	return &Limiter{
		limit:   limit,
		period:  period,
		windows: make(map[string]*window),
		now:     time.Now,
	}
}

// Allow registra una petición de `key` y dice si se permite.
// Si no se permite, retryAfter indica cuánto falta para que se reinicie la ventana.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	l.calls++
	if l.calls >= cleanupEvery {
		l.calls = 0
		l.cleanup(now)
	}

	w, exists := l.windows[key]
	if !exists || now.Sub(w.start) >= l.period {
		l.windows[key] = &window{start: now, count: 1}
		return true, 0
	}

	if w.count >= l.limit {
		return false, w.start.Add(l.period).Sub(now)
	}

	w.count++
	return true, 0
}

// cleanup borra las ventanas que ya vencieron. Se llama con el Mutex tomado.
func (l *Limiter) cleanup(now time.Time) {
	for key, w := range l.windows {
		if now.Sub(w.start) >= l.period {
			delete(l.windows, key)
		}
	}
}
